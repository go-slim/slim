package slim

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"zestack.dev/slim/nego"
	"zestack.dev/slim/serde"
)

// HandlerFunc defines a function to serve HTTP requests.
type HandlerFunc func(c Context) error

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(c Context, next HandlerFunc) error

// MiddlewareRegistrar 中间件注册接口
type MiddlewareRegistrar interface {
	// Use 注册中间件
	Use(middleware ...MiddlewareFunc)
	// Middleware 返回注册的所有中间件
	Middleware() []MiddlewareFunc
}

// MiddlewareComposer 中间件合成器接口
type MiddlewareComposer interface {
	// Compose 将注册的所有中间件合并成一个中间件
	Compose() MiddlewareFunc
}

// ErrorHandler is a centralized error handler.
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(c Context, err error)
}

// ErrorHandlerFunc defines a function to centralize errors.
type ErrorHandlerFunc func(c Context, err error)

// HandleError 实现 ErrorHandler 接口
func (h ErrorHandlerFunc) HandleError(c Context, err error) {
	h(c, err)
}

// ErrorHandlerRegistrar 错误处理器注册接口
type ErrorHandlerRegistrar interface {
	// UseErrorHandler 注册错误处理器
	// 重复调用该方法会覆盖之前设置的错误处理器
	UseErrorHandler(h ErrorHandler)
}

type MiddlewareConfigurator interface {
	// ToMiddleware 将实例转换成中间件函数
	ToMiddleware() MiddlewareFunc
}

// Validator is the interface that wraps the Validate function.
type Validator interface {
	Validate(i any) error
}

// IPExtractor is a function to extract IP addr from http.Request.
// Set appropriate one to Slim.IPExtractor.
type IPExtractor func(*http.Request) string

// Map defines a generic map of type `map[string]any`.
type Map map[string]any

type RouterCreator func(*Slim) Router

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "slim context value " + k.name
}

var (
	SlimContextKey     = &contextKey{"slim"}
	RequestContextKey  = &contextKey{"request"}
	ResponseContextKey = &contextKey{"response"}
	ContextKey         = &contextKey{"context"}
)

type Slim struct {
	// router 默认路由
	router Router
	// routers 虚拟主机（Virtual Hosting）表，是对虚拟主机的简单实现，
	// 支持实域名和泛域名两种模式，当请求的域名不在此表内时使用 Slim.router，
	// 所以其优先级高于 Slim.router。
	routers map[string]Router
	// routerCreator 创建自定义路由
	routerCreator RouterCreator
	// contextPool 网络请求上下文管理池
	contextPool sync.Pool
	// contextPathParamAllocSize 上下文中参数的最大数量
	contextPathParamAllocSize int
	// middleware 中间件列表
	middleware []MiddlewareFunc

	negotiator *nego.Negotiator

	// TrustedPlatform if set to a constant of value gin.Platform*, trusts the headers set by
	// that platform, for example to determine the client IP
	TrustedPlatform string

	NewContextFunc       func(pathParamAllocSize int) EditableContext // 自定义 `slim.Context` 构造函数
	ErrorHandler         ErrorHandlerFunc
	Filesystem           fs.FS // 静态资源文件系统，默认值 `os.DirFS(".")`。
	Binder               Binder
	Validator            Validator
	Renderer             Renderer // 自定义模板渲染器
	JSONSerializer       serde.Serializer
	XMLSerializer        serde.Serializer
	Logger               *log.Logger
	Debug                bool     // 是否开启调试模式
	MultipartMemoryLimit int64    // 文件上传大小限制
	PrettyIndent         string   // json/xml 格式化缩进
	JSONPCallbacks       []string // jsonp 回调函数
	IPExtractor          IPExtractor
}

func Classic() *Slim {
	s := New()
	s.Use(Logging())
	s.Use(Recovery())
	s.Use(Static("public"))
	return s
}

func New() *Slim {
	s := &Slim{
		routers:              make(map[string]Router),
		negotiator:           nego.New(10, nil),
		NewContextFunc:       nil,
		ErrorHandler:         DefaultErrorHandler,
		Filesystem:           os.DirFS("."),
		Binder:               &DefaultBinder{},
		Validator:            nil,
		Renderer:             nil,
		JSONSerializer:       serde.JSONSerializer{},
		XMLSerializer:        serde.XMLSerializer{},
		Debug:                true,
		MultipartMemoryLimit: 32 << 20, // 32 MB
		PrettyIndent:         "  ",
		JSONPCallbacks:       []string{"jsonp", "callback"},
	}
	s.router = s.NewRouter()
	s.contextPool.New = func() any {
		if s.NewContextFunc != nil {
			return s.NewContextFunc(s.contextPathParamAllocSize)
		}
		return s.NewContext(nil, nil)
	}
	return s
}

func (s *Slim) NewContext(w http.ResponseWriter, r *http.Request) Context {
	p := make(PathParams, s.contextPathParamAllocSize)
	c := &contextImpl{
		request:       r,
		response:      nil,
		allowsMethods: make([]string, 0),
		store:         make(Map),
		slim:          s,
		pathParams:    &p,
		matchType:     RouteMatchUnknown,
		route:         nil,
	}
	if w != nil && r != nil {
		c.response = NewResponseWriter(r.Method, w)
	}
	return c
}

func (s *Slim) NewRouter() Router {
	var r Router
	if s.routerCreator != nil {
		r = s.routerCreator(s)
	} else {
		r = NewRouter(RouterConfig{})
	}
	if x, ok := r.(*routerImpl); ok {
		x.slim = s
	}
	return r
}

// Router 返回默认路由器
func (s *Slim) Router() Router {
	return s.router
}

// Routers 返回 vhost 的 `host => router` 映射
func (s *Slim) Routers() map[string]Router {
	return s.routers
}

// RouterFor 返回与指定 `host` 相关的路由器
func (s *Slim) RouterFor(host string) Router {
	return s.routers[host]
}

// ResetRouterCreator 重置路由器创建函数。
// 注意：会立即重新创建默认路由器，并且 vhost 路由器会被清除。
func (s *Slim) ResetRouterCreator(creator func(s *Slim) Router) {
	s.routerCreator = creator
	s.router = s.NewRouter()
	clear(s.routers)
}

// Use adds middleware to the chain which is run before router.
func (s *Slim) Use(middleware ...MiddlewareFunc) {
	s.middleware = append(s.middleware, middleware...)
}

// Host 通过提供名称和中间件函数创建对应 `host` 的路由器实例
func (s *Slim) Host(name string, middleware ...MiddlewareFunc) Router {
	router := s.NewRouter()
	router.Use(middleware...)
	s.routers[name] = router
	return router
}

// Group 实现路由分组注册，实际调用 `RouteCollector.Route` 实现
func (s *Slim) Group(fn func(sub RouteCollector)) {
	s.router.Group(fn)
}

// Route 以指定前缀实现路由分组注册
func (s *Slim) Route(prefix string, fn func(sub RouteCollector)) {
	s.router.Route(prefix, fn)
}

// Some registers a new route for multiple HTTP methods and path with matching
// handler in the router. Panics on error.
func (s *Slim) Some(methods []string, pattern string, h HandlerFunc) Route {
	return s.router.Some(methods, pattern, h)
}

// Any registers a new route for all supported HTTP methods and path with matching
// handler in the router. Panics on error.
func (s *Slim) Any(pattern string, h HandlerFunc) Route {
	return s.router.Any(pattern, h)
}

// CONNECT registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) CONNECT(path string, h HandlerFunc) Route {
	return s.router.CONNECT(path, h)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Slim) DELETE(path string, h HandlerFunc) Route {
	return s.router.DELETE(path, h)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Slim) GET(path string, h HandlerFunc) Route {
	return s.router.GET(path, h)
}

// HEAD registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) HEAD(path string, h HandlerFunc) Route {
	return s.router.HEAD(path, h)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) OPTIONS(path string, h HandlerFunc) Route {
	return s.router.OPTIONS(path, h)
}

// PATCH registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) PATCH(path string, h HandlerFunc) Route {
	return s.router.PATCH(path, h)
}

// POST registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) POST(path string, h HandlerFunc) Route {
	return s.router.POST(path, h)
}

// PUT registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) PUT(path string, h HandlerFunc) Route {
	return s.router.PUT(path, h)
}

// TRACE registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (s *Slim) TRACE(path string, h HandlerFunc) Route {
	return s.router.TRACE(path, h)
}

// Static registers a new route with path prefix to serve static files
// from the provided root directory. Panics on error.
func (s *Slim) Static(prefix, root string) Route {
	return s.router.Static(prefix, root)
}

// File registers a new route with a path to serve a static file.
// Panics on error.
func (s *Slim) File(path, file string) Route {
	return s.router.File(path, file)
}

// URI generates a URI from handler.
// In case when Slim serves multiple hosts/domains use `s.Routers()["domain2.site"].Reverse()` to get specific host URL.
func (s *Slim) URI(h HandlerFunc, params ...any) string {
	return s.router.URI(h, params...)
}

// Reverse generates a URL from route name and provided parameters.
// In case when Slim serves multiple hosts/domains use `s.Routers()["domain2.site"].Reverse()` to get specific host URL.
func (s *Slim) Reverse(name string, params ...any) string {
	return s.router.Reverse(name, params...)
}

// Routes returns the registered routes for default router.
// In case when Slim serves multiple hosts/domains use `s.Routers()["domain2.site"].Routes()` to get specific host routes.
func (s *Slim) Routes() []Route {
	return s.router.Routes()
}

// Negotiator 返回内容协商工具
func (s *Slim) Negotiator() *nego.Negotiator {
	return s.negotiator
}

// SetNegotiator 设置自定义内容协商工具
func (s *Slim) SetNegotiator(negotiator *nego.Negotiator) {
	s.negotiator = negotiator
}

// AcquireContext returns an empty `Context` instance from the pool.
// You must return the context by calling `ReleaseContext()`.
func (s *Slim) AcquireContext() Context {
	return s.contextPool.Get().(Context)
}

// ReleaseContext returns the `Context` instance back to the pool.
// You must call it after `AcquireContext()`.
func (s *Slim) ReleaseContext(c Context) {
	s.contextPool.Put(c)
}

// ServeHTTP implements `http.Handler` interface, which serves HTTP requests.
func (s *Slim) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
	c := s.AcquireContext().(EditableContext)
	c.Reset(w, r)

	// Execute chain
	mw := Compose(s.middleware...)
	var err error
	if mw == nil {
		router := s.findRouterByRequest(r)
		err = s.findHandler(c, router)(c)
	} else {
		err = mw(c, func(cc Context) error {
			router := s.findRouterByRequest(r)
			return s.findHandler(c, router)(cc)
		})
	}

	// Handle error
	if err != nil {
		s.handleError(c, err)
	}

	// Release context
	s.ReleaseContext(c)
}

// findRouterByRequest 通过 `*http.Request` 实例获取对应的路由器
func (s *Slim) findRouterByRequest(r *http.Request) Router {
	if len(s.routers) == 0 {
		return s.router
	}

	// 在正常情况下，我们是通过如负载均衡服务器来反向代理我们的程序实现对外服务的，
	// 所以反向代理的域名或端口号可能会与处理请求的源头服务器有所不同，在这种情况下，
	// 可以使用报头 X-Forwarded-Host 用来确定哪一个域名是最初被用来访问的。
	// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/X-Forwarded-Host
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		// 报头 X-Forwarded-Host 不属于任何一份既有规范，所以有可能无法获取到数据，
		// 此时根据 RFC 7239 标准定义的另外一个报头 Forwarded 来来获取包含代理服务器的
		// 客户端的信息；在这里，为什么 RFC 标准滞后于 X-Forwarded-Host 的原因是
		// 由于后者已经成为既成标准了。
		// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/Forwarded
		if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
			for _, forwardedPair := range strings.Split(forwarded, ";") {
				if tv := strings.SplitN(forwardedPair, "=", 2); len(tv) == 2 {
					token, value := tv[0], tv[1]
					token = strings.TrimSpace(token)
					value = strings.TrimSpace(strings.Trim(value, `"`))
					if strings.ToLower(token) == "host" {
						host = value
						break
					}
				}
			}
		}

		if host == "" {
			host = r.Host
		}
	}

	return s.findRouter(host)
}

// findRouter 根据 host 查找路由器
// Note: 调用该方法前，需要将参数转换成小写形式。
func (s *Slim) findRouter(host string) Router {
	if len(s.routers) > 0 && strings.Contains(host, ".") && host != "." {
		// 优先使用完全匹配来查找，如：
		// * 实域名 blog.example.com；
		// * 泛域名 *.example.com。
		if router, ok := s.routers[host]; ok {
			return router
		}

		// 我们只支持简单形式的 host 表达式（如二级域名 *.example.com 或
		// 三级域名 *.foo.example.com 等形式，不支持复杂的如 *.*.example.com 这类
		// 的），所以对于已经是泛域名的，就是用默认路由器。
		if host[:2] == "*." {
			goto fallback
		}

		i := strings.IndexByte(host, '.')
		j := strings.LastIndexByte(host, '.')
		// 参数 host 至少是一个二级域名才行，所以对于非域名或
		// 一级域名，我们同样采用默认路由器。
		if i == -1 || i == j {
			goto fallback
		}

		// 将 host 转化成 *.example.com 或 *.foo.example.com 的形式，
		// 然后到虚拟主机表里面查询关联的路由器。
		if router, ok := s.routers["*."+host[j:]]; ok {
			return router
		}
	}

fallback:
	// 如果没有注册虚拟主机，就返回默认路由就可以了，
	// 所以对于非 SASS 系统，尽量不启用虚拟主机功能。
	return s.router
}

func (s *Slim) findHandler(c EditableContext, router Router) HandlerFunc {
	r := c.Request()
	params := c.RawPathParams()
	match := router.Match(r, params)
	c.SetRawPathParams(params)
	c.SetAllowsMethods(match.AllowMethods)
	c.SetRouteInfo(match.RouteInfo)
	c.SetRouteMatchType(match.Type)
	if i, ok := c.(interface{ SetRouter(Router) }); ok {
		i.SetRouter(router)
	}
	mw := router.Compose()
	if mw != nil {
		return func(c Context) error {
			return mw(c, match.Handler)
		}
	}
	return match.Handler
}

// handleError 处理路由执行错误
func (s *Slim) handleError(c Context, err error) {
	if err == nil {
		return
	}

	// FIXME: 这里有点问题需要考虑：
	//  如果错误发生在中间件中，就不需要后续的 RouterCollector 的
	//  错误处理器来处理，而应该提交给上级来处理。
	if info := c.RouteInfo(); info != nil {
		// 优先使用路由收集器中定义的错误处理器处理错误
		collector := info.Collector()
		for collector != nil {
			if i, ok := collector.(ErrorHandler); ok {
				i.HandleError(c, err)
				return
			}
			collector = collector.Parent()
		}
		// 路由器中定义的错误处理器次之
		router := info.Router()
		if i, ok := router.(ErrorHandler); ok {
			i.HandleError(c, err)
			return
		}
	}

	// 最后使用上下文的错误处理器。
	c.Error(err)
}

// WrapHandler wraps `http.Handler` into `slim.HandlerFunc`.
func WrapHandler(h http.Handler) HandlerFunc {
	return func(c Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// WrapHandlerFunc wraps `http.HandlerFunc` into `slim.HandlerFunc`.
func WrapHandlerFunc(h http.HandlerFunc) HandlerFunc {
	return func(c Context) error {
		h(c.Response(), c.Request())
		return nil
	}
}

// WrapMiddleware wraps `func(http.Handler) http.Handler` into `slim.MiddlewareFunc`
func WrapMiddleware(m func(http.Handler) http.Handler) MiddlewareFunc {
	return func(c Context, next HandlerFunc) (err error) {
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.SetRequest(r)
			c.SetResponse(NewResponseWriter(r.Method, w))
			err = next(c)
		})).ServeHTTP(c.Response(), c.Request())
		return
	}
}

func Tap(h HandlerFunc, mw ...MiddlewareFunc) HandlerFunc {
	if len(mw) == 0 {
		return h
	}
	return func(c Context) error {
		return Compose(mw...)(c, h)
	}
}

// DefaultErrorHandler 默认错误处理函数
func DefaultErrorHandler(c Context, err error) {
	if c.Written() {
		if c.Slim().Debug {
			c.Logger().Println(err.Error())
		}
		return
	}
	// TODO(hupeh): 根据 Accept 报头返回对应的格式
	if errors.Is(err, ErrNotFound) {
		http.NotFound(c.Response(), c.Request())
	} else if errors.Is(err, ErrMethodNotAllowed) {
		c.SetHeader("Allow", c.AllowsMethods()...)
		http.Error(c.Response(), http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	} else {
		http.Error(c.Response(), err.Error(), http.StatusInternalServerError)
	}
}

func NotFoundHandler(_ Context) error {
	return ErrNotFound
}

func MethodNotAllowedHandler(_ Context) error {
	return ErrMethodNotAllowed
}
