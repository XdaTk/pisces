package middleware

import (
	"github.com/sirupsen/logrus"
	"github.com/xdatk/pisces"
	"time"
)

type (
	// LoggerConfig defines the config for Logger middleware.
	LoggerConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		Logger *logrus.Logger
	}
)

var (
	// DefaultLoggerConfig is the default Logger middleware config.
	DefaultLoggerConfig = LoggerConfig{
		Skipper: DefaultSkipper,
		Logger:  logrus.New(),
	}
)

// Logger returns a middleware that logs HTTP requests.
func Logger() pisces.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with config.
// See: `Logger()`.
func LoggerWithConfig(config LoggerConfig) pisces.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultLoggerConfig.Skipper
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	return func(next pisces.HandlerFunc) pisces.HandlerFunc {
		return func(c pisces.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()
			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()

			config.Logger.WithFields(logrus.Fields{
				"time":          time.Now().Format(time.RFC3339Nano),
				"id":            res.Header().Get(pisces.HeaderXRequestID),
				"remote_ip":     c.RealIP(),
				"protocol":      req.Proto,
				"host":          req.Host,
				"method":        req.Method,
				"uri":           req.RequestURI,
				"path":          req.URL.Path,
				"referer":       req.Referer(),
				"user_agent":    req.UserAgent(),
				"status":        res.Status,
				"error":         err,
				"latency":       stop.Sub(start),
				"latency_human": stop.Sub(start),
				"bytes_in":      pisces.HeaderContentLength,
				"bytes_out":     res.Size,
			}).Info()
			return
		}
	}
}
