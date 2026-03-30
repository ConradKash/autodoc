// autodoc-gen is a code generation tool for building OpenAPI specs at compile time.
//
// Usage:
//
//	go:generate autodoc-gen -router=chi -out=docs_gen.go -spec=openapi.json
//	go:generate autodoc-gen -router=http -out=docs_gen.go
//
// Flags:
//
//	-router       Router type: "chi" or "http" (required)
//	-out          Output Go file (default: "docs_gen.go")
//	-spec         Output OpenAPI spec file (optional, default: no separate file)
//	-pkg          Package name (default: current package)
//	-title        API Title (default: "API")
//	-version      API Version (default: "1.0.0")
//	-scan         Directories to scan (default: ".")
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ConradKash/autodoc"
)

func main() {
	var (
		routerType     = flag.String("router", "", "Router type: chi or http (required)")
		outputFile     = flag.String("out", "docs_gen.go", "Output Go file")
		specOutputFile = flag.String("spec", "", "Output OpenAPI spec file (optional)")
		pkgName        = flag.String("pkg", "", "Package name (auto-detected if empty)")
		title          = flag.String("title", "API", "API title")
		version        = flag.String("version", "1.0.0", "API version")
		scanFlag       = flag.String("scan", ".", "Comma-separated dirs to scan")
		docsPath       = flag.String("docs", "/docs", "Swagger UI path")
		specPath       = flag.String("spec-path", "/openapi.json", "OpenAPI spec path")
	)
	flag.Parse()

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

	cg := autodoc.NewCodeGen(cfg, *routerType)
	cg.OutputFile = *outputFile
	cg.SpecOutputFile = *specOutputFile
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
