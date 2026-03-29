# autodoc

Automatic OpenAPI 3.0 Documentation for Go HTTP APIs

---

autodoc is a Go library that provides **automatic OpenAPI 3.0 documentation generation** for Go HTTP handlers—**no annotations, no comments, no code generation required**. Inspired by Python's drf-spectacular, simply mount autodoc and your entire API is documented instantly. Swagger UI and ReDoc are served out of the box.

## Features
- **Zero-annotation**: No need to add comments or tags to your handlers.
- **Automatic OpenAPI 3.0 spec**: Reflects on your registered routes and types.
- **Swagger UI & ReDoc**: Interactive API docs at `/docs` and `/docs/redoc`.
- **JSON Schema generation**: Supports struct tags, validation, and more.
- **Flexible integration**: Works with `http.ServeMux` and custom routers.
- **Middleware support**: Auto-discovers routes by observing traffic.
- **Customizable**: Add summaries, descriptions, tags, security, and more.

## Installation

```
go get autodoc
```

## Quick Start

```go
mux := http.NewServeMux()
doc := autodoc.New(autodoc.Config{
    Title:   "Acme API",
    Version: "1.0.0",
})
doc.Mount(mux) // registers /docs, /docs/redoc, /openapi.json
doc.Handle(mux, "GET /users", listUsersHandler)
doc.HandleTyped(mux, "POST /users", createUserHandler,
    autodoc.WithRequestType(reflect.TypeOf(CreateUserRequest{})),
    autodoc.WithResponseType(reflect.TypeOf(UserResponse{})),
)
http.ListenAndServe(":8080", mux)
```

- Visit `/docs` for Swagger UI, `/docs/redoc` for ReDoc, `/openapi.json` for the raw spec.

## Usage

- **Register routes**: Use `doc.Handle`, `doc.HandleFunc`, or `doc.Register` to add routes.
- **Type-safe docs**: Use `WithRequestOf[T]()` and `WithResponseOf[T]()` for typed handlers.
- **Scan existing mux**: Use `autodoc.ScanMux(doc, mux)` to auto-discover all registered routes.
- **Middleware**: Wrap your handler with `doc.Middleware` to auto-detect routes from traffic.
- **Customize**: Add summaries, descriptions, tags, security schemes, and more via config and options.

See [autodoc.go](autodoc.go) and [autodoc_test.go](autodoc_test.go) for advanced usage and examples.

## Gin Framework Integration

autodoc now supports the Gin framework via `GinAdapter`:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/ConradKash/autodoc"
)

doc := autodoc.New(autodoc.Config{Title: "Gin API", Version: "1.0.0"})
ginAdapter := autodoc.NewGinAdapter(doc)

ginAdapter.Handle("GET", "/users", func(c *gin.Context) {
    // handler code
}, autodoc.WithResponseOf[UserResponse]())

ginAdapter.Handle("POST", "/users", func(c *gin.Context) {
    // handler code
}, autodoc.WithRequestOf[CreateUserRequest](), autodoc.WithResponseOf[UserResponse]())

ginAdapter.Mount()

ginAdapter.Engine.Run(":8080")
```

This will auto-document all registered Gin routes, just like with http.ServeMux.

## Contributing

Contributions are welcome! Please open issues or pull requests on GitHub. Ensure your code is tested (`go test ./...`) and follows idiomatic Go style.

## License

Specify your license here (e.g., MIT, Apache 2.0).

## Support

For questions or support, open an issue on the GitHub repository.
