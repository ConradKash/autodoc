package autodoc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/ConradKash/autodoc"
)

// ─── Schema generator tests ───────────────────────────────────────────────────

type Address struct {
	Street string `json:"street" validate:"required,min=1"`
	City   string `json:"city"   validate:"required"`
	Zip    string `json:"zip"    jsonschema:"pattern=^[0-9]{5}$"`
}

type CreateOrderRequest struct {
	ProductID string  `json:"productId" validate:"required,uuid"`
	Quantity  int     `json:"quantity"  validate:"required,gte=1,lte=100"`
	Notes     string  `json:"notes,omitempty" validate:"max=500"`
	ShipTo    Address `json:"shipTo"    validate:"required"`
}

type OrderResponse struct {
	ID        string    `json:"id"`
	ProductID string    `json:"productId"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"createdAt"`
}

func TestSpec_ContainsPaths(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "Test API", Version: "1.0.0"})

	doc.Register("GET /users")
	doc.Register("POST /users",
		autodoc.WithRequestOf[CreateOrderRequest](),
		autodoc.WithResponseOf[OrderResponse](),
		autodoc.WithStatusCode(201),
	)
	doc.Register("GET /users/{id}")
	doc.Register("DELETE /users/{id}")

	spec := doc.Spec()

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("spec.paths must be a map")
	}

	for _, path := range []string{"/users", "/users/{id}"} {
		if _, found := paths[path]; !found {
			t.Errorf("expected path %q in spec", path)
		}
	}

	// Check /users has both get and post.
	usersItem, ok := paths["/users"].(map[string]interface{})
	if !ok {
		t.Fatal("paths[/users] is not a map")
	}
	if _, ok := usersItem["get"]; !ok {
		t.Error("expected GET on /users")
	}
	if _, ok := usersItem["post"]; !ok {
		t.Error("expected POST on /users")
	}
}

func TestSpec_RequestBodySchema(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "Test", Version: "1"})
	doc.Register("POST /orders",
		autodoc.WithRequestOf[CreateOrderRequest](),
		autodoc.WithResponseOf[OrderResponse](),
		autodoc.WithStatusCode(201),
	)

	spec := doc.Spec()

	// Check components.schemas contains CreateOrderRequest.
	components, _ := spec["components"].(map[string]interface{})
	schemas, _ := components["schemas"].(map[string]interface{})

	reqSchema, ok := schemas["CreateOrderRequest"]
	if !ok {
		t.Fatalf("expected CreateOrderRequest in schemas, got keys: %v", schemaKeys(schemas))
	}

	reqMap, _ := reqSchema.(map[string]interface{})
	props, _ := reqMap["properties"].(map[string]interface{})

	// quantity should have validate constraints applied.
	qty, ok := props["quantity"].(map[string]interface{})
	if !ok {
		t.Fatal("expected quantity property")
	}
	if qty["minimum"] != float64(1) {
		t.Errorf("quantity.minimum: want 1.0, got %v", qty["minimum"])
	}
	if qty["maximum"] != float64(100) {
		t.Errorf("quantity.maximum: want 100.0, got %v", qty["maximum"])
	}

	// productId should have uuid format.
	pid, ok := props["productId"].(map[string]interface{})
	if !ok {
		t.Fatal("expected productId property")
	}
	if pid["format"] != "uuid" {
		t.Errorf("productId.format: want 'uuid', got %v", pid["format"])
	}

	// shipTo should be a $ref.
	shipTo, ok := props["shipTo"].(map[string]interface{})
	if !ok {
		t.Fatal("expected shipTo property")
	}
	if _, hasRef := shipTo["$ref"]; !hasRef {
		t.Error("expected shipTo to be a $ref to Address")
	}
}

func TestSpec_PathParameters(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("GET /api/v1/orders/{id}")

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	item, _ := paths["/api/v1/orders/{id}"].(map[string]interface{})
	getOp, _ := item["get"].(map[string]interface{})

	params, _ := getOp["parameters"].([]interface{})
	if len(params) == 0 {
		t.Fatal("expected path parameters")
	}
	p, _ := params[0].(map[string]interface{})
	if p["name"] != "id" {
		t.Errorf("want param name 'id', got %v", p["name"])
	}
	if p["in"] != "path" {
		t.Errorf("want param in 'path', got %v", p["in"])
	}
	if p["required"] != true {
		t.Error("path param should be required")
	}
}

func TestSpec_ResponseCodes(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("POST /items",
		autodoc.WithStatusCode(201),
		autodoc.WithErrorCodes(409, 422),
	)

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	item, _ := paths["/items"].(map[string]interface{})
	postOp, _ := item["post"].(map[string]interface{})
	responses, _ := postOp["responses"].(map[string]interface{})

	for _, code := range []string{"201", "400", "401", "403", "404", "409", "422", "500"} {
		if _, ok := responses[code]; !ok {
			t.Errorf("expected response code %s in operation", code)
		}
	}
}

func TestSpec_Tags(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("GET /api/v1/products", autodoc.WithTags("products"))
	doc.Register("GET /api/v1/users", autodoc.WithTags("users"))

	spec := doc.Spec()
	tags, _ := spec["tags"].([]map[string]interface{})
	tagNames := map[string]bool{}
	for _, t := range tags {
		tagNames[t["name"].(string)] = true
	}
	for _, want := range []string{"products", "users"} {
		if !tagNames[want] {
			t.Errorf("expected tag %q in spec", want)
		}
	}
}

func TestSpecEndpoint(t *testing.T) {
	doc := autodoc.New(autodoc.Config{
		Title:    "HTTP Test",
		Version:  "1.0.0",
		SpecPath: "/openapi.json",
	})
	doc.Register("GET /ping", autodoc.WithSummary("Ping"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	doc.Mount(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("unexpected content-type: %s", ct)
	}

	var spec map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("want openapi=3.0.3, got %v", spec["openapi"])
	}
}

func TestSwaggerUIEndpoint(t *testing.T) {
	doc := autodoc.New(autodoc.Config{
		Title:    "UI Test",
		DocsPath: "/docs",
	})
	mux := http.NewServeMux()
	doc.Mount(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/docs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

func TestMiddleware_AutoDetect(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "MW Test", Version: "1"})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := doc.Middleware(inner)

	// Simulate two requests.
	for _, path := range []string{"/things", "/other"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	if _, ok := paths["/things"]; !ok {
		t.Error("middleware should have auto-detected /things")
	}
	if _, ok := paths["/other"]; !ok {
		t.Error("middleware should have auto-detected /other")
	}
}

func TestStdMuxAdapter(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "Adapter Test", Version: "1"})
	mux := autodoc.NewStdMux(doc)

	mux.HandleFunc("GET /widgets", func(w http.ResponseWriter, r *http.Request) {},
		autodoc.WithResponseOf[OrderResponse](),
	)
	mux.HandleFunc("POST /widgets", func(w http.ResponseWriter, r *http.Request) {},
		autodoc.WithRequestOf[CreateOrderRequest](),
		autodoc.WithResponseOf[OrderResponse](),
		autodoc.WithStatusCode(201),
	)
	mux.Mount()

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	if _, ok := paths["/widgets"]; !ok {
		t.Error("expected /widgets in spec")
	}
}

func TestWithQueryParam(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("GET /search",
		autodoc.WithQueryParam("q", "Search query", true),
		autodoc.WithQueryParam("limit", "Max results", false),
	)

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	item, _ := paths["/search"].(map[string]interface{})
	getOp, _ := item["get"].(map[string]interface{})
	params, _ := getOp["parameters"].([]interface{})

	found := map[string]bool{}
	for _, p := range params {
		pm, _ := p.(map[string]interface{})
		found[pm["name"].(string)] = true
	}
	if !found["q"] || !found["limit"] {
		t.Errorf("expected q and limit params, found: %v", found)
	}
}

func TestOperationIDGeneration(t *testing.T) {
	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("GET /api/v1/users/{id}/orders")

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})
	item, _ := paths["/api/v1/users/{id}/orders"].(map[string]interface{})
	getOp, _ := item["get"].(map[string]interface{})

	opID, _ := getOp["operationId"].(string)
	if opID == "" {
		t.Error("expected non-empty operationId")
	}
	t.Logf("operationId: %s", opID)
}

func TestExcludePaths(t *testing.T) {
	doc := autodoc.New(autodoc.Config{
		Title:        "T",
		Version:      "1",
		ExcludePaths: []string{"/internal"},
		SpecPath:     "/openapi.json",
		DocsPath:     "/docs",
	})

	doc.Register("GET /users")
	doc.Register("GET /internal/admin") // should be excluded
	doc.Register("GET /openapi.json")   // should be excluded

	spec := doc.Spec()
	paths, _ := spec["paths"].(map[string]interface{})

	if _, ok := paths["/users"]; !ok {
		t.Error("expected /users in spec")
	}
	if _, ok := paths["/internal/admin"]; ok {
		t.Error("/internal/admin should be excluded")
	}
	if _, ok := paths["/openapi.json"]; ok {
		t.Error("/openapi.json should be excluded")
	}
}

func TestSchema_NullablePointer(t *testing.T) {
	type WithPointer struct {
		Required string  `json:"required"    validate:"required"`
		Optional *string `json:"optional,omitempty"`
	}

	doc := autodoc.New(autodoc.Config{Title: "T", Version: "1"})
	doc.Register("POST /test",
		autodoc.WithRequestType(reflect.TypeOf(WithPointer{})),
	)

	spec := doc.Spec()
	components, _ := spec["components"].(map[string]interface{})
	schemas, _ := components["schemas"].(map[string]interface{})
	ws, ok := schemas["WithPointer"].(map[string]interface{})
	if !ok {
		t.Fatal("expected WithPointer schema")
	}
	props, _ := ws["properties"].(map[string]interface{})
	optProp, _ := props["optional"].(map[string]interface{})
	if optProp["nullable"] != true {
		t.Error("pointer field should be marked nullable")
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func schemaKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
