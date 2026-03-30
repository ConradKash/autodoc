package autodoc

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"
)

func TestConvertGinPatternToOAS(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"/users/:id", "/users/{id}"},
		{"/foo/:bar/baz/:qux", "/foo/{bar}/baz/{qux}"},
		{"/static/path", "/static/path"},
		{"/a/:b_c123", "/a/{b_c123}"},
		{"/x/:y/:z", "/x/{y}/{z}"},
	}
	for _, c := range cases {
		got := convertGinPatternToOAS(c.in)
		if got != c.out {
			t.Errorf("convertGinPatternToOAS(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestParseGinRoute(t *testing.T) {
	cg := &CodeGen{}

	// Helper to build a CallExpr for r.GET("/users/:id", handler)
	makeCall := func(method, path string) *ast.CallExpr {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				Sel: &ast.Ident{Name: method},
			},
			Args: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: `"` + path + `"`},
				&ast.Ident{Name: "handler"},
			},
		}
	}

	cases := []struct {
		method string
		path   string
		expect *Route
	}{
		{"GET", "/users/:id", &Route{Method: "GET", Path: "/users/{id}", OperationID: buildOperationID("GET", "/users/{id}"), StatusCode: 200, Tags: []string{"users"}}},
		{"POST", "/foo/:bar/baz", &Route{Method: "POST", Path: "/foo/{bar}/baz", OperationID: buildOperationID("POST", "/foo/{bar}/baz"), StatusCode: 200, Tags: []string{"foo"}}},
		{"PUT", "/static/path", &Route{Method: "PUT", Path: "/static/path", OperationID: buildOperationID("PUT", "/static/path"), StatusCode: 200, Tags: []string{"static"}}},
		{"PATCH", "/a/:b_c123", &Route{Method: "PATCH", Path: "/a/{b_c123}", OperationID: buildOperationID("PATCH", "/a/{b_c123}"), StatusCode: 200, Tags: []string{"a"}}},
	}

	for _, c := range cases {
		call := makeCall(c.method, c.path)
		got := cg.parseGinRoute(c.method, call)
		if !reflect.DeepEqual(got, c.expect) {
			t.Errorf("parseGinRoute(%q, %q) = %+v, want %+v", c.method, c.path, got, c.expect)
		}
	}

	// Test unsupported method
	call := makeCall("FOOBAR", "/x/:y")
	if cg.parseGinRoute("FOOBAR", call) != nil {
		t.Error("parseGinRoute should return nil for unsupported method")
	}

	// Test missing/invalid args
	call = &ast.CallExpr{Fun: &ast.SelectorExpr{Sel: &ast.Ident{Name: "GET"}}, Args: []ast.Expr{}}
	if cg.parseGinRoute("GET", call) != nil {
		t.Error("parseGinRoute should return nil for missing args")
	}
	call = &ast.CallExpr{Fun: &ast.SelectorExpr{Sel: &ast.Ident{Name: "GET"}}, Args: []ast.Expr{&ast.Ident{Name: "not_a_string"}, &ast.Ident{Name: "handler"}}}
	if cg.parseGinRoute("GET", call) != nil {
		t.Error("parseGinRoute should return nil for non-string path arg")
	}
}
