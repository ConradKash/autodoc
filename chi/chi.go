// Package chi provides a chi.Router adapter for autodoc.
//
// This adapter registers routes with chi and auto-documents them.
// Import this package only if you use chi — it pulls in chi as a dependency.
//
//	adapter := chi.NewChiAdapter(doc, router)
//	adapter.Get("/users", listUsers, autodoc.WithSummary("List users"))
//	adapter.Mount() // serves /docs, /openapi.json, etc
package chi

import (
	"net/http"

	"github.com/ConradKash/autodoc"
	"github.com/go-chi/chi/v5"
)

// ChiAdapter wraps a chi.Router and auto-registers routes with AutoDoc.
type ChiAdapter struct {
	router *chi.Mux
	doc    *autodoc.AutoDoc
	prefix string
}

// NewChiAdapter creates a ChiAdapter.
func NewChiAdapter(doc *autodoc.AutoDoc, router *chi.Mux) *ChiAdapter {
	return &ChiAdapter{router: router, doc: doc}
}

// Handle registers a handler with documentation options.
func (a *ChiAdapter) Handle(method, pattern string, handler http.Handler, opts ...autodoc.HandleOption) {
	fullPattern := a.prefix + pattern
	a.router.Method(method, pattern, handler)
	patternStr := method + " " + fullPattern
	a.doc.Register(patternStr, opts...)
}

// Use registers middleware to the adapter.
func (a *ChiAdapter) Use(middleware func(http.Handler) http.Handler) {
	a.router.Use(middleware)
}

// HandleFunc is a convenience wrapper for http.HandlerFunc.
func (a *ChiAdapter) HandleFunc(method, pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.Handle(method, pattern, handler, opts...)
}

// Get registers a GET handler.
func (a *ChiAdapter) Get(pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.HandleFunc(http.MethodGet, pattern, handler, opts...)
}

// Post registers a POST handler.
func (a *ChiAdapter) Post(pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.HandleFunc(http.MethodPost, pattern, handler, opts...)
}

// Put registers a PUT handler.
func (a *ChiAdapter) Put(pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.HandleFunc(http.MethodPut, pattern, handler, opts...)
}

// Patch registers a PATCH handler.
func (a *ChiAdapter) Patch(pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.HandleFunc(http.MethodPatch, pattern, handler, opts...)
}

// Delete registers a DELETE handler.
func (a *ChiAdapter) Delete(pattern string, handler http.HandlerFunc, opts ...autodoc.HandleOption) {
	a.HandleFunc(http.MethodDelete, pattern, handler, opts...)
}

// Group creates a sub-router with a common prefix.
func (a *ChiAdapter) Group(prefix string, fn func(a *ChiAdapter)) {
	r := chi.NewRouter()
	fn(&ChiAdapter{router: r, doc: a.doc, prefix: a.currentPrefix(prefix)})
	a.router.Mount(prefix, r)
}

func (a *ChiAdapter) currentPrefix(prefix string) string {
	if a.prefix == "" {
		return prefix
	}
	return a.prefix + prefix
}

// Mount registers the doc endpoints on the router.
func (a *ChiAdapter) Mount() {
	if !a.doc.IsEnabled() {
		return
	}

	if p := a.doc.GetSpecPath(); p != "" {
		a.router.Get(p, a.doc.ServeSpec)
	}
	if p := a.doc.GetDocsPath(); p != "" {
		a.router.Get(p, a.doc.ServeSwaggerUI)
	}
	if p := a.doc.GetReDocPath(); p != "" {
		a.router.Get(p, a.doc.ServeReDoc)
	}
}
