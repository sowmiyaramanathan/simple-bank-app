package gapi

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GrpcLogger(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	startTime := time.Now()
	result, err := handler(ctx, req)
	duration := time.Since(startTime)

	statusCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
	}

	logger := log.Info()
	if err != nil {
		logger = log.Error().Err(err)
	}

	logger.Str("protocol", "grpc").Str("method", info.FullMethod).
		Int("status_code", int(statusCode)).Str("status_text", statusCode.String()).
		Dur("duration", duration).Msg("received GRPC request")

	return result, err
}

type ResponseRecorder struct {
	http.ResponseWriter
	int
	Body []byte
}

func (rec *ResponseRecorder) WriteHeader(statusCode int) {
	rec.int = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *ResponseRecorder) Write(body []byte) (int, error) {
	rec.Body = body
	return rec.ResponseWriter.Write(body)
}

func HttpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		rec := &ResponseRecorder{
			ResponseWriter: w,
			int:            http.StatusOK,
		}
		handler.ServeHTTP(rec, r)
		duration := time.Since(startTime)

		logger := log.Info()
		if rec.int != http.StatusOK {
			logger = log.Error().Bytes("body", rec.Body)
		}

		logger.Str("protocol", "http").Str("method", r.Method).Str("path", r.RequestURI).
			Int("status_code", rec.int).Str("status_text", http.StatusText(rec.int)).
			Dur("duration", duration).Msg("received HTTP request")
	})
}
