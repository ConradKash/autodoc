package autodoc

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

type swaggerUITemplateData struct {
	Title               string
	FaviconHref         string
	SwaggerUICSS        string
	SwaggerUIBundle     string
	SwaggerUIStandalone string
	SchemaURL           string
}

func TestSwaggerUITemplate_AllFields(t *testing.T) {
	tmpl, err := template.New("swaggerui").Parse(SwaggerUITemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	data := swaggerUITemplateData{
		Title:               "Test API",
		FaviconHref:         "/favicon.ico",
		SwaggerUICSS:        "/swagger-ui.css",
		SwaggerUIBundle:     "/swagger-ui-bundle.js",
		SwaggerUIStandalone: "/swagger-ui-standalone-preset.js",
		SchemaURL:           "/api/schema",
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, data.Title) {
		t.Errorf("output missing title")
	}
	if !strings.Contains(out, data.FaviconHref) {
		t.Errorf("output missing favicon href")
	}
	if !strings.Contains(out, data.SwaggerUICSS) {
		t.Errorf("output missing swagger ui css")
	}
	if !strings.Contains(out, data.SwaggerUIBundle) {
		t.Errorf("output missing swagger ui bundle")
	}
	if !strings.Contains(out, data.SwaggerUIStandalone) {
		t.Errorf("output missing swagger ui standalone")
	}
	jsEscapedSchemaURL := strings.ReplaceAll(data.SchemaURL, "/", "\\/")
	if !strings.Contains(out, jsEscapedSchemaURL) {
		t.Errorf("output missing schema url: want JS-escaped %q, got output = %q", jsEscapedSchemaURL, out)
	}
}

func TestSwaggerUITemplate_EmptyFavicon(t *testing.T) {
	tmpl, err := template.New("swaggerui").Parse(SwaggerUITemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	data := swaggerUITemplateData{
		Title:               "Test API",
		FaviconHref:         "",
		SwaggerUICSS:        "/swagger-ui.css",
		SwaggerUIBundle:     "/swagger-ui-bundle.js",
		SwaggerUIStandalone: "/swagger-ui-standalone-preset.js",
		SchemaURL:           "/api/schema",
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "rel=\"icon\"") {
		t.Errorf("favicon link should not be present when FaviconHref is empty")
	}
}

func TestSwaggerUITemplate_Compiles(t *testing.T) {
	_, err := template.New("swaggerui").Parse(SwaggerUITemplate)
	if err != nil {
		t.Fatalf("template failed to compile: %v", err)
	}
}
