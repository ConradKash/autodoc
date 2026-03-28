package autodoc

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
)

// ─── Schema Generator ─────────────────────────────────────────────────────────

var (
	timeType    = reflect.TypeOf(time.Time{})
	jsonRawType = reflect.TypeOf(json.RawMessage{})
)

// schemaGen generates JSON Schema (OpenAPI 3.0 style) from reflect.Type.
// It accumulates definitions in a shared schemas map.
type schemaGen struct {
	mu       sync.Mutex
	names    map[reflect.Type]string // type → stable schema name
	inFlight map[reflect.Type]bool   // cycle detection
}

func newSchemaGen() *schemaGen {
	return &schemaGen{
		names:    make(map[reflect.Type]string),
		inFlight: make(map[reflect.Type]bool),
	}
}

// schemaRef returns either an inline schema or a $ref to components/schemas.
// All named object schemas are added to the shared `schemas` map.
func (g *schemaGen) schemaRef(t reflect.Type, schemas map[string]interface{}) map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buildRef(t, schemas)
}

func (g *schemaGen) buildRef(t reflect.Type, schemas map[string]interface{}) map[string]interface{} {
	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Named struct → $ref.
	if t.Kind() == reflect.Struct && t != timeType && t != jsonRawType {
		name := g.typeName(t)
		if !g.inFlight[t] {
			g.inFlight[t] = true
			s := g.structSchema(t, schemas)
			schemas[name] = s
			g.inFlight[t] = false
		}
		return map[string]interface{}{"$ref": "#/components/schemas/" + name}
	}

	return g.inlineSchema(t, schemas)
}

func (g *schemaGen) inlineSchema(t reflect.Type, schemas map[string]interface{}) map[string]interface{} {
	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == timeType {
		return map[string]interface{}{"type": "string", "format": "date-time"}
	}
	if t == jsonRawType {
		return map[string]interface{}{}
	}

	switch t.Kind() {
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]interface{}{"type": "integer", "format": "int32"}
	case reflect.Int64:
		return map[string]interface{}{"type": "integer", "format": "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]interface{}{"type": "integer", "minimum": 0, "maximum": math.MaxUint32}
	case reflect.Uint64:
		return map[string]interface{}{"type": "integer", "minimum": 0}
	case reflect.Float32:
		return map[string]interface{}{"type": "number", "format": "float"}
	case reflect.Float64:
		return map[string]interface{}{"type": "number", "format": "double"}
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]interface{}{"type": "string", "format": "byte"}
		}
		return map[string]interface{}{
			"type":  "array",
			"items": g.buildRef(t.Elem(), schemas),
		}
	case reflect.Array:
		s := map[string]interface{}{
			"type":     "array",
			"items":    g.buildRef(t.Elem(), schemas),
			"minItems": t.Len(),
			"maxItems": t.Len(),
		}
		return s
	case reflect.Map:
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": g.buildRef(t.Elem(), schemas),
		}
	case reflect.Struct:
		name := g.typeName(t)
		if !g.inFlight[t] {
			g.inFlight[t] = true
			s := g.structSchema(t, schemas)
			schemas[name] = s
			g.inFlight[t] = false
		}
		return map[string]interface{}{"$ref": "#/components/schemas/" + name}
	case reflect.Interface:
		return map[string]interface{}{} // any
	default:
		return map[string]interface{}{}
	}
}

func (g *schemaGen) structSchema(t reflect.Type, schemas map[string]interface{}) map[string]interface{} {
	props := map[string]interface{}{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Embedded struct: flatten.
		if field.Anonymous {
			ft := field.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded := g.structSchema(ft, schemas)
				if ep, ok := embedded["properties"].(map[string]interface{}); ok {
					for k, v := range ep {
						props[k] = v
					}
				}
				if er, ok := embedded["required"].([]string); ok {
					required = append(required, er...)
				}
			}
			continue
		}

		jsonName, omitempty := jsonTag(field)
		if jsonName == "-" {
			continue
		}

		fieldSchema := g.inlineSchema(field.Type, schemas)
		applyFieldTags(fieldSchema, field)

		// Required logic:
		// 1. validate:"required" tag → required
		// 2. Non-pointer, non-omitempty → required
		// 3. Pointer or omitempty → optional
		isReq := isRequiredField(field, omitempty)
		if isReq {
			required = appendUniq(required, jsonName)
		}

		// Mark nullable if pointer.
		if field.Type.Kind() == reflect.Ptr {
			fieldSchema["nullable"] = true
		}

		props[jsonName] = fieldSchema
	}

	s := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		s["required"] = required
	}

	// Description from doc comment (not available via reflection; use title).
	if t.Name() != "" {
		s["title"] = splitCamel(t.Name())
	}

	return s
}

// ─── Tag parsing ──────────────────────────────────────────────────────────────

func jsonTag(f reflect.StructField) (name string, omitempty bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = f.Name
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

// applyFieldTags enriches a schema with constraints from struct tags:
//   - `jsonschema:"..."` — goapi-style tags
//   - `validate:"..."` — go-playground/validator tags
//   - `binding:"..."` — gin binding tags
func applyFieldTags(s map[string]interface{}, f reflect.StructField) {
	applyJSONSchemaTag(s, f.Tag.Get("jsonschema"))
	applyValidateTag(s, f.Tag.Get("validate"))
	applyValidateTag(s, f.Tag.Get("binding")) // gin uses the same syntax
	applyExampleTag(s, f.Tag.Get("example"))
	applyDescriptionTag(s, f.Tag.Get("description"))
}

func applyJSONSchemaTag(s map[string]interface{}, tag string) {
	if tag == "" {
		return
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(kv[0])
		var val string
		if len(kv) > 1 {
			val = strings.TrimSpace(kv[1])
		}

		switch key {
		case "title":
			s["title"] = val
		case "description":
			s["description"] = val
		case "format":
			s["format"] = val
		case "pattern":
			s["pattern"] = val
		case "enum":
			parts := strings.Split(val, "|")
			enums := make([]interface{}, len(parts))
			for i, p := range parts {
				enums[i] = strings.TrimSpace(p)
			}
			s["enum"] = enums
		case "default":
			s["default"] = val
		case "example":
			s["example"] = val
		case "minLength":
			s["minLength"] = parseInt(val)
		case "maxLength":
			s["maxLength"] = parseInt(val)
		case "minimum":
			s["minimum"] = parseFloat(val)
		case "maximum":
			s["maximum"] = parseFloat(val)
		case "exclusiveMinimum":
			s["exclusiveMinimum"] = parseFloat(val)
		case "exclusiveMaximum":
			s["exclusiveMaximum"] = parseFloat(val)
		case "multipleOf":
			s["multipleOf"] = parseFloat(val)
		case "minItems":
			s["minItems"] = parseInt(val)
		case "maxItems":
			s["maxItems"] = parseInt(val)
		case "uniqueItems":
			s["uniqueItems"] = true
		case "minProperties":
			s["minProperties"] = parseInt(val)
		case "maxProperties":
			s["maxProperties"] = parseInt(val)
		case "readOnly":
			s["readOnly"] = true
		case "writeOnly":
			s["writeOnly"] = true
		case "deprecated":
			s["deprecated"] = true
		case "nullable":
			s["nullable"] = true
		case "required":
			// handled at struct level
		}
	}
}

// applyValidateTag parses go-playground/validator tags for schema constraints.
func applyValidateTag(s map[string]interface{}, tag string) {
	if tag == "" {
		return
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "required":
			// handled at struct level
		case part == "email":
			s["format"] = "email"
		case part == "url" || part == "uri":
			s["format"] = "uri"
		case part == "uuid" || part == "uuid4":
			s["format"] = "uuid"
		case part == "datetime":
			s["format"] = "date-time"
		case part == "date":
			s["format"] = "date"
		case part == "alpha":
			s["pattern"] = "^[a-zA-Z]+$"
		case part == "alphanum":
			s["pattern"] = "^[a-zA-Z0-9]+$"
		case part == "numeric":
			s["pattern"] = "^[0-9]+$"
		case part == "lowercase":
			s["pattern"] = "^[a-z]+$"
		case part == "uppercase":
			s["pattern"] = "^[A-Z]+$"
		case strings.HasPrefix(part, "min="):
			v := strings.TrimPrefix(part, "min=")
			if t, ok := s["type"].(string); ok && t == "string" {
				s["minLength"] = parseInt(v)
			} else {
				s["minimum"] = parseFloat(v)
			}
		case strings.HasPrefix(part, "max="):
			v := strings.TrimPrefix(part, "max=")
			if t, ok := s["type"].(string); ok && t == "string" {
				s["maxLength"] = parseInt(v)
			} else {
				s["maximum"] = parseFloat(v)
			}
		case strings.HasPrefix(part, "len="):
			v := strings.TrimPrefix(part, "len=")
			n := parseInt(v)
			s["minLength"] = n
			s["maxLength"] = n
		case strings.HasPrefix(part, "oneof="):
			v := strings.TrimPrefix(part, "oneof=")
			parts := strings.Fields(v)
			enums := make([]interface{}, len(parts))
			for i, p := range parts {
				enums[i] = p
			}
			s["enum"] = enums
		case strings.HasPrefix(part, "gt="):
			s["exclusiveMinimum"] = parseFloat(strings.TrimPrefix(part, "gt="))
		case strings.HasPrefix(part, "gte="):
			s["minimum"] = parseFloat(strings.TrimPrefix(part, "gte="))
		case strings.HasPrefix(part, "lt="):
			s["exclusiveMaximum"] = parseFloat(strings.TrimPrefix(part, "lt="))
		case strings.HasPrefix(part, "lte="):
			s["maximum"] = parseFloat(strings.TrimPrefix(part, "lte="))
		case strings.HasPrefix(part, "dive"):
			// handled by array item generation
		}
	}
}

func applyExampleTag(s map[string]interface{}, tag string) {
	if tag != "" {
		s["example"] = tag
	}
}

func applyDescriptionTag(s map[string]interface{}, tag string) {
	if tag != "" {
		s["description"] = tag
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (g *schemaGen) typeName(t reflect.Type) string {
	if name, ok := g.names[t]; ok {
		return name
	}
	name := t.Name()
	if name == "" {
		name = "Anonymous"
	}
	// Disambiguate same-name types from different packages.
	base := name
	for i := 2; ; i++ {
		found := false
		for _, n := range g.names {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			break
		}
		name = fmt.Sprintf("%s%d", base, i)
	}
	g.names[t] = name
	return name
}

func isRequiredField(f reflect.StructField, omitempty bool) bool {
	// validate:"required" or binding:"required" → required.
	for _, tag := range []string{"validate", "binding"} {
		for _, part := range strings.Split(f.Tag.Get(tag), ",") {
			if strings.TrimSpace(part) == "required" {
				return true
			}
		}
	}
	// jsonschema:"required" → required.
	for _, part := range strings.Split(f.Tag.Get("jsonschema"), ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	// Pointer or omitempty → optional.
	if f.Type.Kind() == reflect.Ptr || omitempty {
		return false
	}
	return true
}

func appendUniq(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// splitCamel splits "CreateUserRequest" → "Create User Request".
func splitCamel(s string) string {
	var out strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out.WriteByte(' ')
		}
		out.WriteRune(r)
	}
	return out.String()
}
