package autodoc

// swaggerUIHTML is a complete Swagger UI page loaded from the official CDN.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{TITLE}} — API Docs</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" sizes="32x32">
  <style>
    *, *::before, *::after { box-sizing: border-box; }

    :root {
      --bg:      #0d0f14;
      --surface: #161b27;
      --border:  #1f2a3c;
      --accent:  #38bdf8;
      --accent2: #818cf8;
      --text:    #e2e8f0;
      --muted:   #64748b;
      --success: #34d399;
      --danger:  #f87171;
      --warn:    #fbbf24;
      --patch:   #a78bfa;
    }

    html { scroll-behavior: smooth; }

    body {
      margin: 0;
      background: var(--bg);
      color: var(--text);
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    }

    /* ── Top bar ── */
    .autodoc-header {
      position: sticky;
      top: 0;
      z-index: 100;
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 0.75rem 2rem;
      background: rgba(13,15,20,0.85);
      backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--border);
    }
    .autodoc-header .logo {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      font-weight: 700;
      font-size: 1rem;
      letter-spacing: -0.02em;
      color: var(--text);
      text-decoration: none;
    }
    .autodoc-header .logo svg { flex-shrink: 0; }
    .autodoc-header .badge {
      margin-left: auto;
      font-size: 0.7rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      padding: 0.2rem 0.55rem;
      border-radius: 9999px;
      border: 1px solid var(--border);
      color: var(--muted);
    }
    .autodoc-header .view-switch {
      display: flex;
      gap: 0.5rem;
    }
    .autodoc-header .view-switch a {
      font-size: 0.78rem;
      padding: 0.3rem 0.7rem;
      border-radius: 6px;
      border: 1px solid var(--border);
      color: var(--muted);
      text-decoration: none;
      transition: all 0.15s;
    }
    .autodoc-header .view-switch a:hover,
    .autodoc-header .view-switch a.active {
      border-color: var(--accent);
      color: var(--accent);
      background: rgba(56,189,248,0.07);
    }

    /* ── Swagger container ── */
    #swagger-ui { max-width: 1280px; margin: 0 auto; padding: 2rem 1.5rem 4rem; }

    /* ── Override swagger-ui dark theme ── */
    .swagger-ui { color: var(--text); }
    .swagger-ui .info { margin: 1.5rem 0 2rem; }
    .swagger-ui .info .title { color: var(--text); font-size: 2rem; font-weight: 800; }
    .swagger-ui .info .description { color: var(--muted); }
    .swagger-ui .info a { color: var(--accent); }
    .swagger-ui .scheme-container,
    .swagger-ui .wrapper { background: transparent; box-shadow: none; }
    .swagger-ui .opblock-tag {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 10px;
      margin-bottom: 0.5rem;
      font-size: 1rem;
      font-weight: 700;
      color: var(--text);
    }
    .swagger-ui .opblock-tag:hover { border-color: var(--accent); }
    .swagger-ui .opblock {
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--surface);
      margin-bottom: 0.4rem;
      box-shadow: none;
    }
    .swagger-ui .opblock .opblock-summary {
      border-bottom: none;
      padding: 0.6rem 1rem;
    }
    .swagger-ui .opblock.is-open { border-color: var(--accent); }
    .swagger-ui .opblock .opblock-summary-path {
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
      font-size: 0.88rem;
      color: var(--text);
    }
    .swagger-ui .opblock .opblock-summary-description { color: var(--muted); font-size: 0.82rem; }

    /* Method badges */
    .swagger-ui .opblock-summary-method {
      border-radius: 5px;
      font-size: 0.7rem;
      font-weight: 800;
      letter-spacing: 0.05em;
      min-width: 62px;
      text-align: center;
      padding: 0.25rem 0.5rem;
    }
    .swagger-ui .opblock.opblock-get    .opblock-summary-method { background: var(--accent);  color: #0c1a2e; }
    .swagger-ui .opblock.opblock-post   .opblock-summary-method { background: var(--success); color: #022c22; }
    .swagger-ui .opblock.opblock-put    .opblock-summary-method { background: var(--warn);    color: #1c1003; }
    .swagger-ui .opblock.opblock-patch  .opblock-summary-method { background: var(--patch);   color: #1e1030; }
    .swagger-ui .opblock.opblock-delete .opblock-summary-method { background: var(--danger);  color: #1a0505; }

    /* Expanded operation body */
    .swagger-ui .opblock-body { background: rgba(0,0,0,0.15); padding: 1rem; border-radius: 0 0 7px 7px; }
    .swagger-ui .tab li { color: var(--muted); font-size: 0.82rem; }
    .swagger-ui .tab li.active { color: var(--accent); border-bottom: 2px solid var(--accent); }
    .swagger-ui table { border-collapse: collapse; width: 100%; }
    .swagger-ui table thead tr { background: rgba(255,255,255,0.03); }
    .swagger-ui table thead tr th { color: var(--muted); font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.07em; border-bottom: 1px solid var(--border); padding: 0.5rem 0.75rem; }
    .swagger-ui table tbody tr td { border-bottom: 1px solid var(--border); padding: 0.5rem 0.75rem; font-size: 0.84rem; }
    .swagger-ui .parameter__name { color: var(--text); font-family: monospace; }
    .swagger-ui .parameter__type { color: var(--accent2); font-size: 0.78rem; }
    .swagger-ui .required::after { content: " *"; color: var(--danger); }
    .swagger-ui .body-param__text,
    .swagger-ui textarea { background: #0a0c10 !important; color: var(--text) !important; border: 1px solid var(--border) !important; border-radius: 6px; font-family: 'JetBrains Mono', monospace; font-size: 0.82rem; }
    .swagger-ui select { background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 6px; }
    .swagger-ui input[type=text] { background: #0a0c10; color: var(--text); border: 1px solid var(--border); border-radius: 6px; padding: 0.35rem 0.6rem; }
    .swagger-ui .execute-wrapper .btn.execute { background: var(--accent); color: #0c1a2e; font-weight: 700; border: none; border-radius: 6px; padding: 0.45rem 1.2rem; cursor: pointer; transition: opacity 0.15s; }
    .swagger-ui .execute-wrapper .btn.execute:hover { opacity: 0.85; }
    .swagger-ui .btn.cancel { background: transparent; border: 1px solid var(--danger); color: var(--danger); border-radius: 6px; }
    .swagger-ui .responses-wrapper { background: transparent; }
    .swagger-ui .response-col_status { color: var(--success); font-weight: 700; }
    .swagger-ui .response-col_description { color: var(--muted); font-size: 0.82rem; }
    .swagger-ui .highlight-code pre { background: #0a0c10 !important; border-radius: 6px; }
    .swagger-ui code, .swagger-ui pre { font-family: 'JetBrains Mono', 'Fira Code', monospace !important; }
    .swagger-ui .model-box { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; }
    .swagger-ui .model { color: var(--text); }
    .swagger-ui .model .property { color: var(--muted); }
    .swagger-ui .model-title { color: var(--accent2); font-weight: 700; }
    .swagger-ui .servers > label { color: var(--muted); font-size: 0.82rem; }
    .swagger-ui .servers select { margin-left: 0.5rem; }
    .swagger-ui section.models { border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
    .swagger-ui section.models h4 { color: var(--text); }
    .swagger-ui .arrow { fill: var(--muted); }
    .swagger-ui svg { fill: currentColor; }
    .swagger-ui .topbar { display: none; }
    .swagger-ui .info .base-url { color: var(--muted); font-size: 0.82rem; }
    .swagger-ui .auth-wrapper .authorize { background: var(--accent); color: #0c1a2e; border-color: var(--accent); }
    .swagger-ui .auth-wrapper .authorize svg { fill: #0c1a2e; }
    .swagger-ui .auth-container { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; color: var(--text); }
    .swagger-ui .dialog-ux .modal-ux { background: var(--surface); border: 1px solid var(--border); color: var(--text); }
    .swagger-ui .dialog-ux .modal-ux-header { border-bottom: 1px solid var(--border); }
    .swagger-ui .dialog-ux .modal-ux-header h3 { color: var(--text); }
    .swagger-ui .dialog-ux .modal-ux-content { background: var(--surface); }
    .swagger-ui .loading-container .loading::after { color: var(--muted); }
    .swagger-ui .filter .operation-filter-input { background: var(--surface); border: 1px solid var(--border); color: var(--text); border-radius: 6px; }
  </style>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
  <header class="autodoc-header">
    <a class="logo" href="#">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="#38bdf8" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      {{SPEC_URL}}
    </a>
    <span class="badge">OpenAPI 3.0</span>
    <div class="view-switch">
      <a href="#" class="active">Swagger</a>
      <a href="redoc">ReDoc</a>
      <a href="{{SPEC_URL}}" target="_blank">JSON</a>
    </div>
  </header>

  <div id="swagger-ui"></div>

  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "{{SPEC_URL}}",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        plugins: [SwaggerUIBundle.plugins.DownloadUrl],
        layout: "StandaloneLayout",
        displayRequestDuration: true,
        filter: true,
        tryItOutEnabled: true,
        persistAuthorization: true,
        syntaxHighlight: { activate: true, theme: "monokai" },
        requestInterceptor: function(request) {
          return request;
        }
      });
    };
  </script>
</body>
</html>`

// redocHTML is a full ReDoc page.
const redocHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{TITLE}} — API Reference</title>
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5/favicon-32x32.png" sizes="32x32">
  <style>
    *, *::before, *::after { box-sizing: border-box; }
    :root {
      --bg:      #0d0f14;
      --surface: #161b27;
      --border:  #1f2a3c;
      --accent:  #38bdf8;
      --text:    #e2e8f0;
      --muted:   #64748b;
    }
    html, body { margin: 0; padding: 0; background: var(--bg); }

    .autodoc-header {
      position: sticky;
      top: 0;
      z-index: 1000;
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 0.75rem 2rem;
      background: rgba(13,15,20,0.9);
      backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--border);
    }
    .autodoc-header .logo {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      font-weight: 700;
      font-size: 1rem;
      color: var(--text);
      text-decoration: none;
      font-family: 'Inter', sans-serif;
    }
    .autodoc-header .badge {
      margin-left: auto;
      font-size: 0.7rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      padding: 0.2rem 0.55rem;
      border-radius: 9999px;
      border: 1px solid var(--border);
      color: var(--muted);
      font-family: 'Inter', sans-serif;
    }
    .autodoc-header .view-switch {
      display: flex;
      gap: 0.5rem;
    }
    .autodoc-header .view-switch a {
      font-size: 0.78rem;
      padding: 0.3rem 0.7rem;
      border-radius: 6px;
      border: 1px solid var(--border);
      color: var(--muted);
      text-decoration: none;
      font-family: 'Inter', sans-serif;
    }
    .autodoc-header .view-switch a:hover,
    .autodoc-header .view-switch a.active {
      border-color: var(--accent);
      color: var(--accent);
      background: rgba(56,189,248,0.07);
    }
    #redoc-container { padding-top: 0; }
  </style>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
</head>
<body>
  <header class="autodoc-header">
    <a class="logo" href="#">
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="#38bdf8" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      {{SPEC_URL}}
    </a>
    <span class="badge">ReDoc</span>
    <div class="view-switch">
      <a href="../docs">Swagger</a>
      <a href="#" class="active">ReDoc</a>
      <a href="{{SPEC_URL}}" target="_blank">JSON</a>
    </div>
  </header>
  <div id="redoc-container"></div>
  <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
  <script>
    Redoc.init('{{SPEC_URL}}', {
      theme: {
        colors: {
          primary: { main: '#38bdf8' },
          success: { main: '#34d399' },
          warning: { main: '#fbbf24' },
          error:   { main: '#f87171' },
          text: { primary: '#e2e8f0', secondary: '#94a3b8' },
          border: { dark: '#1f2a3c', light: '#1f2a3c' },
          background: '#0d0f14',
        },
        sidebar: {
          backgroundColor: '#0d0f14',
          textColor: '#94a3b8',
        },
        rightPanel: { backgroundColor: '#0a0c10' },
        typography: {
          fontSize:     '14px',
          fontFamily:   "'Inter', -apple-system, sans-serif",
          code: {
            fontSize:   '13px',
            fontFamily: "'JetBrains Mono', monospace",
          },
        },
      },
      hideDownloadButton: false,
      expandResponses: '200,201',
      pathInMiddlePanel: true,
      nativeScrollbars: false,
    }, document.getElementById('redoc-container'));
  </script>
</body>
</html>`
