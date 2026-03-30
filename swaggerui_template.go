package autodoc

// SwaggerUITemplate is the Go template for dynamic Swagger UI HTML generation.
const SwaggerUITemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>{{ .Title }}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    {{ if .FaviconHref }}<link rel="icon" href="{{ .FaviconHref }}">{{ end }}
    <link rel="stylesheet" href="{{ .SwaggerUICSS }}">
    <style>
      html { box-sizing: border-box; overflow-y: scroll; }
      *, *:after, *:before { box-sizing: inherit; }
      body { background: #fafafa; margin: 0; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="{{ .SwaggerUIBundle }}"></script>
    <script src="{{ .SwaggerUIStandalone }}"></script>
    <script>
      window.onload = function() {
        window.ui = SwaggerUIBundle({
          url: "{{ .SchemaURL }}",
          dom_id: '#swagger-ui',
          presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIStandalonePreset
          ],
          layout: "StandaloneLayout"
        });
      };
    </script>
  </body>
</html>
`
