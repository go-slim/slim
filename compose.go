package slim

import (
	"errors"
	"sync/atomic"
)

// Explicitly a middleware that connects the previous and next
func Explicitly(c Context, next HandlerFunc) error {
	return next(c)
}

// Compose merges multiple middlewares into one. During execution, it passes requests from top to bottom,
// then filters and returns responses in reverse order, thus implementing a friendly and intuitive onion model.
func Compose(middleware ...MiddlewareFunc) MiddlewareFunc {
	l := len(middleware)
	if l == 0 {
		return nil
	}
	if l == 1 {
		return middleware[0]
	}
	var index int32 = -1
	return func(c Context, next HandlerFunc) error {
		var dispatch func(int) error
		dispatch = func(i int) error {
			if int32(i) <= atomic.LoadInt32(&index) {
				return errors.New("slim: next() called multiple times")
			}
			atomic.StoreInt32(&index, int32(i))
			if i == len(middleware) {
				return next(c)
			}
			return middleware[i](c, func(c Context) error {
				return dispatch(i + 1)
			})
		}
		return dispatch(0)
	}
}
