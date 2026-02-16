package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime" // Added for runtime.Caller
)

func main() {
	addr := "http://localhost:8001/apis/registry/v3" // Hardcode for main.go

	// Determine protoRoot relative to main.go's execution
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintf(os.Stderr, "Failed to get current file information\n")
		os.Exit(1)
	}
	currentDir := filepath.Dir(filename)
	protoRoot := filepath.Join(currentDir, "../proto")

	if err := RegisterProtoArtifacts(addr, protoRoot); err != nil { // Call function from proto_register.go
		fmt.Fprintf(os.Stderr, "Error registering artifacts: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("All proto artifacts registered successfully.")
}
