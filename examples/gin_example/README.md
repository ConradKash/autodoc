# Gin Example for autodoc

This is a minimal Gin web server example to test the autodoc tool.

## How to Run

1. Install dependencies (if not already):
   ```sh
   cd examples/gin_example
   go mod tidy
   ```
2. Run the server:
   ```sh
   go run main.go
   ```
3. Try the endpoints:
   - [GET] http://localhost:8080/ping → `{ "message": "pong" }`
   - [GET] http://localhost:8080/hello/World → `{ "message": "Hello, World!" }`

## How to Test autodoc

1. From the root of the autodoc project, run autodoc against this example:
   ```sh
   # Example command, adjust as needed for your autodoc usage
   autodoc-gen --dir=examples/gin_example
   ```
2. Review the generated documentation.

