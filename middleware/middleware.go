package middleware

import (
	"context"

	"go-slim.dev/slim"
)

// BeforeFunc defines a function which is executed just before the middleware.
type BeforeFunc func(c slim.Context)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "slim/middleware context value " + k.name
}

func valueIntoContext(c slim.Context, ctxKey, value any) {
	ctx := c.Request().Context()
	ctx = context.WithValue(ctx, ctxKey, value)
	c.SetRequest(c.Request().WithContext(ctx))
}
