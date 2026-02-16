package main_test

import (
	"context"
	"fmt"

	// "os" // Not needed anymore after initProtoFiles refactor
	"net/http"
	"path/filepath"
	"runtime"
	"strings" // Still needed for strings.ReplaceAll, strings.TrimSuffix etc. for artifactID
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"encoding/json"                 // For JSON marshaling/unmarshaling
	"github.com/segmentio/kafka-go" // Import kafka-go

	cmd "pablin.com/apireg/cmd" // Added for direct call to RegisterProtoArtifacts and ParseProtoFiles

	"github.com/mollie/go-apicurio-registry/apis"
	"github.com/mollie/go-apicurio-registry/client"
	"github.com/mollie/go-apicurio-registry/models"
)

// Go structs mirroring the proto definitions
type Message5 struct {
	Id   int32  `json:"id"`
	Name string `json:"name"`
}

type Message6 struct {
	Name       string   `json:"name"`
	NestedMsg5 Message5 `json:"nested_msg5"`
}

// Kafka broker address constant
const kafkaBroker = "localhost:9092"
const kafkaTopic = "test-topic-msg6"

// Global variables for setup
var (
	apiClient        *client.Client
	registryService  *apis.ArtifactsAPI
	metadataService  *apis.MetadataAPI
	importToPath     map[string]string
	fileToImports    map[string][]string
	sortedFiles      []string
	kafkaAdminClient *kafka.Client // New Kafka admin client
)

// initProtoFiles scans the filesystem for proto files and performs a topological sort.
// This function must run at package level to populate sortedFiles before Describe blocks are evaluated.
func initProtoFiles() {
	// Get the directory of the current file (apicurio_test.go)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current file information")
	}
	currentDir := filepath.Dir(filename)

	// Construct protoRoot relative to the test file's directory
	protoRoot := filepath.Join(currentDir, "../proto")

	var err error
	importToPath, fileToImports, sortedFiles, err = cmd.ParseProtoFiles(protoRoot) // Call shared function
	if err != nil {
		panic(fmt.Sprintf("Failed to parse proto files: %v", err))
	}
}

// Execute initProtoFiles at package initialization.
func init() {
	initProtoFiles()
}

// TestApicurio is the entry point for the Ginkgo test suite.
func TestApicurio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Apicurio Suite")
}

var _ = BeforeSuite(func() {
	GinkgoWriter.Printf("\n--- Apicurio Test Suite Setup ---\n")

	// 1. Initialize Apicurio Registry Client with Options
	addr := "http://localhost:8001/apis/registry/v3" // Updated to v3 based on client library version
	apiClient = client.NewClient(addr)
	Expect(apiClient).NotTo(BeNil(), "apiClient failed to initialize")

	registryService = apis.NewArtifactsAPI(apiClient)
	metadataService = apis.NewMetadataAPI(apiClient)

	// Add wait/retry logic for Apicurio Registry to be ready
	GinkgoWriter.Printf("Waiting for Apicurio Registry to be ready at %s/health/ready...\n", "http://localhost:8001")

	maxRetries := 30
	retryInterval := 2 * time.Second
	healthCheckURL := "http://localhost:8001/health/ready"

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(healthCheckURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			GinkgoWriter.Printf("Apicurio Registry is ready!\n")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}

		statusCode := "N/A"
		if resp != nil {
			statusCode = fmt.Sprintf("%d", resp.StatusCode)
		}
		GinkgoWriter.Printf("Attempt %d/%d: Apicurio Registry not ready. Status: %s, Error: %v. Retrying in %v...\n", i+1, maxRetries, statusCode, err, retryInterval)
		time.Sleep(retryInterval)

		if i == maxRetries-1 {
			Fail(fmt.Sprintf("Apicurio Registry did not become ready within %v", time.Duration(maxRetries)*retryInterval))
		}
	}

	// Register protobuf artifacts using cmd.RegisterProtoArtifacts
	GinkgoWriter.Printf("Registering protobuf artifacts using cmd.RegisterProtoArtifacts...\n")
	_, filenameForRegister, _, okForRegister := runtime.Caller(0)
	if !okForRegister {
		Fail("Failed to get current file information for proto scan during registration")
	}
	currentDirForRegister := filepath.Dir(filenameForRegister)
	protoRootForRegister := filepath.Join(currentDirForRegister, "../proto")
	err := cmd.RegisterProtoArtifacts(addr, protoRootForRegister)
	Expect(err).NotTo(HaveOccurred(), "Failed to register artifacts via cmd.RegisterProtoArtifacts")
	GinkgoWriter.Printf("Artifact registration complete.\n")

	GinkgoWriter.Printf("Sleeping for 5 seconds to allow registry to process registrations...\n")
	time.Sleep(5 * time.Second)

	GinkgoWriter.Printf("Found %d proto files for validation, sorted: %v\n", len(sortedFiles), sortedFiles)

	// 2. Initialize Kafka Admin Client
	kafkaAdminClient = &kafka.Client{
		Addr:    kafka.TCP(kafkaBroker),
		Timeout: 10 * time.Second, // Set a timeout for admin operations
	}
	Expect(kafkaAdminClient).NotTo(BeNil(), "kafkaAdminClient failed to initialize")
})

var _ = Describe("Apicurio Registry Proto File Operations", func() {
	Context("Proto file validation", func() {
		// Dynamically create It blocks for validation
		for _, path := range sortedFiles {
			path := path // capture range variable
			It(fmt.Sprintf("should ensure registered proto file is accessible: %s", path), func() {
				// Use the same dynamic protoRoot for Rel
				_, filename, _, _ := runtime.Caller(0)
				testFileDir := filepath.Dir(filename)
				protoRootForRel := filepath.Join(testFileDir, "../proto")

				relPath, _ := filepath.Rel(protoRootForRel, path)
				artifactID := strings.ReplaceAll(strings.TrimSuffix(relPath, ".proto"), string(filepath.Separator), "-")

				// Retry logic for GetArtifactVersionMetadata
				maxRetriesMeta := 10
				retryIntervalMeta := 1 * time.Second
				// First, try to get general artifact metadata (without version)
				var genericGetErr error
				for k := 0; k < maxRetriesMeta; k++ {
					_, genericGetErr = metadataService.GetArtifactMetadata(context.Background(), "default", artifactID)
					if genericGetErr == nil {
						break // Success, exit retry loop
					}
					GinkgoWriter.Printf("Attempt %d/%d: Artifact %s not found (generic metadata). Retrying in %v. Error: %v\n", k+1, maxRetriesMeta, artifactID, retryIntervalMeta, genericGetErr)
					time.Sleep(retryIntervalMeta)
				}
				Expect(genericGetErr).NotTo(HaveOccurred(), "Artifact %s not found in registry (generic metadata) after upload and retries", artifactID)

				// Then, try to get specific artifact version metadata
				var meta *models.ArtifactVersionMetadata
				var getErr error

				for k := 0; k < maxRetriesMeta; k++ {
					meta, getErr = metadataService.GetArtifactVersionMetadata(context.Background(), "default", artifactID, "1")
					if getErr == nil {
						break // Success, exit retry loop
					}
					GinkgoWriter.Printf("Attempt %d/%d: Artifact %s version 1 not found. Retrying in %v. Error: %v\n", k+1, maxRetriesMeta, artifactID, retryIntervalMeta, getErr)
					time.Sleep(retryIntervalMeta)
				}

				Expect(getErr).NotTo(HaveOccurred(), "Artifact %s version 1 not found in registry after upload and retries", artifactID)

				GinkgoWriter.Printf("Validated %s: GlobalId %d\n", artifactID, meta.GlobalID)
			})
		}
	})

	// New Context for Kafka Producer-Consumer Integration
	Context("Kafka Producer-Consumer Integration", func() {
		BeforeEach(func() {
			GinkgoWriter.Printf("Creating Kafka topic: %s\n", kafkaTopic)
			createReq := kafka.CreateTopicsRequest{
				Topics: []kafka.TopicConfig{
					{
						Topic:             kafkaTopic,
						NumPartitions:     1,
						ReplicationFactor: 1,
					},
				},
				ValidateOnly: false,
			}
			// Use the kafka.Client.CreateTopics method
			res, err := kafkaAdminClient.CreateTopics(context.Background(), &createReq)
			if err != nil && !strings.Contains(err.Error(), "topic already exists") { // Check specific error from response
				Fail(fmt.Sprintf("Failed to create Kafka topic %s: %v", kafkaTopic, err))
			}
			// Check response errors if topic creation failed for specific topic
			if res != nil && res.Errors[kafkaTopic] != nil && res.Errors[kafkaTopic].Error() != "topic already exists" {
				Fail(fmt.Sprintf("Failed to create Kafka topic %s with response error: %v", kafkaTopic, res.Errors[kafkaTopic]))
			}
			GinkgoWriter.Printf("Kafka topic %s created (or already exists).\n", kafkaTopic)

			// Give Kafka a moment to settle after topic creation
			time.Sleep(2 * time.Second)
		})

		AfterEach(func() {
			GinkgoWriter.Printf("Deleting Kafka topic: %s\n", kafkaTopic)
			deleteReq := kafka.DeleteTopicsRequest{
				Topics: []string{kafkaTopic},
			}
			// Use the kafka.Client.DeleteTopics method
			res, err := kafkaAdminClient.DeleteTopics(context.Background(), &deleteReq)
			if err != nil {
				GinkgoWriter.Printf("Warning: Failed to delete Kafka topic %s: %v\n", kafkaTopic, err)
			} else {
				if res.Errors[kafkaTopic] != nil {
					GinkgoWriter.Printf("Warning: Failed to delete Kafka topic %s with response error: %v\n", kafkaTopic, res.Errors[kafkaTopic])
				} else {
					GinkgoWriter.Printf("Kafka topic %s deleted.\n", kafkaTopic)
				}
			}
		})

		It("should produce a message using level6/msg6.proto schema and consume it", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// --- Producer Setup ---
			producer := &kafka.Writer{
				Addr:         kafka.TCP(kafkaBroker),
				Topic:        kafkaTopic,
				Balancer:     &kafka.LeastBytes{},
				RequiredAcks: kafka.RequireAll,
			}
			defer producer.Close()

			// Create a message based on proto/level6/msg6.proto schema
			originalMessage := Message6{
				Name: "Test Message 6",
				NestedMsg5: Message5{
					Id:   123,
					Name: "Nested Test Message 5",
				},
			}

			// Marshal the message to JSON bytes
			messageBytes, err := json.Marshal(originalMessage)
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal original message")

			// Produce the message to Kafka
			err = producer.WriteMessages(ctx,
				kafka.Message{
					Key:   []byte("test-key"),
					Value: messageBytes,
				},
			)
			Expect(err).NotTo(HaveOccurred(), "Failed to produce message to Kafka")
			GinkgoWriter.Printf("Produced message to topic %s: %s\n", kafkaTopic, string(messageBytes))

			// --- Consumer Setup ---
			consumer := kafka.NewReader(kafka.ReaderConfig{
				Brokers:     []string{kafkaBroker},
				Topic:       kafkaTopic,
				GroupID:     "test-consumer-group", // Use a consumer group for reliable consumption
				StartOffset: kafka.FirstOffset,     // Start from the beginning of the topic
			})
			defer consumer.Close()

			// Consume the message from Kafka
			GinkgoWriter.Printf("Attempting to consume message from topic %s...\n", kafkaTopic)
			readMessage, err := consumer.ReadMessage(ctx)
			Expect(err).NotTo(HaveOccurred(), "Failed to consume message from Kafka")
			GinkgoWriter.Printf("Consumed message from topic %s: Key=%s, Value=%s\n", kafkaTopic, string(readMessage.Key), string(readMessage.Value))

			// Unmarshal the consumed message
			var consumedMessage Message6
			err = json.Unmarshal(readMessage.Value, &consumedMessage)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal consumed message")

			// Validate the consumed message
			Expect(consumedMessage.Name).To(Equal(originalMessage.Name), "Consumed message name mismatch")
			Expect(consumedMessage.NestedMsg5.Id).To(Equal(originalMessage.NestedMsg5.Id), "Consumed nested message ID mismatch")
			Expect(consumedMessage.NestedMsg5.Name).To(Equal(originalMessage.NestedMsg5.Name), "Consumed nested message name mismatch")

			GinkgoWriter.Printf("Successfully validated consumed message.\n")
		})
	})
})
