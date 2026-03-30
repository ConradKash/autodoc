package autodoc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ─── Code Generation (Build-Time) ──────────────────────────────────────────

// CodeGen is a build-time generator that scans your router registration calls
// and generates a Go source file with embedded OpenAPI spec and route table.
//
// Usage:
//
//	go generate ./...
//
// In your code, add:
//
//	//go:generate autodoc-gen -router=chi -out=generated_docs.go -spec=openapi.json
//
// This will:
//  1. Parse all .go files in the package
//  2. Find chi.Router or http.ServeMux registrations
//  3. Extract route patterns, methods, and type hints
//  4. Generate a static route table + embedded OpenAPI spec
//  5. Emit generated_docs.go with zero runtime cost
type CodeGen struct {
	// RouterType is "chi" or "http"
	RouterType string
	// OutputFile is where to write generated code
	OutputFile string
	// SpecOutputFile is where to write the OpenAPI spec (optional, separate file)
	SpecOutputFile string
	// PackageName is the target package name
	PackageName string
	// Paths to scan for router registration code
	ScanDirs []string
	// Config to use for the generated spec
	Config Config
}

// Route represents a single discovered route at build time.
type Route struct {
	Method       string
	Path         string
	Summary      string
	Description  string
	OperationID  string
	RequestType  string // fully qualified type name, e.g., "github.com/example/api.CreateUserRequest"
	ResponseType string // fully qualified type name
	StatusCode   int
	Tags         []string
}

// NewCodeGen creates a new code generation engine.
func NewCodeGen(cfg Config, routerType string) *CodeGen {
	return &CodeGen{
		RouterType: routerType,
		Config:     cfg,
		ScanDirs:   []string{"."},
	}
}

// Scan walks the filesystem and extracts route registrations.
func (cg *CodeGen) Scan() ([]Route, error) {
	var routes []Route

	for _, dir := range cg.ScanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
			if err != nil {
				// Skip files that don't parse; they may be incomplete
				return nil
			}

			ast.Inspect(node, func(n ast.Node) bool {
				routes = cg.extractRoutes(n, routes)
				return true
			})

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return routes, nil
}

// extractRoutes finds route registration calls in the AST.
// Supports patterns like:
//   - chi: r.Get(), r.Post(), r.Route()
//   - http: mux.HandleFunc("GET /path", ...)
func (cg *CodeGen) extractRoutes(node ast.Node, routes []Route) []Route {
	if callExpr, ok := node.(*ast.CallExpr); ok {
		route := cg.parseRouteCall(callExpr)
		if route != nil {
			routes = append(routes, *route)
		}
	}
	return routes
}

// parseRouteCall extracts a Route from a function call expression.
func (cg *CodeGen) parseRouteCall(call *ast.CallExpr) *Route {
	// Get the function being called
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	method := fn.Sel.Name

	switch cg.RouterType {
	case "chi":
		return cg.parseChiRoute(method, call)
	case "http":
		return cg.parseHTTPRoute(method, call)
	}

	return nil
}

// parseChiRoute extracts routes from chi calls like r.Get("/path", handler).
//
// Supported chi patterns:
//   - r.Get("/path", handler)              → GET /path
//   - r.Post("/path", handler)             → POST /path
//   - r.Put("/path", handler)              → PUT /path
//   - r.Delete("/path", handler)           → DELETE /path
//   - r.Patch("/path", handler)            → PATCH /path
//   - r.Head("/path", handler)             → HEAD /path
//   - r.Options("/path", handler)          → OPTIONS /path
//   - r.Method("GET", "/path", handler)    → GET /path
//   - r.MethodFunc("GET", "/path", handler) → GET /path
//   - r.Handle("/path", handler)           → all methods /path
//   - r.HandleFunc("/path", handler)       → all methods /path
//   - r.Route("/prefix", ...)              → skipped (sub-router)
//   - r.Group(...)                         → skipped (middleware group)
//   - r.Mount("/path", subRouter)          → skipped (mount)
func (cg *CodeGen) parseChiRoute(method string, call *ast.CallExpr) *Route {
	switch method {
	case "Get", "Post", "Put", "Delete", "Patch", "Head", "Options":
		return cg.parseChiHTTPMethodRoute(method, call)
	case "Method", "MethodFunc":
		return cg.parseChiMethodRoute(call)
	case "Handle", "HandleFunc":
		return cg.parseChiCatchAllRoute(call)
	case "Route", "Group", "Mount":
		return nil // sub-routers/groups handled at runtime or separately
	default:
		return nil
	}
}

// parseChiHTTPMethodRoute handles r.Get("/path", handler), r.Post(...), etc.
func (cg *CodeGen) parseChiHTTPMethodRoute(httpMethod string, call *ast.CallExpr) *Route {
	if len(call.Args) < 2 {
		return nil
	}

	pathLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return nil
	}

	path := strings.Trim(pathLit.Value, `"`)
	return &Route{
		Method:      strings.ToUpper(httpMethod),
		Path:        path,
		OperationID: buildOperationID(httpMethod, path),
		StatusCode:  200,
		Tags:        []string{extractTag(path)},
	}
}

// parseChiMethodRoute handles r.Method("GET", "/path", handler) and r.MethodFunc(...).
func (cg *CodeGen) parseChiMethodRoute(call *ast.CallExpr) *Route {
	if len(call.Args) < 3 {
		return nil
	}

	methodLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || methodLit.Kind != token.STRING {
		return nil
	}
	pathLit, ok := call.Args[1].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return nil
	}

	httpMethod := strings.ToUpper(strings.Trim(methodLit.Value, `"`))
	path := strings.Trim(pathLit.Value, `"`)
	return &Route{
		Method:      httpMethod,
		Path:        path,
		OperationID: buildOperationID(httpMethod, path),
		StatusCode:  200,
		Tags:        []string{extractTag(path)},
	}
}

// parseChiCatchAllRoute handles r.Handle("/path", handler) and r.HandleFunc(...).
// These register a handler for all HTTP methods; we emit a GET entry by default.
func (cg *CodeGen) parseChiCatchAllRoute(call *ast.CallExpr) *Route {
	if len(call.Args) < 2 {
		return nil
	}

	pathLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || pathLit.Kind != token.STRING {
		return nil
	}

	path := strings.Trim(pathLit.Value, `"`)
	return &Route{
		Method:      "GET",
		Path:        path,
		OperationID: buildOperationID("GET", path),
		StatusCode:  200,
		Tags:        []string{extractTag(path)},
	}
}

// parseHTTPRoute extracts routes from http.ServeMux calls like
// mux.HandleFunc("GET /path", handler)
func (cg *CodeGen) parseHTTPRoute(method string, call *ast.CallExpr) *Route {
	if method != "HandleFunc" && method != "Handle" {
		return nil
	}

	if len(call.Args) < 1 {
		return nil
	}

	// First arg is the pattern "GET /path"
	patternLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || patternLit.Kind != token.STRING {
		return nil
	}

	pattern := strings.Trim(patternLit.Value, `"`)
	httpMethod, path := parsePattern(pattern)

	if httpMethod == "" {
		return nil
	}

	return &Route{
		Method:      httpMethod,
		Path:        toOASPath(path),
		OperationID: buildOperationID(httpMethod, path),
		StatusCode:  200,
		Tags:        []string{extractTag(path)},
	}
}

// Generate writes the generated Go code to OutputFile.
func (cg *CodeGen) Generate(routes []Route) error {
	// Build the OpenAPI spec from routes
	doc := New(cg.Config)
	for _, r := range routes {
		doc.Register(r.Method+" "+r.Path,
			WithSummary(r.Summary),
			WithOperationID(r.OperationID),
			WithTags(r.Tags...),
			WithStatusCode(r.StatusCode),
		)
	}

	specJSON, err := doc.SpecJSON()
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}

	// Write the spec to a separate file if requested
	if cg.SpecOutputFile != "" {
		if err := os.WriteFile(cg.SpecOutputFile, specJSON, 0644); err != nil {
			return fmt.Errorf("failed to write spec file: %w", err)
		}
	}

	// Embed the spec as a string in the generated Go code
	specStr := string(specJSON)
	escapedSpec := strings.ReplaceAll(specStr, `"`, `\"`)
	escapedSpec = strings.ReplaceAll(escapedSpec, "\n", `\n`)

	code := fmt.Sprintf(`// Code generated by autodoc-gen; DO NOT EDIT.
package %s

import (
	"encoding/json"
)

// generatedOpenAPISpec is the embedded OpenAPI 3.0 specification.
// Generated at build time from router registration code.
const generatedOpenAPISpec = "%s"

// GetOpenAPISpec returns the embedded OpenAPI specification.
func GetOpenAPISpec() map[string]interface{} {
	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(generatedOpenAPISpec), &spec); err != nil {
		panic(err)
	}
	return spec
}

// StaticRoutes is a pre-computed route table for fast lookup at runtime.
var StaticRoutes = []struct {
	Method string
	Path   string
}{
`, cg.PackageName, escapedSpec)

	// Add route entries
	for _, r := range routes {
		code += fmt.Sprintf("\t{Method: %q, Path: %q},\n", r.Method, r.Path)
	}

	code += `}
`

	// Write the generated file
	if err := os.WriteFile(cg.OutputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Printf("Generated %s with %d routes", cg.OutputFile, len(routes))
	return nil
}

// GenerateAll is a convenience method that scans and generates in one call.
func (cg *CodeGen) GenerateAll() error {
	routes, err := cg.Scan()
	if err != nil {
		return err
	}
	return cg.Generate(routes)
}

// ─── Helper functions ─────────────────────────────────────────────────────────

func extractTag(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "default"
	}
	seg := parts[0]
	// Skip versioning segments like v1, v2, api
	if seg == "api" || isVersionSegment(seg) {
		if len(parts) > 1 && parts[1] != "" {
			seg = parts[1]
		}
	}
	return seg
}

func isVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
