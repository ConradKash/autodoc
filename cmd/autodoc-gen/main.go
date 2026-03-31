// autodoc-gen is a code generation tool for building OpenAPI specs at compile time.
//
// Usage:
//
// //go:generate autodoc-gen -router=chi -out=docs_gen.go -spec=openapi.json
// //go:generate autodoc-gen -router=http -out=docs_gen.go
//
// Flags:
//
//	-router       Router type: "chi" or "http" (required)
//	-out          Output Go file (default: "docs_gen.go")
//	-spec         Output OpenAPI spec file (optional, default: no separate file)
//	-spec-yaml    Output OpenAPI spec YAML file (optional)
//	-pkg          Package name (default: current package)
//	-title        API Title (default: "API")
//	-version      API Version (default: "1.0.0")
//	-scan         Directories to scan (default: ".")
//	-models       Comma-separated model type names to include in schema (e.g., "User,Order")
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ConradKash/autodoc"
)

func main() {
	var (
		routerType         = flag.String("router", "", "Router type: chi, gin or http (required)")
		outputFile         = flag.String("out", "../doc/docs_gen.go", "Output Go file (default: ../doc/docs_gen.go)")
		specOutputFile     = flag.String("spec", "../doc/openapi.json", "Output OpenAPI spec JSON file (default: ../doc/openapi.json)")
		specYAMLOutputFile = flag.String("spec-yaml", "../doc/openapi.yaml", "Output OpenAPI spec YAML file (default: ../doc/openapi.yaml)")
		pkgName            = flag.String("pkg", "", "Package name (auto-detected if empty)")
		title              = flag.String("title", "API", "API title")
		version            = flag.String("version", "1.0.0", "API version")
		scanFlag           = flag.String("scan", ".", "Comma-separated dirs to scan")
		docsPath           = flag.String("docs", "/docs", "Swagger UI path")
		specPath           = flag.String("spec-path", "/openapi.json", "OpenAPI spec path")
		modelsFlag         = flag.String("models", "", "Comma-separated model type names to include in schema (e.g., 'User,Order,Product')")
	)
	flag.Parse()

	// Ensure parent directories for all output files exist
	for _, path := range []string{*outputFile, *specOutputFile, *specYAMLOutputFile} {
		if path != "" {
			dir := filepath.Dir(path)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Fatalf("error creating directory %s: %v", dir, err)
				}
			}
		}
	}

	// Validate required flags
	if *routerType == "" {
		log.Fatal("error: -router flag is required (chi or http)")
	}
	if *routerType != "chi" && *routerType != "http" && *routerType != "gin" {
		log.Fatal("error: -router must be 'chi' or 'http' or 'gin'")
	}

	// Auto-detect package name if not provided
	if *pkgName == "" {
		var err error
		*pkgName, err = detectPackageName()
		if err != nil {
			log.Fatalf("error detecting package name: %v (use -pkg flag)", err)
		}
	}

	// Parse scan directories
	scanDirs := strings.Split(*scanFlag, ",")
	for i, d := range scanDirs {
		scanDirs[i] = strings.TrimSpace(d)
	}

	// Create code generator
	cfg := autodoc.Config{
		Title:    *title,
		Version:  *version,
		DocsPath: *docsPath,
		SpecPath: *specPath,
	}

	// Parse models - these will be resolved during code generation
	if *modelsFlag != "" {
		modelNames := strings.Split(*modelsFlag, ",")
		for _, name := range modelNames {
			name = strings.TrimSpace(name)
			if name != "" {
				cfg.Models = append(cfg.Models, name) // Store name as string for codegen
			}
		}
	}

	cg := autodoc.NewCodeGen(cfg, *routerType)
	cg.OutputFile = *outputFile
	cg.SpecOutputFile = *specOutputFile
	cg.SpecYAMLOutputFile = *specYAMLOutputFile
	cg.PackageName = *pkgName
	cg.ScanDirs = scanDirs

	// Generate
	if err := cg.GenerateAll(); err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("✓ Generated %s\n", *outputFile)
	if *specOutputFile != "" {
		fmt.Printf("✓ Generated %s\n", *specOutputFile)
	}
	if *specYAMLOutputFile != "" {
		fmt.Printf("✓ Generated %s\n", *specYAMLOutputFile)
	}
}

// detectPackageName reads the package name from Go files in the current directory
func detectPackageName() (string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		// Simple heuristic: read first few bytes to find package declaration
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			continue
		}

		content := string(data)
		if idx := strings.Index(content, "package "); idx >= 0 {
			start := idx + 8
			end := strings.IndexAny(content[start:], " \t\n")
			if end >= 0 {
				return strings.TrimSpace(content[start : start+end]), nil
			}
		}
	}

	return "main", fmt.Errorf("could not detect package name")
}
