package transport

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func logsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Info("request",
			zap.String("Method", r.Method),
			zap.String("URL", r.URL.String()),
		)

		start := time.Now()
		next(w, r)

		duration := time.Since(start)
		zap.L().Info("response",
			zap.String("Method", r.Method),
			zap.String("URL", r.URL.String()),
			zap.Duration("completion time", duration),
		)
	})
}
