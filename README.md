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
 go get -u github.com/ConradKash/autodoc@latest
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

autodoc supports the Gin framework via the `autodoc/gin` subpackage:

### Quick Example

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/ConradKash/autodoc"
    autodocgin "github.com/ConradKash/autodoc/gin"
)

doc := autodoc.New(autodoc.Config{Title: "Gin API", Version: "1.0.0"})
ginAdapter := autodocgin.NewGinAdapter(doc)

ginAdapter.Handle("GET", "/users", func(c *gin.Context) {
    // handler code
}, autodoc.WithResponseOf[UserResponse]())

ginAdapter.Handle("POST", "/users", func(c *gin.Context) {
    // handler code
}, autodoc.WithRequestOf[CreateUserRequest](), autodoc.WithResponseOf[UserResponse]())

ginAdapter.Mount()

ginAdapter.Engine.Run(":8080")
```

### Usage Notes
- Use `ginAdapter.Handle` or `ginAdapter.HandleFunc` to register routes and document them.
- Path parameters like `:id` are automatically converted to OpenAPI `{id}` syntax in the generated spec.
- Call `ginAdapter.Mount()` to serve `/docs`, `/docs/redoc`, and `/openapi.json` endpoints.
- All autodoc options (e.g., `WithSummary`, `WithRequestOf`, `WithResponseOf`) are supported.
- You can use the returned `ginAdapter.Engine` as your main Gin engine.

### Advanced: Using an Existing gin.Engine

If you want to use an existing `gin.Engine`:

```go
engine := gin.Default()
doc := autodoc.New(autodoc.Config{Title: "Gin API", Version: "1.0.0"})
ginAdapter := autodocgin.NewGinAdapter(doc)
ginAdapter.Engine = engine // use your own engine
// Register routes as above
// ...
ginAdapter.Mount()
engine.Run(":8080")
```

### Type-Safe Documentation

You can use autodoc's type-safe helpers for request/response types:

```go
ginAdapter.Handle("POST", "/users", createUserHandler,
    autodoc.WithRequestOf[CreateUserRequest](),
    autodoc.WithResponseOf[UserResponse](),
)
```

### OpenAPI Path Parameters

Gin's `:param` syntax is automatically converted to OpenAPI `{param}` in the generated spec. For example, `/users/:id` becomes `/users/{id}`.

### Serving Docs

- `/docs` — Swagger UI
- `/docs/redoc` — ReDoc
- `/openapi.json` — Raw OpenAPI 3.0 spec

You can customize these paths via the `autodoc.Config`.

## Chi Framework Integration

autodoc supports the chi router via the `autodoc/chi` subpackage:

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/ConradKash/autodoc"
    autodochi "github.com/ConradKash/autodoc/chi"
)

router := chi.NewRouter()
doc := autodoc.New(autodoc.Config{Title: "Chi API", Version: "1.0.0"})
chiAdapter := autodochi.NewChiAdapter(doc, router)

chiAdapter.Get("/users", listUsers, autodoc.WithSummary("List users"))
chiAdapter.Post("/users", createUser, autodoc.WithRequestOf[CreateUserRequest]())
chiAdapter.Get("/users/{id}", getUser)

chiAdapter.Mount()

http.ListenAndServe(":8080", router)
```

## Build-Time Code Generation

autodoc includes a `go:generate` CLI that scans your router registration code at build time and embeds a static OpenAPI spec — zero runtime cost.

### Install the CLI

```bash
go install github.com/ConradKash/autodoc/cmd/autodoc-gen
```

### Add a generate directive

In any `.go` file in your project:

```go
//go:generate autodoc-gen -router=chi -out=docs_gen.go -spec=openapi.json
```

Then run:

```bash
go generate ./...
```

This parses your `r.Get()`, `r.Post()`, `r.Method()` calls (or `mux.HandleFunc("GET /path", ...)` for stdlib) and generates a Go source file with the embedded spec and a static route table.

### CLI flags

| Flag | Description | Default |
|------|-------------|---------|
| `-router` | Router type: `chi` or `http` | *(required)* |
| `-out` | Output Go file | `docs_gen.go` |
| `-spec` | Output OpenAPI JSON file | *(none)* |
| `-pkg` | Package name | auto-detected |
| `-title` | API title | `API` |
| `-version` | API version | `1.0.0` |
| `-scan` | Comma-separated dirs to scan | `.` |
| `-docs` | Swagger UI path | `/docs` |
| `-spec-path` | OpenAPI spec path | `/openapi.json` |

### Gin Build-Time Code Generation

autodoc's generator also supports Gin projects. You can generate a static OpenAPI spec and route table from your Gin route registrations.

**Add a generate directive:**

```go
//go:generate autodoc-gen -router=gin -out=docs_gen.go -spec=openapi.json
```

Then run:

```bash
go generate ./...
```

- This will scan your Gin route registrations (e.g., `r.GET("/users/:id", ...)`, `r.POST(...)`, etc.).
- Gin’s `:param` syntax is automatically converted to OpenAPI `{param}` in the generated spec.
- The generated Go file will contain the embedded OpenAPI spec and a static route table for your Gin API.

**CLI flags are the same as for chi/http.**

## Contributing

Contributions are welcome! Please open issues or pull requests on GitHub. Ensure your code is tested (`go test ./...`) and follows idiomatic Go style.

## License

MIT License — see [LICENSE](LICENSE) for details.

## Support

For questions or support, open an issue on the GitHub repository.

---

## Recent Changes

### Chi build-time docs and compilation fixes

**Structural change — adapters moved to subpackages**

The `GinAdapter` and `ChiAdapter` were previously in `package autodoc` (the root), which meant importing the core library pulled in chi, gin, and all their transitive dependencies. They are now in separate subpackages:

- `github.com/ConradKash/autodoc/chi` — chi adapter (pulls in chi only)
- `github.com/ConradKash/autodoc/gin` — gin adapter (pulls in gin only)

The core `github.com/ConradKash/autodoc` package has **zero external dependencies**.

**Exported API on AutoDoc**

The following methods were added to support adapter subpackages:

- `ServeSpec(w, r)` — serves the OpenAPI JSON spec
- `ServeSwaggerUI(w, r)` — serves the Swagger UI page
- `ServeReDoc(w, r)` — serves the ReDoc page
- `GetTitle()`, `GetSpecPath()`, `GetDocsPath()`, `GetReDocPath()`, `IsEnabled()` — config accessors

The HTML template constants were also exported:

- `autodoc.SwaggerUIHTML`
- `autodoc.RedocHTML`

**Enhanced chi codegen**

The build-time code generator (`autodoc-gen`) now handles all chi route registration patterns:

- `r.Get("/path", h)`, `r.Post(...)`, `r.Put(...)`, `r.Delete(...)`, `r.Patch(...)`, `r.Head(...)`, `r.Options(...)`
- `r.Method("GET", "/path", h)` and `r.MethodFunc("GET", "/path", h)`
- `r.Handle("/path", h)` and `r.HandleFunc("/path", h)` (catch-all, defaults to GET)

**Compilation fixes**

- `cmd/autodoc-gen/main.go`: merged duplicate import blocks; fixed `scanDirs` variable name collision (`*string` reassigned to `[]string`)
- Replaced deprecated `ioutil.WriteFile` with `os.WriteFile` in `codegen.go`
