// Package gin provides a gin.Engine adapter for autodoc.
//
// This adapter registers routes with gin and auto-documents them.
// Import this package only if you use gin — it pulls in gin as a dependency.
//
//	adapter := gin.NewGinAdapter(doc)
//	adapter.Get("/users", listUsers, autodoc.WithSummary("List users"))
//	adapter.Mount() // serves /docs, /openapi.json, etc
package gin

import (
	"strings"

	"github.com/ConradKash/autodoc"
	"github.com/gin-gonic/gin"
)

// GinAdapter wraps a gin.Engine and auto-registers routes with AutoDoc.
type GinAdapter struct {
	Engine *gin.Engine
	doc    *autodoc.AutoDoc
}

// NewGinAdapter creates a GinAdapter backed by a new gin.Engine.
func NewGinAdapter(doc *autodoc.AutoDoc) *GinAdapter {
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
func (a *GinAdapter) Handle(method, pattern string, handler gin.HandlerFunc, opts ...autodoc.HandleOption) {
	a.Engine.Handle(method, pattern, handler)
	patternOAS := convertGinPatternToOAS(pattern)
	patternStr := method + " " + patternOAS
	a.doc.Register(patternStr, opts...)
}

// HandleFunc is an alias for Handle.
func (a *GinAdapter) HandleFunc(method, pattern string, handler func(*gin.Context), opts ...autodoc.HandleOption) {
	a.Handle(method, pattern, handler, opts...)
}

// Mount registers the doc endpoints on the underlying gin.Engine.
func (a *GinAdapter) Mount() {
	if !a.doc.IsEnabled() {
		return
	}

	title := a.doc.GetTitle()

	if p := a.doc.GetSpecPath(); p != "" {
		a.Engine.GET(p, func(c *gin.Context) {
			b, err := a.doc.SpecJSON()
			if err != nil {
				c.String(500, "failed to generate spec: %s", err.Error())
				return
			}
			c.Data(200, "application/json; charset=utf-8", b)
		})
	}
	if p := a.doc.GetDocsPath(); p != "" {
		a.Engine.GET(p, func(c *gin.Context) {
			replacer := strings.NewReplacer("{{TITLE}}", title, "{{SPEC_URL}}", a.doc.GetSpecPath())
			c.Data(200, "text/html; charset=utf-8", []byte(replacer.Replace(autodoc.SwaggerUIHTML)))
		})
	}
	if p := a.doc.GetReDocPath(); p != "" {
		a.Engine.GET(p, func(c *gin.Context) {
			replacer := strings.NewReplacer("{{TITLE}}", title, "{{SPEC_URL}}", a.doc.GetSpecPath())
			c.Data(200, "text/html; charset=utf-8", []byte(replacer.Replace(autodoc.RedocHTML)))
		})
	}
}
