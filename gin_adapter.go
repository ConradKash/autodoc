package autodoc

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GinAdapter wraps a gin.Engine and auto-registers routes with AutoDoc.
type GinAdapter struct {
	Engine *gin.Engine
	doc    *AutoDoc
}

// NewGinAdapter creates a GinAdapter backed by a new gin.Engine.
func NewGinAdapter(doc *AutoDoc) *GinAdapter {
	engine := gin.New()
	return &GinAdapter{Engine: engine, doc: doc}
}

// convertGinPatternToOAS converts Gin's ":param" syntax to OpenAPI's "{param}".
func convertGinPatternToOAS(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] == ':' {
			j := i + 1
			for j < len(pattern) && (pattern[j] >= 'a' && pattern[j] <= 'z' || pattern[j] >= 'A' && pattern[j] <= 'Z' || pattern[j] >= '0' && pattern[j] <= '9' || pattern[j] == '_') {
				j++
			}
			b.WriteByte('{')
			b.WriteString(pattern[i+1 : j])
			b.WriteByte('}')
			i = j
		} else {
			b.WriteByte(pattern[i])
			i++
		}
	}
	return b.String()
}

// Handle registers a handler and auto-documents the route.
func (a *GinAdapter) Handle(method, pattern string, handler gin.HandlerFunc, opts ...HandleOption) {
	a.Engine.Handle(method, pattern, handler)
	patternOAS := convertGinPatternToOAS(pattern)
	patternStr := method + " " + patternOAS
	a.doc.Register(patternStr, opts...)
}

// Mount registers the doc endpoints on the underlying gin.Engine.
func (a *GinAdapter) Mount() {
	// Gin uses its own router, so we need to register the doc endpoints directly.
	// Serve the OpenAPI spec
	a.Engine.GET(a.doc.cfg.SpecPath, func(c *gin.Context) {
		a.doc.serveSpecGin(c)
	})
	// Serve Swagger UI
	if a.doc.cfg.DocsPath != "" {
		a.Engine.GET(a.doc.cfg.DocsPath, func(c *gin.Context) {
			a.doc.serveSwaggerUIGin(c)
		})
	}
	// Serve ReDoc UI
	if a.doc.cfg.ReDocPath != "" {
		a.Engine.GET(a.doc.cfg.ReDocPath, func(c *gin.Context) {
			a.doc.serveReDocGin(c)
		})
	}
}

// Gin-compatible wrappers for doc endpoints
func (a *AutoDoc) serveSpecGin(c *gin.Context) {
	b, err := a.SpecJSON()
	if err != nil {
		c.String(500, "failed to generate spec: %s", err.Error())
		return
	}
	c.Data(200, "application/json; charset=utf-8", b)
}

func (a *AutoDoc) serveSwaggerUIGin(c *gin.Context) {
	replacer := strings.NewReplacer(
		"{{TITLE}}", a.cfg.Title,
		"{{SPEC_URL}}", a.cfg.SpecPath,
	)
	html := replacer.Replace(swaggerUIHTML)
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

func (a *AutoDoc) serveReDocGin(c *gin.Context) {
	replacer := strings.NewReplacer(
		"{{TITLE}}", a.cfg.Title,
		"{{SPEC_URL}}", a.cfg.SpecPath,
	)
	html := replacer.Replace(redocHTML)
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

// Optionally, add HandleFunc for compatibility
func (a *GinAdapter) HandleFunc(method, pattern string, handler func(*gin.Context), opts ...HandleOption) {
	a.Handle(method, pattern, handler, opts...)
}
