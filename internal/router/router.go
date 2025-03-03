package router

import (
	"net/http"

	"meetsync/internal/handlers"
	"meetsync/pkg/logs"
)

// Router handles HTTP routing
type Router struct {
	mux *http.ServeMux
}

// New creates a new Router
func New() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// Setup sets up all routes
func (r *Router) Setup() {
	// Create handlers
	userHandler := handlers.NewUserHandler()
	meetingHandler := handlers.NewMeetingHandler(userHandler)

	// Register user routes
	r.mux.HandleFunc("POST /api/users", userHandler.CreateUser)
	r.mux.HandleFunc("GET /api/users", userHandler.ListUsers)
	r.mux.HandleFunc("GET /api/users/{id}", userHandler.GetUser)

	// Register meeting routes
	r.mux.HandleFunc("POST /api/meetings", meetingHandler.CreateMeeting)
	r.mux.HandleFunc("POST /api/availabilities", meetingHandler.AddAvailability)
	r.mux.HandleFunc("GET /api/recommendations", meetingHandler.GetRecommendations)

	// Serve OpenAPI documentation
	r.mux.HandleFunc("GET /docs", serveOpenAPIUI)
	r.mux.HandleFunc("GET /docs/openapi.yaml", serveOpenAPISpec)

	// Add middleware for logging
	r.mux = r.logMiddleware(r.mux)
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// logMiddleware logs all requests
func (r *Router) logMiddleware(next http.Handler) *http.ServeMux {
	newMux := http.NewServeMux()

	newMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		logs.Info("%s %s %s", req.Method, req.URL.Path, req.RemoteAddr)
		next.ServeHTTP(w, req)
	})

	return newMux
}

// serveOpenAPISpec serves the OpenAPI specification file
func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}

// serveOpenAPIUI serves a simple HTML page that loads Swagger UI
func serveOpenAPIUI(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>MeetSync API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/docs/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "BaseLayout",
                supportedSubmitMethods: []
            });
        };
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
