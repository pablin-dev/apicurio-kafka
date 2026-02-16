# segmentio/kafka-go Client Library Analysis

This document provides an overview of the packages within the `github.com/segmentio/kafka-go` library and their usage, based solely on code analysis without external documentation or comments.

## 1. Main Package (`kafka`)

The main `kafka` package, residing at the root of the repository, provides the high-level API for interacting with Kafka as well as core data structures and shared utilities.

### Key Files and Their Usage:

*   **`kafka.go`**: Defines fundamental data structures common across the library.
    *   **`Broker`**: Represents a Kafka broker.
    *   **`Topic`**: Represents a Kafka topic, including its name, internal flag, and partitions.
    *   **`Partition`**: Describes a Kafka partition, detailing its topic, ID, leader, replicas, and in-sync replicas (ISR).
    *   **`Version`**: A type for representing Kafka API versions, used for version-specific marshaling/unmarshaling.
    *   **`Marshal`, `Unmarshal`**: Top-level functions for encoding/decoding Go values to/from Kafka's binary format, delegating to the `protocol` package.
*   **`reader.go`**: Implements the Kafka consumer functionality.
    *   **`Reader`**: The primary type for consuming messages. It handles fetching messages, managing offsets, and integrating with Kafka consumer groups.
    *   **`ReaderConfig`**: Configuration for `Reader` instances, including brokers, topic/partition, group ID, dialer, batch sizes, timeouts, and logging.
    *   **`ReaderStats`**: Provides operational statistics for consumers (e.g., message count, bytes, errors, lag).
    *   **`NewReader`**: Constructor for `Reader`.
    *   **`ReadMessage`, `FetchMessage`**: Methods for synchronously reading messages.
    *   **`CommitMessages`**: Commits consumer offsets.
    *   **`Close`**: Shuts down the reader.
*   **`writer.go`**: Implements the Kafka producer functionality.
    *   **`Writer`**: The primary type for producing messages. It manages message batching, partitioning, retries, and acknowledgments (sync/async).
    *   **`WriterConfig`**: Configuration for `Writer` instances (deprecated in favor of direct struct field assignment). Includes brokers, topic, balancer, batch sizes, timeouts, and required acknowledgments.
    *   **`WriterStats`**: Provides operational statistics for producers (e.g., writes, messages, bytes, errors, batch times).
    *   **`NewWriter`**: Constructor for `Writer` (deprecated).
    *   **`WriteMessages`**: The main method for sending messages to Kafka.
    *   **`Close`**: Flushes pending writes and shuts down the writer.
*   **`client.go`**: Provides a high-level API for various administrative and metadata operations.
    *   **`Client`**: Represents a general-purpose Kafka client for direct broker requests.
    *   **`ConsumerOffsets`**: (Deprecated) Retrieves committed offsets for a consumer group.
    *   **`RoundTrip`**: Internal method for executing Kafka protocol requests.
*   **`conn.go`**: Defines the low-level connection to a Kafka broker.
    *   **`Conn`**: Manages raw binary communication, deadlines, I/O buffers, and Kafka protocol requests/responses.
    *   **`ConnConfig`**: Configuration for `Conn` instances.
    *   **`ReadBatchConfig`**: Configuration for reading message batches.
    *   **`NewConn`, `NewConnWith`**: Constructors for `Conn`.
    *   **`ReadBatch`, `WriteMessages`**: Low-level batch read/write operations.
    *   **`Seek`**: Sets the offset for read operations.
    *   **`ReadPartitions`**: Retrieves topic partition metadata.
    *   **`ApiVersions`**: Fetches API versions supported by the broker.
    *   **`saslHandshake`, `saslAuthenticate`**: Performs SASL authentication.
*   **`dialer.go`**: Handles establishing network connections to Kafka brokers.
    *   **`Dialer`**: Extends `net.Dialer` with Kafka-specific features like partition leader resolution, TLS, and SASL authentication.
    *   **`Dial`, `DialContext`**: Establishes basic connections.
    *   **`DialLeader`, `DialPartition`**: Connects to the leader broker of a specific partition.
    *   **`LookupPartition`, `LookupPartitions`**: Discovers partition metadata.
    *   **`DefaultDialer`**: A pre-configured default `Dialer` instance.
*   **`message.go`**: Defines the primary data structures for Kafka messages.
    *   **`Message`**: Represents a single Kafka message with fields like `Topic`, `Partition`, `Offset`, `Key`, `Value`, `Headers`, `Time`.
    *   Internal `message`, `messageSetItem`, `messageSet` types for lower-level message representation.
*   **`error.go`**: Defines custom error types specific to Kafka.
    *   **`Error`**: An `int` type mapping to Kafka protocol error codes, providing `Error()`, `Timeout()`, `Temporary()`, `Title()`, `Description()` methods.
    *   **`MessageTooLargeError`**: Error for messages exceeding size limits.
    *   **`WriteErrors`**: Collects errors from batch write operations.

## 2. `protocol` Package

The `protocol` package contains the core implementation for Kafka's binary wire protocol. It defines the abstract `Message` interface, handles API versioning, and provides the low-level encoding and decoding mechanisms for all Kafka API requests and responses. It is extensively structured with sub-packages for each Kafka API.

### Key Files and Their Usage (within `protocol` root):

*   **`protocol.go`**: Core definitions for the Kafka wire protocol.
    *   **`Message`**: Interface for all Kafka requests/responses.
    *   **`ApiKey`**: Type for Kafka API identifiers (e.g., `Produce`, `Fetch`, `Metadata`).
    *   **`Register`**: Function to register API request/response types.
    *   Interfaces like `RawExchanger`, `BrokerMessage`, `GroupMessage`, `TransactionalMessage`, `PreparedMessage`, `Splitter`, `Merger` for specialized message behaviors.
    *   **`Broker`**, **`Topic`**, **`Partition`**: Redefined for protocol-level details.
*   **`buffer.go`**: Manages efficient memory allocation and buffering for Kafka binary data.
    *   **`Bytes`**: Interface for immutable byte sequences.
    *   **`page`**, **`pageBuffer`**, **`pageRef`**: Structures for page-based memory management with reference counting, optimizing I/O.
*   **`cluster.go`**: Provides a structured representation of Kafka cluster metadata.
    *   **`Cluster`**: Contains `ClusterID`, `Controller`, `Brokers`, and `Topics` data, with methods for accessing and formatting this information.
*   **`conn.go`**: Low-level network connection management for protocol messages.
    *   **`Conn`**: Wraps `net.Conn`, handles buffered I/O, correlation IDs, and API version negotiation at the protocol layer.
    *   **`RoundTrip`**: Method for sending a protocol `Message` and receiving its response.
*   **`decode.go`**: Implements the decoding of Kafka binary data into Go structs.
    *   **`decoder`**: Central type for reading binary data.
    *   **`Unmarshal`**: Main function for deserializing Kafka responses.
    *   Dynamic `decodeFunc`s for various Go types and Kafka encodings (compact, varint).
*   **`encode.go`**: Implements the encoding of Go structs into Kafka binary data.
    *   **`encoder`**: Central type for writing binary data.
    *   **`Marshal`**: Main function for serializing Kafka requests.
    *   Dynamic `encodeFunc`s for various Go types and Kafka encodings.
*   **`request.go`**: Handles the framing, reading, and writing of Kafka protocol requests.
    *   **`ReadRequest`**: Parses an incoming Kafka request, including header and payload.
    *   **`WriteRequest`**: Formats and writes an outgoing Kafka request.
*   **`response.go`**: Handles the framing, reading, and writing of Kafka protocol responses.
    *   **`ReadResponse`**: Parses an incoming Kafka response, including header and payload.
    *   **`WriteResponse`**: Formats and writes an outgoing Kafka response.
*   **`size.go`**: Provides utilities for calculating the encoded size of various Kafka data types.
    *   Functions like `sizeOfVarString`, `sizeOfVarNullBytes`, `sizeOfVarInt` for precise byte size calculations.
*   **`record.go`**: Defines structures and logic for Kafka records and record sets.
    *   **`Attributes`**: Bitset for record properties (compression, transactional).
    *   **`Header`**: Key-value pairs for record headers.
    *   **`Record`**: Interface for a single Kafka message record.
    *   **`RecordSet`**: Represents a batch of records, handling different Kafka message formats.
    *   **`RawRecordSet`**: For raw, pre-encoded record set bytes.

### Sub-packages within `protocol` (by inferred purpose):

Each sub-directory within the `protocol` package typically contains the `Request` and `Response` struct definitions (e.g., `Request`, `Response`, `RequestTopic`, `ResponsePartition`) for a specific Kafka API, along with their associated `ApiKey()` method and `init()` function to register them with the main `protocol` package. These packages encapsulate the Kafka wire format for their respective API operations.

*   **`addoffsetstotxn`**: Defines types for the Kafka "Add Offsets to Transaction" API.
*   **`addpartitionstotxn`**: Defines types for the Kafka "Add Partitions to Transaction" API.
*   **`alterclientquotas`**: Defines types for the Kafka "Alter Client Quotas" API.
*   **`alterconfigs`**: Defines types for the Kafka "Alter Configurations" API.
*   **`alterpartitionreassignments`**: Defines types for the Kafka "Alter Partition Reassignments" API.
*   **`alteruserscramcredentials`**: Defines types for the Kafka "Alter User SCRAM Credentials" API.
*   **`apiversions`**: Defines types for the Kafka "API Versions" API.
*   **`consumer`**: Likely contains internal protocol messages related to consumer group management (e.g., group rebalance, member information).
*   **`createacls`**: Defines types for the Kafka "Create ACLs" API.
*   **`createpartitions`**: Defines types for the Kafka "Create Partitions" API.
*   **`createtopics`**: Defines types for the Kafka "Create Topics" API.
*   **`deleteacls`**: Defines types for the Kafka "Delete ACLs" API.
*   **`deletegroups`**: Defines types for the Kafka "Delete Groups" API.
*   **`deletetopics`**: Defines types for the Kafka "Delete Topics" API.
*   **`describeacls`**: Defines types for the Kafka "Describe ACLs" API.
*   **`describeclientquotas`**: Defines types for the Kafka "Describe Client Quotas" API.
*   **`describeconfigs`**: Defines types for the Kafka "Describe Configurations" API.
*   **`describegroups`**: Defines types for the Kafka "Describe Groups" API.
*   **`describeuserscramcredentials`**: Defines types for the Kafka "Describe User SCRAM Credentials" API.
*   **`electleaders`**: Defines types for the Kafka "Elect Leaders" API.
*   **`endtxn`**: Defines types for the Kafka "End Transaction" API.
*   **`fetch`**: Defines the "Fetch" API request/response, for consumers to retrieve messages.
*   **`findcoordinator`**: Defines types for the Kafka "Find Coordinator" API.
*   **`heartbeat`**: Defines types for the Kafka "Heartbeat" API (for consumer groups).
*   **`incrementalalterconfigs`**: Defines types for the Kafka "Incremental Alter Configurations" API.
*   **`initproducerid`**: Defines types for the Kafka "Init Producer ID" API.
*   **`joingroup`**: Defines types for the Kafka "Join Group" API.
*   **`leavegroup`**: Defines types for the Kafka "Leave Group" API.
*   **`listgroups`**: Defines types for the Kafka "List Groups" API.
*   **`listoffsets`**: Defines types for the Kafka "List Offsets" API.
*   **`listpartitionreassignments`**: Defines types for the Kafka "List Partition Reassignments" API.
*   **`metadata`**: Defines types for the Kafka "Metadata" API (cluster and topic metadata).
*   **`offsetcommit`**: Defines types for the Kafka "Offset Commit" API.
*   **`offsetdelete`**: Defines types for the Kafka "Offset Delete" API.
*   **`offsetfetch`**: Defines types for the Kafka "Offset Fetch" API.
*   **`produce`**: Defines the "Produce" API request/response, for producers to send messages.
*   **`prototest`**: Internal testing utilities for protocol messages.
*   **`rawproduce`**: Defines types for a "Raw Produce" API, possibly for specialized use cases.
*   **`saslauthenticate`**: Defines types for the Kafka "SASL Authenticate" API.
*   **`saslhandshake`**: Defines types for the Kafka "SASL Handshake" API.
*   **`syncgroup`**: Defines types for the Kafka "Sync Group" API.
*   **`txnoffsetcommit`**: Defines types for the Kafka "Txn Offset Commit" API.

## 3. Compression Packages

These packages provide implementations for various compression algorithms used by Kafka.

*   **`compress`**: Likely defines a common `Compression` interface and constants for different compression algorithms.
*   **`gzip`**: Implements the GZIP compression algorithm.
*   **`lz4`**: Implements the LZ4 compression algorithm.
*   **`snappy`**: Implements the Snappy compression algorithm.
*   **`zstd`**: Implements the Zstandard (ZSTD) compression algorithm.

## 4. `sasl` Package

*   **`sasl`**: This package defines interfaces and concrete implementations for various SASL (Simple Authentication and Security Layer) authentication mechanisms supported by Kafka (e.g., SCRAM, PLAIN). It allows `kafka-go` clients to authenticate securely with Kafka brokers.

This concludes the analysis of the `segmentio/kafka-go` library based on its source code.
