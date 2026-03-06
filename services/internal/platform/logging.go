package platform

import (
	"context"
	"log/slog"
	"os"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/trace"
)

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func NewLoggingInterceptor(logger *slog.Logger) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			span := trace.SpanFromContext(ctx)
			sc := span.SpanContext()
			requestLogger := logger.With(
				"trace_id", sc.TraceID().String(),
				"span_id", sc.SpanID().String(),
				"method", req.Spec().Procedure,
			)
			requestLogger.InfoContext(ctx, "request started")
			resp, err := next(ctx, req)
			if err != nil {
				requestLogger.ErrorContext(ctx, "request failed", "error", err, "code", connect.CodeOf(err).String())
				return nil, err
			}
			requestLogger.InfoContext(ctx, "request finished")
			return resp, nil
		}
	})
}
