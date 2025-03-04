package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"

	"meetsync/pkg/errors"
	"meetsync/pkg/logs"
)

// ErrorHandler wraps an http.HandlerFunc and provides consistent error handling
type ErrorHandler func(w http.ResponseWriter, r *http.Request) error

// contextKey is a custom type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "requestID"
)

// WithErrorHandling wraps a handler function with error handling
func WithErrorHandling(handler ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID
		requestID := uuid.New().String()
		r = r.WithContext(r.Context())

		// Defer panic recovery
		defer func() {
			if err := recover(); err != nil {
				// Log the stack trace
				logs.Error("PANIC [%s] %v\n%s", requestID, err, debug.Stack())

				// Create an internal server error
				appErr := errors.NewInternalError(
					"An unexpected error occurred",
					fmt.Errorf("panic: %v", err),
				)

				// Write error response
				errors.WriteError(w, appErr)
			}
		}()

		// Call the handler
		err := handler(w, r)
		if err != nil {
			// Log the error with request ID
			if appErr, ok := err.(*errors.AppError); ok {
				if appErr.Type == errors.ErrorTypeInternal {
					logs.Error("[%s] Internal server error: %v", requestID, appErr.Err)
				} else {
					logs.Warn("[%s] Request error: %v", requestID, appErr)
				}
			} else {
				logs.Error("[%s] Unexpected error: %v", requestID, err)
			}

			// Write error response
			errors.WriteError(w, err)
		}
	}
}

// RequestLogger logs incoming requests
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()

		// Log the incoming request
		logs.Info("[%s] %s %s %s", requestID, r.Method, r.URL.Path, r.RemoteAddr)

		// Add request ID to context
		ctx := r.Context()
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Chain combines multiple middleware into a single middleware
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
