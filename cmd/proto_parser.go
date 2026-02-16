package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// importRegex captures 'import "path/to/file.proto";'
var importRegex = regexp.MustCompile(`^import\s+"([^"]+)";`)

// ParseProtoFiles scans the given protoRoot, parses imports, and performs a topological sort.
// It returns a map of import paths to absolute file paths, a map of file paths to their imports,
// a topologically sorted list of file paths, and any error encountered.
func ParseProtoFiles(protoRoot string) (map[string]string, map[string][]string, []string, error) {
	importToPath := make(map[string]string)
	fileToImports := make(map[string][]string)
	var sortedFiles []string

	err := filepath.WalkDir(protoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".proto") {
			return nil // Skip directories and non-proto files
		}

		relPath, err := filepath.Rel(protoRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		importToPath[relPath] = path

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			matches := importRegex.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
			if len(matches) > 1 {
				fileToImports[path] = append(fileToImports[path], matches[1])
			}
		}
		if err = scanner.Err(); err != nil {
			return fmt.Errorf("failed to scan file %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to scan proto directory: %w", err)
	}

	// Topological Sort (Ensure dependencies are first)
	visited := make(map[string]bool)
	var visit func(string)

	visit = func(path string) {
		if visited[path] {
			return
		}
		visited[path] = true
		for _, imp := range fileToImports[path] {
			if fullPath, exists := importToPath[imp]; exists {
				visit(fullPath)
			}
		}
		sortedFiles = append(sortedFiles, path)
	}

	// Ensure all files are visited, including those without imports
	for _, path := range importToPath {
		visit(path)
	}

	return importToPath, fileToImports, sortedFiles, nil
}
