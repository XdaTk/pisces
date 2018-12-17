package middleware

import (
	"github.com/xdatk/pisces"
)

type (
	// Skipper defines a function to skip middleware. Returning true skips processing
	// the middleware.
	Skipper func(pisces.Context) bool

	// BeforeFunc defines a function which is executed just before the middleware.
	BeforeFunc func(pisces.Context)
)

// DefaultSkipper returns false which processes the middleware.
func DefaultSkipper(pisces.Context) bool {
	return false
}
