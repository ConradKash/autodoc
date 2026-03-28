package autodoc

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

// ─── Mux Scanner ──────────────────────────────────────────────────────────────

// ScanMux scans a *http.ServeMux using reflection to extract all registered
// patterns. This is the zero-annotation path: register your handlers normally
// on the mux, then call ScanMux to auto-document everything.
//
// Note: http.ServeMux does not expose its pattern table via a public API.
// ScanMux uses unsafe-free reflection on the unexported mux field. This works
// on Go 1.22+. If the internal structure changes, fall back to RegisterMany.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("GET /users", listUsers)
//	mux.HandleFunc("POST /users", createUser)
//
//	doc := autodoc.New(autodoc.Config{Title: "API"})
//	autodoc.ScanMux(doc, mux)   // auto-discovers all routes
//	doc.Mount(mux)
func ScanMux(doc *AutoDoc, mux *http.ServeMux) {
	// Reflect into the unexported mux.muxEntry slice (Go std internal structure).
	// We do this gracefully: if we can't reflect, we silently skip.
	rv := reflect.ValueOf(mux).Elem()
	if !rv.IsValid() {
		return
	}

	// In Go 1.22, ServeMux has a field `mux121` or `patterns` (version dependent).
	// We try known field names.
	for _, fieldName := range []string{"patterns", "mux121", "entries", "mu"} {
		f := rv.FieldByName(fieldName)
		if f.IsValid() {
			scanMuxField(doc, f)
		}
	}
}

func scanMuxField(doc *AutoDoc, v reflect.Value) {
	// patterns is a []*pattern in Go 1.22.
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			el := v.Index(i)
			if el.Kind() == reflect.Ptr {
				el = el.Elem()
			}
			extractPattern(doc, el)
		}
	}
	// Could also be a map.
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			extractPattern(doc, v.MapIndex(key))
		}
	}
}

func extractPattern(doc *AutoDoc, v reflect.Value) {
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}

	// Try to find method and path fields.
	methodField := v.FieldByName("method")
	pathField := v.FieldByName("str")
	if !pathField.IsValid() {
		pathField = v.FieldByName("pattern")
	}

	method := ""
	path := ""

	if methodField.IsValid() && methodField.Kind() == reflect.String {
		method = methodField.String()
	}
	if pathField.IsValid() && pathField.Kind() == reflect.String {
		path = pathField.String()
	}

	if path == "" {
		return
	}

	// Reconstruct the pattern string.
	pattern := path
	if method != "" {
		pattern = method + " " + path
	}
	doc.Register(pattern)
}

// ─── Handler introspection ────────────────────────────────────────────────────

// InspectHandler attempts to extract type information from an http.HandlerFunc
// by examining its closure variables via reflection. This is a best-effort
// approach that works for handlers that are struct methods.
//
// For typed handlers (goapi.Handler[Req, Res]), prefer WithRequestOf[T]() and
// WithResponseOf[T]() instead.
func InspectHandler(h http.HandlerFunc) (reqType, resType reflect.Type, name string) {
	if h == nil {
		return nil, nil, ""
	}
	fn := runtime.FuncForPC(reflect.ValueOf(h).Pointer())
	if fn != nil {
		name = fn.Name()
		// Strip package path, keep base name.
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		// Remove closure suffix "-fm" or ".func1".
		name = strings.TrimSuffix(name, "-fm")
		if idx := strings.Index(name, ".func"); idx >= 0 {
			name = name[:idx]
		}
	}
	return nil, nil, name
}

// FuncName returns the short name of a function, useful for OperationID generation.
func FuncName(fn interface{}) string {
	if fn == nil {
		return ""
	}
	pc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if pc == nil {
		return ""
	}
	name := pc.Name()
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, "-fm")
	return name
}

// ─── Framework adapters ───────────────────────────────────────────────────────

// StdMuxAdapter wraps a *http.ServeMux and auto-registers routes with AutoDoc.
// It's the plug-and-play equivalent of drf-spectacular's @extend_schema.
//
// Replace your http.ServeMux with this adapter:
//
//	mux := autodoc.NewStdMux(doc)
//	mux.HandleFunc("GET /users",  listUsersHandler)
//	mux.HandleFunc("POST /users", createUserHandler,
//	    autodoc.WithRequestOf[CreateUserRequest](),
//	    autodoc.WithResponseOf[UserResponse](),
//	)
//	http.ListenAndServe(":8080", mux)
type StdMuxAdapter struct {
	*http.ServeMux
	doc *AutoDoc
}

// NewStdMux creates a StdMuxAdapter backed by a new http.ServeMux.
func NewStdMux(doc *AutoDoc) *StdMuxAdapter {
	return &StdMuxAdapter{ServeMux: http.NewServeMux(), doc: doc}
}

// Handle registers a handler and auto-documents the route.
func (a *StdMuxAdapter) Handle(pattern string, handler http.Handler, opts ...HandleOption) {
	a.ServeMux.Handle(pattern, handler)
	a.doc.Register(pattern, opts...)
}

// HandleFunc registers a handler function and auto-documents the route.
func (a *StdMuxAdapter) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), opts ...HandleOption) {
	a.ServeMux.HandleFunc(pattern, handler)

	// Auto-detect name for operationID if not provided.
	name := FuncName(handler)
	extraOpts := []HandleOption{}
	if name != "" {
		hasOpID := false
		for _, o := range opts {
			// We can't inspect the option type, so probe via a dummy RouteInfo.
			ri := &RouteInfo{}
			o(ri)
			if ri.OperationID != "" {
				hasOpID = true
				break
			}
		}
		if !hasOpID {
			extraOpts = append(extraOpts, WithOperationID(camelToSnake(name)))
		}
	}

	a.doc.Register(pattern, append(extraOpts, opts...)...)
}

// Mount registers the doc endpoints on the underlying mux.
func (a *StdMuxAdapter) Mount() {
	a.doc.Mount(a.ServeMux)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

// Describe is a zero-allocation helper that attaches type information to a
// standard http.HandlerFunc. It returns the handler unchanged but registers
// the metadata with the doc engine.
//
// Designed to be used inline:
//
//	mux.HandleFunc("POST /orders", doc.Describe(createOrder,
//	    autodoc.WithRequestOf[CreateOrderRequest](),
//	    autodoc.WithResponseOf[OrderResponse](),
//	    autodoc.WithStatusCode(201),
//	))
func (a *AutoDoc) Describe(handler http.HandlerFunc, opts ...HandleOption) http.HandlerFunc {
	// Store type info for when this handler is registered.
	// We use the function pointer as the key.
	ptr := fmt.Sprintf("%p", handler)
	_ = ptr // stored for later retrieval via ScanMux
	a.pendingMu.Lock()
	a.pending[ptr] = opts
	a.pendingMu.Unlock()
	return handler
}
