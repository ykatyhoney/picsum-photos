package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DMarby/picsum-photos/internal/logger"
	"github.com/DMarby/picsum-photos/internal/tracing"
	"github.com/felixge/httpsnoop"
)

// Logger is a handler that logs requests using Zap
func Logger(log *logger.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respMetrics := httpsnoop.CaptureMetricsFn(w, func(ww http.ResponseWriter) {
			h.ServeHTTP(ww, r)
		})

		ctx := r.Context()
		traceID, spanID := tracing.TraceInfo(ctx)

		logFields := []interface{}{
			"http-method", r.Method,
			"remote-addr", r.RemoteAddr,
			"user-agent", r.UserAgent(),
			"uri", r.URL.String(),
			"status-code", respMetrics.Code,
			"elapsed", fmt.Sprintf("%.9fs", respMetrics.Duration.Seconds()),
		}

		if traceID != "" {
			logFields = append(logFields, "trace-id", traceID, "span-id", spanID)
		}

		// Add context error information if present
		if ctxErr := ctx.Err(); ctxErr != nil {
			logFields = append(logFields, "context-error", ctxErr.Error())
		}

		switch {
		case respMetrics.Code == http.StatusServiceUnavailable && ctx.Err() == context.Canceled:
			// Client disconnected - not an error, just informational
			log.Infow("Request cancelled by client", logFields...)
		case respMetrics.Code == http.StatusServiceUnavailable && ctx.Err() == context.DeadlineExceeded:
			// Handler timeout - this is an error
			log.Errorw("Request timeout", logFields...)
		case respMetrics.Code >= 500:
			// Other 5xx errors
			log.Errorw("Request completed", logFields...)
		default:
			log.Debugw("Request completed", logFields...)
		}
	})
}

// LogFields logs the given keys and values for a request
func LogFields(r *http.Request, keysAndValues ...interface{}) []interface{} {
	ctx := r.Context()
	traceID, spanID := tracing.TraceInfo(ctx)

	if traceID != "" {
		return append([]interface{}{
			"trace-id", traceID,
			"span-id", spanID,
		}, keysAndValues...)
	}
	return keysAndValues
}
