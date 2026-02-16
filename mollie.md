# Mollie Go Apicurio Registry Client - Package Analysis (Revised)

This document provides an updated overview of the `models`, `apis`, and `client` packages within the `mollie/go-apicurio-registry` Go module, explaining their purpose and how their APIs are structured and used, based on direct code analysis.

## 1. `models` Package

*   **Location:** `go-apicurio-registry/models`
*   **Purpose:** The `models` package defines the core data structures (Go structs) that represent various entities and their metadata within the Apicurio Registry. These models are fundamental for serializing and deserializing data when interacting with the Apicurio Registry API. They serve as the common data language between your Go application and the Apicurio Registry.
*   **API Usage:** Developers will use these structs to:
    *   Construct request bodies for API calls (e.g., providing artifact content or metadata).
    *   Interpret response bodies received from the API (e.g., reading artifact details, search results, or user information).
    *   Work with and manipulate domain-specific data returned by the Apicurio Registry.

    **Examples of Models (from `models.go`):**
    *   `ArtifactReference`: Represents a reference to an artifact.
    *   `SearchedArtifact`: Represents the search result of an artifact.
    *   `ArtifactContent`: Represents the content of an artifact along with its type.
    *   `ArtifactDetail`: Provides detailed information about an artifact.
    *   `ArtifactVersion`: Represents a single version of an artifact with its metadata.
    *   `UserInfo`: Contains information about a user.
    *   `GroupInfo`: Represents details about an artifact group.
    *   `BranchInfo`: Describes a branch of artifacts.

## 2. `client` Package (Revised)

*   **Location:** `go-apicurio-registry/client`
*   **Purpose:** The `client` package provides the foundational and core implementation for connecting and interacting with the Apicurio Registry. It handles the underlying HTTP communication, request serialization (implicitly, by accepting `interface{}` for the body), response deserialization, error handling, and authentication. It serves as the central configuration and communication manager for the entire client library.
*   **API Usage:** This is typically the first package a developer interacts with to set up their connection to the Apicurio Registry.

    **Key Components & Usage:**
    *   **`Client` struct**: The main client object that encapsulates the `BaseURL`, an `*http.Client`, and an `AuthHeader` (e.g., a bearer token).
    *   **`Option` type**: A function signature (`func(*Client)`) used to configure the `Client` struct using the functional options pattern.
    *   **Configuration Functions (Functional Options)**:
        *   `WithRetryableHTTP(cfg *retryablehttp.Client)`: Configures the client to use a retryable HTTP client, optionally with a custom `retryablehttp.Client` instance.
        *   `WithHTTPClient(httpClient *http.Client)`: Allows providing a completely custom `http.Client` instance.
        *   `WithAuthHeader(authHeader string)`: Sets the `Authorization` header for all requests made by this client.
    *   **`NewClient(baseURL string, options ...Option) *Client`**: This is the *actual* constructor function for the core client. It takes the registry's base URL and a variadic list of `Option` functions to apply custom configurations.
    *   **`Do(req *http.Request) (*http.Response, error)`**: A method on the `Client` struct used to execute raw HTTP requests. It automatically adds the `Authorization` and `Content-Type: application/json` headers if `AuthHeader` is set. This method is primarily used internally by the `apis` package.

    **Overall Interaction Flow (Client Initialization):**

    ```go
    package main

    import (
    	"fmt"
    	"github.com/mollie/go-apicurio-registry/client"
    	"github.com/hashicorp/go-retryablehttp" // if using retryable http
    )

    func main() {
    	// Example 1: Basic client
    	simpleClient := client.NewClient("https://my-registry.example.com")

    	// Example 2: Client with authentication and retries
    	retryCfg := retryablehttp.NewClient()
    	retryCfg.RetryMax = 5 // Custom retry attempts

    	configuredClient := client.NewClient(
    		"https://my-registry.example.com",
    		client.WithAuthHeader("Bearer my-secret-token"),
    		client.WithRetryableHTTP(retryCfg),
    	)
    	_ = simpleClient // Use the client to avoid compile errors
    	_ = configuredClient // Use the client to avoid compile errors

    	fmt.Println("Clients initialized.")
    }
    ```

## 3. `apis` Package (Revised)

*   **Location:** `go-apicurio-registry/apis`
*   **Purpose:** The `apis` package provides structured, high-level interfaces for interacting with different functional domains of the Apicurio Registry. It groups related API operations into distinct API clients (e.g., `ArtifactsAPI`, `AdminAPI`), offering a modular and user-friendly abstraction over the core `client` package. Each API client within `apis` leverages an underlying `client.Client` instance to perform its HTTP requests.
*   **API Usage:** Developers will typically obtain specific API clients from this package. These clients are instantiated by passing an *already initialized* `*client.Client` instance to their respective constructor functions. Each API client then provides methods that map directly to Apicurio Registry operations, often accepting and returning structs from the `models` package.

    **Available APIs (Examples):**
    *   **`ArtifactsAPI`**: Manages schema artifacts (creation, retrieval, updating, deletion). Constructor: `apis.NewArtifactsAPI(*client.Client)`.
    *   **`VersionsAPI`**: Handles operations related to artifact versions. Constructor: `apis.NewVersionsAPI(*client.Client)`.
    *   **`GroupsAPI`**: Supports grouping of artifacts. Constructor: `apis.NewGroupsAPI(*client.Client)`.
    *   **`MetadataAPI`**: Provides methods to manage artifact metadata. Constructor: `apis.NewMetadataAPI(*client.Client)`.
    *   **`AdminAPI`**: Enables administrative operations. Constructor: `apis.NewAdminAPI(*client.Client)`.
    *   **`SystemAPI`**: Offers methods for querying registry status and configuration. Constructor: `apis.NewSystemAPI(*client.Client)`.

    **Example Flow (Integrating `client` and `apis`):**

    ```go
    package main

    import (
    	"context"
    	"fmt"
    	"github.com/mollie/go-apicurio-registry/client"
    	"github.com/mollie/go-apicurio-registry/apis"
    	"github.com/mollie/go-apicurio-registry/models"
    )

    func main() {
    	// 1. Initialize the core client from the 'client' package
    	coreClient := client.NewClient("https://my-registry.example.com", client.WithAuthHeader("Bearer my-token"))

    	// 2. Instantiate a specific API client from the 'apis' package, passing the core client
    	artifactsAPI := apis.NewArtifactsAPI(coreClient)

    	// 3. Use methods on the API client to interact with the Registry
    	//    (Example: search for artifacts)
    	searchResults, err := artifactsAPI.SearchArtifacts(context.Background(), &models.SearchArtifactsParams{})
    	if err != nil {
    		fmt.Printf("Error searching artifacts: %v\n", err)
    		return
    	}
    	fmt.Printf("Found %d artifacts.\n", len(searchResults))

    	// Further operations using other API clients would follow a similar pattern
    }
    ```
