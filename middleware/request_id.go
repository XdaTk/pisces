package middleware

import (
	"pisces"
	"pisces/middleware/random"
)

type (
	// RequestIDConfig defines the config for RequestID middleware.
	RequestIDConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// Generator defines a function to generate an ID.
		// Optional. Default value random.String(32).
		Generator func() string
	}
)

var (
	// DefaultRequestIDConfig is the default RequestID middleware config.
	DefaultRequestIDConfig = RequestIDConfig{
		Skipper:   DefaultSkipper,
		Generator: generator,
	}
)

// RequestID returns a X-Request-ID middleware.
func RequestID() pisces.MiddlewareFunc {
	return RequestIDWithConfig(DefaultRequestIDConfig)
}

// RequestIDWithConfig returns a X-Request-ID middleware with config.
func RequestIDWithConfig(config RequestIDConfig) pisces.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultRequestIDConfig.Skipper
	}
	if config.Generator == nil {
		config.Generator = generator
	}

	return func(next pisces.HandlerFunc) pisces.HandlerFunc {
		return func(c pisces.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()
			rid := req.Header.Get(pisces.HeaderXRequestID)
			if rid == "" {
				rid = config.Generator()
			}
			res.Header().Set(pisces.HeaderXRequestID, rid)

			return next(c)
		}
	}
}

func generator() string {
	return random.String(32)
}
