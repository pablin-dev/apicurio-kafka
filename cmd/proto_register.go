package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mollie/go-apicurio-registry/apis"
	"github.com/mollie/go-apicurio-registry/client"
	"github.com/mollie/go-apicurio-registry/models"
)

// RegisterProtoArtifacts initializes the client, scans proto files,
// performs a topological sort, and registers them with the Apicurio Registry.
func RegisterProtoArtifacts(addr, protoRoot string) error {
	apiClient := client.NewClient(addr)
	if apiClient == nil {
		return fmt.Errorf("apiClient failed to initialize")
	}

	registryService := apis.NewArtifactsAPI(apiClient)
	metadataService := apis.NewMetadataAPI(apiClient)

	// Use the shared ParseProtoFiles function
	importToPath, fileToImports, sortedFiles, err := ParseProtoFiles(protoRoot) // Call shared function
	if err != nil {
		return fmt.Errorf("failed to parse proto files: %w", err)
	}

	for _, path := range sortedFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read proto file %s: %w", path, err)
		}

		relPath, _ := filepath.Rel(protoRoot, path)
		artifactID := strings.ReplaceAll(strings.TrimSuffix(relPath, ".proto"), string(filepath.Separator), "-")

		var artifactRefs []models.ArtifactReference
		for _, imp := range fileToImports[path] {
			if _, exists := importToPath[imp]; exists {
				refID := strings.ReplaceAll(strings.TrimSuffix(imp, ".proto"), "/", "-")
				artifactRefs = append(artifactRefs, models.ArtifactReference{
					Name:       imp,
					ArtifactID: refID,
					GroupID:    "default",
					Version:    "1",
				})
			}
		}

		createReq := models.CreateArtifactRequest{
			ArtifactID:   artifactID,
			ArtifactType: "PROTOBUF",
			FirstVersion: models.CreateVersionRequest{
				Content: models.CreateContentRequest{
					Content:     string(content),
					ContentType: "application/x-protobuffer",
					References:  artifactRefs,
				},
			},
		}

		_, err = registryService.CreateArtifact(context.Background(), "default", createReq, nil)
		if err != nil && !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("failed to register %s: %w", artifactID, err)
		}
		fmt.Printf("Registered: %s\n", artifactID)

		// Immediately try to retrieve the artifact to verify registration
		maxRetriesVerify := 5
		retryIntervalVerify := 1 * time.Second
		var verifyErr error
		for i := 0; i < maxRetriesVerify; i++ {
			_, verifyErr = metadataService.GetArtifactMetadata(context.Background(), "default", artifactID)
			if verifyErr == nil {
				fmt.Printf("Verified registration of: %s\n", artifactID)
				break
			}
			fmt.Printf("Verification attempt %d/%d for %s failed: %v. Retrying in %v...\n", i+1, maxRetriesVerify, artifactID, verifyErr, retryIntervalVerify)
			time.Sleep(retryIntervalVerify)
		}
		if verifyErr != nil {
			return fmt.Errorf("failed to verify registration of %s after retries: %w", artifactID, verifyErr)
		}
	}

	return nil
}
