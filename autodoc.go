// Package autodoc provides automatic OpenAPI 3.0 documentation generation for
// Go HTTP handlers — no annotations, no comments, no code generation.
//
// It works like Python's drf-spectacular: mount it, and your entire API is
// documented automatically. Swagger UI and ReDoc are served out of the box.
//
// # How it works
//
// autodoc wraps your http.ServeMux (or any router) and intercepts every call
// to Handle / HandleFunc. It uses reflection to:
//
//  1. Record every registered path + HTTP method
//  2. Extract request/response Go types from typed handlers
//  3. Generate JSON Schema for each type (struct tags, validation tags)
//  4. Build a complete OpenAPI 3.0 spec on-demand
//
// # Quick start
//
//	mux := http.NewServeMux()
//	doc := autodoc.New(autodoc.Config{
//	    Title:   "Acme API",
//	    Version: "1.0.0",
//	})
//	doc.Mount(mux)           // registers /docs, /docs/redoc, /openapi.json
//	doc.Handle(mux, "GET /users", listUsersHandler)
//	doc.HandleTyped(mux, "POST /users", createUserHandler,
//	    autodoc.WithRequestType(reflect.TypeOf(CreateUserRequest{})),
//	    autodoc.WithResponseType(reflect.TypeOf(UserResponse{})),
//	)
//	http.ListenAndServe(":8080", mux)
package autodoc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ─── Config ───────────────────────────────────────────────────────────────────

// Config holds the autodoc configuration.
type Config struct {
	// Title is the API title shown in the spec and UIs. Required.
	Title string
	// Version is the API version string. Defaults to "1.0.0".
	Version string
	// Description is the API description (markdown supported).
	Description string
	// TermsOfService URL.
	TermsOfService string
	// Contact information.
	Contact *Contact
	// License information.
	License *License
	// Servers is the list of server URLs. Defaults to localhost.
	Servers []Server

	// DocsPath is the path prefix for Swagger UI. Defaults to "/docs".
	// Set to "" to disable Swagger UI.
	DocsPath string
	// ReDocPath is the path for ReDoc UI. Defaults to "/docs/redoc".
	// Set to "" to disable ReDoc.
	ReDocPath string
	// SpecPath is the path for the raw OpenAPI JSON spec. Defaults to "/openapi.json".
	SpecPath string

	// ExcludePaths is a list of path prefixes to exclude from the spec.
	// The doc paths themselves (/docs, /openapi.json) are always excluded.
	ExcludePaths []string

	// TagsFunc maps a path to a list of tags. Defaults to first path segment.
	TagsFunc func(method, path string) []string

	// Security schemes to include in components.securitySchemes.
	SecuritySchemes map[string]SecurityScheme
	// GlobalSecurity requirements applied to all operations.
	GlobalSecurity []SecurityRequirement

	// Enabled controls whether the doc UI is served. Useful to disable in prod.
	// Defaults to true.
	Enabled *bool
}

func (c *Config) defaults() {
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.DocsPath == "" {
		c.DocsPath = "/docs"
	}
	if c.ReDocPath == "" {
		c.ReDocPath = "/docs/redoc"
	}
	if c.SpecPath == "" {
		c.SpecPath = "/openapi.json"
	}
	if c.Enabled == nil {
		t := true
		c.Enabled = &t
	}
	if len(c.Servers) == 0 {
		c.Servers = []Server{{URL: "http://localhost", Description: "Default"}}
	}
	if c.TagsFunc == nil {
		c.TagsFunc = defaultTagsFunc
	}
}

func defaultTagsFunc(_, path string) []string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return []string{"default"}
	}
	// Skip versioning segments like v1, v2, api.
	seg := parts[0]
	if seg == "api" || regexp.MustCompile(`^v\d+$`).MatchString(seg) {
		if len(parts) > 1 && parts[1] != "" {
			seg = parts[1]
		}
	}
	return []string{seg}
}

// ─── OpenAPI minimal types (self-contained, no import cycle) ──────────────────

type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`
	In               string      `json:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	Flows            interface{} `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

type SecurityRequirement map[string][]string

// ─── Route descriptor ─────────────────────────────────────────────────────────

// RouteInfo holds everything autodoc knows about a single route.
type RouteInfo struct {
	Method      string
	Path        string // OpenAPI-style path: /users/{id}
	Summary     string
	Description string
	OperationID string
	Tags        []string
	Deprecated  bool

	RequestType  reflect.Type // nil if no request body
	ResponseType reflect.Type // nil if no response body
	StatusCode   int          // default 200

	// Explicit query/header/path params.
	ExtraParams []Param
	// Extra response codes to document alongside the primary.
	ErrorCodes []int
	// Security requirements for this operation (overrides global).
	Security []SecurityRequirement
}

// Param describes an explicit parameter.
type Param struct {
	Name        string
	In          string // path | query | header | cookie
	Description string
	Required    bool
	Type        string // string | integer | number | boolean
	Format      string
	Enum        []string
}

// ─── Handle options ───────────────────────────────────────────────────────────

// HandleOption configures metadata for a single registered route.
type HandleOption func(*RouteInfo)

// WithRequestType tells autodoc the request body type (enables schema generation).
func WithRequestType(t reflect.Type) HandleOption {
	return func(r *RouteInfo) { r.RequestType = t }
}

// WithResponseType tells autodoc the response body type (enables schema generation).
func WithResponseType(t reflect.Type) HandleOption {
	return func(r *RouteInfo) { r.ResponseType = t }
}

// WithRequestOf is a convenience generic wrapper: WithRequestOf[MyRequest]().
func WithRequestOf[T any]() HandleOption {
	return WithRequestType(reflect.TypeOf((*T)(nil)).Elem())
}

// WithResponseOf is a convenience generic wrapper: WithResponseOf[MyResponse]().
func WithResponseOf[T any]() HandleOption {
	return WithResponseType(reflect.TypeOf((*T)(nil)).Elem())
}

// WithSummary sets the OpenAPI summary for the operation.
func WithSummary(s string) HandleOption { return func(r *RouteInfo) { r.Summary = s } }

// WithDescription sets the OpenAPI description.
func WithDescription(d string) HandleOption { return func(r *RouteInfo) { r.Description = d } }

// WithTags overrides the auto-computed tags.
func WithTags(tags ...string) HandleOption {
	return func(r *RouteInfo) { r.Tags = tags }
}

// WithOperationID sets the operationId.
func WithOperationID(id string) HandleOption {
	return func(r *RouteInfo) { r.OperationID = id }
}

// WithStatusCode sets the success status code (default 200).
func WithStatusCode(code int) HandleOption { return func(r *RouteInfo) { r.StatusCode = code } }

// WithDeprecated marks the operation as deprecated.
func WithDeprecated() HandleOption { return func(r *RouteInfo) { r.Deprecated = true } }

// WithParam documents an explicit parameter.
func WithParam(p Param) HandleOption {
	return func(r *RouteInfo) { r.ExtraParams = append(r.ExtraParams, p) }
}

// WithQueryParam is a shorthand for a simple query parameter.
func WithQueryParam(name, description string, required bool) HandleOption {
	return WithParam(Param{Name: name, In: "query", Description: description, Required: required, Type: "string"})
}

// WithErrorCodes documents additional error status codes.
func WithErrorCodes(codes ...int) HandleOption {
	return func(r *RouteInfo) { r.ErrorCodes = append(r.ErrorCodes, codes...) }
}

// WithSecurity sets per-operation security requirements.
func WithSecurity(reqs ...SecurityRequirement) HandleOption {
	return func(r *RouteInfo) { r.Security = reqs }
}

// ─── AutoDoc ──────────────────────────────────────────────────────────────────

// AutoDoc is the core autodoc engine. Create one with New(), register routes
// through it, and mount it onto your mux to serve docs.
type AutoDoc struct {
	cfg    Config
	mu     sync.RWMutex
	routes []*RouteInfo
	gen    *schemaGen

	// cached spec bytes (invalidated on each new route)
	specCache []byte
	specDirty bool

	// pending maps handler pointer → options (used by Describe + ScanMux)
	pendingMu sync.Mutex
	pending   map[string][]HandleOption
}

// New creates a new AutoDoc instance.
func New(cfg Config) *AutoDoc {
	cfg.defaults()
	return &AutoDoc{
		cfg:       cfg,
		gen:       newSchemaGen(),
		specDirty: true,
		pending:   make(map[string][]HandleOption),
	}
}

// ─── Route Registration ───────────────────────────────────────────────────────

// Handle wraps mux.HandleFunc and auto-registers the route in the spec.
// pattern follows Go 1.22 format: "GET /users" or "/users" (method-less).
func (a *AutoDoc) Handle(mux *http.ServeMux, pattern string, handler http.HandlerFunc, opts ...HandleOption) {
	mux.HandleFunc(pattern, handler)
	a.Register(pattern, opts...)
}

// HandleFunc is an alias for Handle.
func (a *AutoDoc) HandleFunc(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request), opts ...HandleOption) {
	a.Handle(mux, pattern, handler, opts...)
}

// Register records route metadata without registering an HTTP handler.
// Use this when you've already registered the handler separately.
func (a *AutoDoc) Register(pattern string, opts ...HandleOption) {
	method, path := parsePattern(pattern)
	if a.isExcluded(path) {
		return
	}

	ri := &RouteInfo{
		Method:     method,
		Path:       toOASPath(path),
		Tags:       a.cfg.TagsFunc(method, path),
		StatusCode: 200,
	}
	ri.OperationID = buildOperationID(method, path)

	for _, o := range opts {
		o(ri)
	}
	if ri.Summary == "" {
		ri.Summary = humanise(method, path)
	}

	a.mu.Lock()
	a.routes = append(a.routes, ri)
	a.specDirty = true
	a.mu.Unlock()
}

// RegisterMany registers multiple patterns at once from an existing mux.
// Useful for zero-annotation integration: pass every pattern you've already
// registered on the mux and autodoc will detect them automatically.
func (a *AutoDoc) RegisterMany(patterns []string, opts ...HandleOption) {
	for _, p := range patterns {
		a.Register(p, opts...)
	}
}

// ─── Middleware (zero-registration approach) ───────────────────────────────────

// Middleware returns an http.Handler that wraps next and silently records
// every request path+method it sees. After a warm-up period (or explicit
// flush), all seen routes appear in the spec.
//
// This is the drf-spectacular equivalent: just wrap your handler and autodoc
// observes traffic to build the spec.
func (a *AutoDoc) Middleware(next http.Handler) http.Handler {
	seen := &sync.Map{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.Path
		if !a.isExcluded(path) && method != http.MethodOptions {
			key := method + " " + path
			if _, loaded := seen.LoadOrStore(key, struct{}{}); !loaded {
				a.Register(method + " " + path)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Mux mounting ─────────────────────────────────────────────────────────────

// Mount registers the Swagger UI, ReDoc, and OpenAPI JSON endpoints on mux.
// Call this after all your routes are registered.
func (a *AutoDoc) Mount(mux *http.ServeMux) {
	if !*a.cfg.Enabled {
		return
	}

	if a.cfg.SpecPath != "" {
		mux.HandleFunc("GET "+a.cfg.SpecPath, a.ServeSpec)
	}
	if a.cfg.DocsPath != "" {
		// Serve Swagger UI at /docs and /docs/
		mux.HandleFunc("GET "+a.cfg.DocsPath, a.ServeSwaggerUI)
		mux.HandleFunc("GET "+a.cfg.DocsPath+"/", a.ServeSwaggerUI)
	}
	if a.cfg.ReDocPath != "" {
		mux.HandleFunc("GET "+a.cfg.ReDocPath, a.ServeReDoc)
	}
}

// Handler returns an http.Handler that serves only the doc endpoints.
// Useful when you want to mount the docs on a sub-path.
func (a *AutoDoc) Handler() http.Handler {
	mux := http.NewServeMux()
	a.Mount(mux)
	return mux
}

// ─── Spec generation ──────────────────────────────────────────────────────────

// Spec returns the fully assembled OpenAPI 3.0 specification as a Go map.
func (a *AutoDoc) Spec() map[string]interface{} {
	a.mu.RLock()
	routes := make([]*RouteInfo, len(a.routes))
	copy(routes, a.routes)
	a.mu.RUnlock()

	return a.buildSpec(routes)
}

// SpecJSON returns the spec as indented JSON bytes.
func (a *AutoDoc) SpecJSON() ([]byte, error) {
	a.mu.Lock()
	if !a.specDirty && a.specCache != nil {
		b := make([]byte, len(a.specCache))
		copy(b, a.specCache)
		a.mu.Unlock()
		return b, nil
	}
	a.mu.Unlock()

	b, err := json.MarshalIndent(a.Spec(), "", "  ")
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.specCache = b
	a.specDirty = false
	a.mu.Unlock()

	return b, nil
}

func (a *AutoDoc) buildSpec(routes []*RouteInfo) map[string]interface{} {
	info := map[string]interface{}{
		"title":   a.cfg.Title,
		"version": a.cfg.Version,
	}
	if a.cfg.Description != "" {
		info["description"] = a.cfg.Description
	}
	if a.cfg.TermsOfService != "" {
		info["termsOfService"] = a.cfg.TermsOfService
	}
	if a.cfg.Contact != nil {
		info["contact"] = a.cfg.Contact
	}
	if a.cfg.License != nil {
		info["license"] = a.cfg.License
	}

	servers := make([]map[string]interface{}, len(a.cfg.Servers))
	for i, s := range a.cfg.Servers {
		servers[i] = map[string]interface{}{"url": s.URL}
		if s.Description != "" {
			servers[i]["description"] = s.Description
		}
	}

	// Build paths.
	paths := map[string]interface{}{}
	schemas := map[string]interface{}{}

	// Collect unique tags.
	tagSet := map[string]bool{}

	for _, ri := range routes {
		if ri.Method == "" {
			continue
		}

		op := a.buildOperation(ri, schemas)
		for _, t := range ri.Tags {
			tagSet[t] = true
		}

		oasPath := ri.Path
		pathItem, ok := paths[oasPath].(map[string]interface{})
		if !ok {
			pathItem = map[string]interface{}{}
			paths[oasPath] = pathItem
		}
		pathItem[strings.ToLower(ri.Method)] = op
	}

	// Sort tags for deterministic output.
	tags := make([]map[string]interface{}, 0, len(tagSet))
	tagNames := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tagNames = append(tagNames, t)
	}
	sort.Strings(tagNames)
	for _, t := range tagNames {
		tags = append(tags, map[string]interface{}{"name": t})
	}

	// Components.
	components := map[string]interface{}{}
	if len(schemas) > 0 {
		components["schemas"] = schemas
	}

	// Problem schema always included.
	if components["schemas"] == nil {
		components["schemas"] = map[string]interface{}{}
	}
	components["schemas"].(map[string]interface{})["Problem"] = problemSchema()

	if len(a.cfg.SecuritySchemes) > 0 {
		components["securitySchemes"] = a.cfg.SecuritySchemes
	}

	spec := map[string]interface{}{
		"openapi":    "3.0.3",
		"info":       info,
		"servers":    servers,
		"paths":      paths,
		"components": components,
		"tags":       tags,
	}

	if len(a.cfg.GlobalSecurity) > 0 {
		spec["security"] = a.cfg.GlobalSecurity
	}

	return spec
}

func (a *AutoDoc) buildOperation(ri *RouteInfo, schemas map[string]interface{}) map[string]interface{} {
	op := map[string]interface{}{
		"operationId": ri.OperationID,
		"tags":        ri.Tags,
		"responses":   map[string]interface{}{},
	}

	if ri.Summary != "" {
		op["summary"] = ri.Summary
	}
	if ri.Description != "" {
		op["description"] = ri.Description
	}
	if ri.Deprecated {
		op["deprecated"] = true
	}
	if len(ri.Security) > 0 {
		op["security"] = ri.Security
	} else if len(a.cfg.GlobalSecurity) > 0 {
		op["security"] = a.cfg.GlobalSecurity
	}

	// Path parameters (auto-extracted from path).
	params := []interface{}{}
	for _, seg := range strings.Split(ri.Path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			params = append(params, map[string]interface{}{
				"name":     name,
				"in":       "path",
				"required": true,
				"schema":   map[string]interface{}{"type": "string"},
			})
		}
	}
	// Extra explicit params.
	for _, p := range ri.ExtraParams {
		pm := map[string]interface{}{
			"name":     p.Name,
			"in":       p.In,
			"required": p.Required,
			"schema":   paramSchema(p),
		}
		if p.Description != "" {
			pm["description"] = p.Description
		}
		params = append(params, pm)
	}
	if len(params) > 0 {
		op["parameters"] = params
	}

	// Request body.
	method := strings.ToUpper(ri.Method)
	if ri.RequestType != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		schemaRef := a.gen.schemaRef(ri.RequestType, schemas)
		op["requestBody"] = map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": schemaRef,
				},
			},
		}
	}

	// Responses.
	responses := map[string]interface{}{}
	statusCode := ri.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	statusKey := fmt.Sprintf("%d", statusCode)

	if ri.ResponseType != nil {
		schemaRef := a.gen.schemaRef(ri.ResponseType, schemas)
		responses[statusKey] = map[string]interface{}{
			"description": httpStatusText(statusCode),
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": schemaRef,
				},
			},
		}
	} else {
		responses[statusKey] = map[string]interface{}{
			"description": httpStatusText(statusCode),
		}
	}

	// Standard error responses.
	problemRef := map[string]interface{}{"$ref": "#/components/schemas/Problem"}
	defaultErrors := []int{400, 401, 403, 404, 500}
	allErrors := append(defaultErrors, ri.ErrorCodes...)
	seenCodes := map[int]bool{statusCode: true}
	for _, code := range allErrors {
		if seenCodes[code] {
			continue
		}
		seenCodes[code] = true
		responses[fmt.Sprintf("%d", code)] = map[string]interface{}{
			"description": httpStatusText(code),
			"content": map[string]interface{}{
				"application/problem+json": map[string]interface{}{
					"schema": problemRef,
				},
			},
		}
	}
	op["responses"] = responses

	return op
}

// ─── HTTP handlers ────────────────────────────────────────────────────────────

// ServeSpec serves the OpenAPI JSON spec.
func (a *AutoDoc) ServeSpec(w http.ResponseWriter, r *http.Request) {
	b, err := a.SpecJSON()
	if err != nil {
		http.Error(w, "failed to generate spec: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// ServeSwaggerUI serves the Swagger UI HTML page.
func (a *AutoDoc) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	replacer := strings.NewReplacer(
		"{{TITLE}}", a.cfg.Title,
		"{{SPEC_URL}}", a.cfg.SpecPath,
	)
	_, _ = replacer.WriteString(w, SwaggerUIHTML)
}

// ServeReDoc serves the ReDoc HTML page.
func (a *AutoDoc) ServeReDoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	replacer := strings.NewReplacer(
		"{{TITLE}}", a.cfg.Title,
		"{{SPEC_URL}}", a.cfg.SpecPath,
	)
	_, _ = replacer.WriteString(w, RedocHTML)
}

// Config getters for adapter subpackages.

// GetTitle returns the configured API title.
func (a *AutoDoc) GetTitle() string { return a.cfg.Title }

// SpecPath returns the configured OpenAPI spec endpoint path.
func (a *AutoDoc) GetSpecPath() string { return a.cfg.SpecPath }

// DocsPath returns the configured Swagger UI endpoint path.
func (a *AutoDoc) GetDocsPath() string { return a.cfg.DocsPath }

// ReDocPath returns the configured ReDoc endpoint path.
func (a *AutoDoc) GetReDocPath() string { return a.cfg.ReDocPath }

// IsEnabled returns whether doc serving is enabled.
func (a *AutoDoc) IsEnabled() bool { return a.cfg.Enabled != nil && *a.cfg.Enabled }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (a *AutoDoc) isExcluded(path string) bool {
	excluded := []string{a.cfg.DocsPath, a.cfg.ReDocPath, a.cfg.SpecPath}
	for _, ex := range append(excluded, a.cfg.ExcludePaths...) {
		if ex != "" && (path == ex || strings.HasPrefix(path, ex+"/")) {
			return true
		}
	}
	return false
}

func parsePattern(pattern string) (method, path string) {
	parts := strings.SplitN(strings.TrimSpace(pattern), " ", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	m := strings.ToUpper(parts[0])
	if isHTTPMethod(m) {
		return m, parts[1]
	}
	return "", pattern
}

func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
		return true
	}
	return false
}

// toOASPath converts Go 1.22 mux patterns to OpenAPI paths.
// /users/{id...} → /users/{id}
func toOASPath(path string) string {
	re := regexp.MustCompile(`\{([^}]+)\.\.\.}`)
	path = re.ReplaceAllStringFunc(path, func(s string) string {
		inner := s[1 : len(s)-1]
		inner = strings.TrimSuffix(inner, "...")
		return "{" + inner + "}"
	})
	return path
}

func buildOperationID(method, path string) string {
	parts := []string{strings.ToLower(method)}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") {
			name := seg[1 : len(seg)-1]
			name = strings.TrimSuffix(name, "...")
			seg = "by_" + name
		}
		parts = append(parts, strings.ReplaceAll(seg, "-", "_"))
	}
	return strings.Join(parts, "_")
}

func humanise(method, path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	var label []string
	for _, s := range segs {
		if !strings.HasPrefix(s, "{") {
			label = append(label, strings.Title(strings.ReplaceAll(s, "-", " ")))
		}
	}
	resource := strings.Join(label, " ")
	switch strings.ToUpper(method) {
	case "GET":
		if strings.Contains(path, "{") {
			return "Get " + resource
		}
		return "List " + resource
	case "POST":
		return "Create " + resource
	case "PUT":
		return "Replace " + resource
	case "PATCH":
		return "Update " + resource
	case "DELETE":
		return "Delete " + resource
	}
	return strings.ToUpper(method) + " " + resource
}

func paramSchema(p Param) map[string]interface{} {
	s := map[string]interface{}{"type": p.Type}
	if p.Type == "" {
		s["type"] = "string"
	}
	if p.Format != "" {
		s["format"] = p.Format
	}
	if len(p.Enum) > 0 {
		enums := make([]interface{}, len(p.Enum))
		for i, e := range p.Enum {
			enums[i] = e
		}
		s["enum"] = enums
	}
	return s
}

func httpStatusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return fmt.Sprintf("HTTP %d", code)
}

func problemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Problem Details (RFC 9457)",
		"properties": map[string]interface{}{
			"type":     map[string]interface{}{"type": "string", "format": "uri"},
			"title":    map[string]interface{}{"type": "string"},
			"status":   map[string]interface{}{"type": "integer"},
			"detail":   map[string]interface{}{"type": "string"},
			"instance": map[string]interface{}{"type": "string", "format": "uri"},
		},
		"required": []string{"title", "status"},
	}
}
