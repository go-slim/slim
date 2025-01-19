package slim

import (
	"errors"
	"sync/atomic"
)

// Explicitly 一个承上启下的中间件
func Explicitly(c Context, next HandlerFunc) error {
	return next(c)
}

// Compose 将多个中间件合并为一个，在执行期间，会自上而下传递请求，
// 之后过滤并逆序返回响应，因此实现了友好且符合直观思维的洋葱模型。
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
