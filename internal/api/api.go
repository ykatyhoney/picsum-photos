package api

import (
	"net/http"
	"path"
	"time"

	"github.com/DMarby/picsum-photos/internal/handler"
	"github.com/DMarby/picsum-photos/internal/hmac"
	"github.com/DMarby/picsum-photos/internal/tracing"
	"github.com/rs/cors"

	"github.com/DMarby/picsum-photos/internal/database"
	"github.com/DMarby/picsum-photos/internal/logger"
	"github.com/gorilla/mux"
)

// API is a http api
type API struct {
	Database        database.Provider
	Log             *logger.Logger
	Tracer          *tracing.Tracer
	RootURL         string
	ImageServiceURL string
	StaticPath      string
	HandlerTimeout  time.Duration
	HMAC            *hmac.HMAC
}

// Utility methods for logging
func (a *API) logError(r *http.Request, message string, err error) {
	a.Log.Errorw(message, handler.LogFields(r, "error", err)...)
}

// Router returns a http router
func (a *API) Router() http.Handler {
	router := mux.NewRouter()

	router.NotFoundHandler = handler.Handler(a.notFoundHandler)

	// Redirect trailing slashes
	router.StrictSlash(true)

	// Image list
	router.Handle("/v2/list", handler.Handler(a.listHandler)).Methods("GET").Name("List")

	// Query parameters:
	// ?page={page} - What page to display
	// ?limit={limit} - How many entries to display per page

	// Image routes
	oldRouter := router.PathPrefix("").Subrouter()
	oldRouter.Use(a.deprecatedParams)

	oldRouter.Handle("/{size:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.randomImageRedirectHandler)).Methods("GET").Name("Random image")
	oldRouter.Handle("/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.randomImageRedirectHandler)).Methods("GET").Name("Random image")

	// Image by ID routes
	router.Handle("/id/{id}/{size:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.imageRedirectHandler)).Methods("GET").Name("Image by ID")
	router.Handle("/id/{id}/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.imageRedirectHandler)).Methods("GET").Name("Image by ID")

	// Image info routes
	router.Handle("/id/{id}/info", handler.Handler(a.infoHandler)).Methods("GET").Name("Image info by ID")
	router.Handle("/seed/{id}/info", handler.Handler(a.infoSeedHandler)).Methods("GET").Name("Image info by seed")

	// Image by seed routes
	router.Handle("/seed/{seed}/{size:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.seedImageRedirectHandler)).Methods("GET").Name("Image by seed")
	router.Handle("/seed/{seed}/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.seedImageRedirectHandler)).Methods("GET").Name("Image by seed")

	// Query parameters:
	// ?grayscale - Grayscale the image
	// ?blur - Blur the image
	// ?blur={amount} - Blur the image by {amount}

	// Deprecated query parameters:
	// ?image={id} - Get image by id

	// Deprecated routes
	router.Handle("/list", handler.Handler(a.deprecatedListHandler)).Methods("GET").Name("Deprecated list")
	router.Handle("/g/{size:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.deprecatedImageHandler)).Methods("GET").Name("Deprecated image")
	router.Handle("/g/{width:[0-9]+}/{height:[0-9]+}{extension:(?:\\..*)?}", handler.Handler(a.deprecatedImageHandler)).Methods("GET").Name("Deprecated image")

	// Static files
	router.HandleFunc("/", serveFile(path.Join(a.StaticPath, "index.html"))).Name("Static assets")
	router.HandleFunc("/images", serveFile(path.Join(a.StaticPath, "images.html"))).Name("Static assets")
	router.HandleFunc("/favicon.ico", serveFile(path.Join(a.StaticPath, "assets/images/favicon/favicon.ico"))).Name("Static assets")
	router.PathPrefix("/assets/").HandlerFunc(fileHeaders(http.StripPrefix("/assets/", http.FileServer(http.Dir(path.Join(a.StaticPath, "assets/")))).ServeHTTP)).Name("Static assets")

	// Set up handlers
	cors := cors.New(cors.Options{
		AllowedMethods: []string{"GET"},
		AllowedOrigins: []string{"*"},
	})

	httpHandler := cors.Handler(router)
	httpHandler = handler.Recovery(a.Log, httpHandler)
	httpHandler = http.TimeoutHandler(httpHandler, a.HandlerTimeout, "Something went wrong. Timed out.")
	httpHandler = handler.Logger(a.Log, httpHandler)

	routeMatcher := &handler.MuxRouteMatcher{Router: router}
	httpHandler = handler.Tracer(a.Tracer, httpHandler, routeMatcher)
	httpHandler = handler.Metrics(httpHandler, routeMatcher)

	return httpHandler
}

// Handle not found errors
var notFoundError = &handler.Error{
	Message: "page not found",
	Code:    http.StatusNotFound,
}

func (a *API) notFoundHandler(w http.ResponseWriter, r *http.Request) *handler.Error {
	return notFoundError
}

// Set headers for static file handlers
func fileHeaders(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		handler(w, r)
	}
}

// Serve a static file
func serveFile(name string) func(w http.ResponseWriter, r *http.Request) {
	return fileHeaders(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	})
}
