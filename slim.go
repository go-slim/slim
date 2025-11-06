package slim

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// HandlerFunc defines a function to serve HTTP requests.
type HandlerFunc func(c Context) error

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(c Context, next HandlerFunc) error

// MiddlewareRegistrar middleware registration interface
type MiddlewareRegistrar interface {
	// Use registers middleware
	Use(middleware ...MiddlewareFunc)
	// Middleware returns all registered middleware
	Middleware() []MiddlewareFunc
}

// MiddlewareComposer middleware composer interface
type MiddlewareComposer interface {
	// Compose merges all registered middleware into one middleware
	Compose() MiddlewareFunc
}

// ErrorHandler is a centralized error handler.
type ErrorHandler interface {
	// HandleError handles errors
	HandleError(c Context, err error)
}

// ErrorHandlerFunc defines a function to centralize errors.
type ErrorHandlerFunc func(c Context, err error)

// HandleError implements ErrorHandler interface
func (h ErrorHandlerFunc) HandleError(c Context, err error) {
	h(c, err)
}

// ErrorHandlerRegistrar error handler registration interface
type ErrorHandlerRegistrar interface {
	// UseErrorHandler registers error handler
	// Repeated calls to this method will override the previously set error handler
	UseErrorHandler(h ErrorHandler)
}

type MiddlewareConfigurator interface {
	// ToMiddleware converts instance to middleware function
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

type Slim struct {
	// startupMutex is mutex to lock Server instance access during server configuration and startup. Useful for to get
	// listener address info (on which interface/port was listener bound) without having data races.
	startupMutex sync.RWMutex

	// middleware list
	middleware []MiddlewareFunc

	// router default router
	router Router
	// routers virtual hosting table, is a simple implementation of virtual hosting,
	// supports both real domain and wildcard domain modes, when the requested domain is not in this table, use Slim.router,
	// so its priority is higher than Slim.router.
	routers map[string]Router
	// routerCreator create custom router
	routerCreator RouterCreator

	// contextPool network request context management pool
	contextPool sync.Pool
	// contextPathParamAllocSize maximum number of parameters in context
	contextPathParamAllocSize int

	negotiator *Negotiator

	NewContextFunc       func(pathParamAllocSize int) EditableContext // custom `slim.Context` constructor function
	ErrorHandler         ErrorHandlerFunc
	Filesystem           fs.FS // static resource file system, default value `os.DirFS(".")`.
	Binder               Binder
	Validator            Validator
	Renderer             Renderer // custom template renderer
	JSONCodec            Codec
	XMLCodec             Codec
	Server               *http.Server
	TLSServer            *http.Server
	Listener             net.Listener
	TLSListener          net.Listener
	AutoTLSManager       autocert.Manager
	StdLogger            *log.Logger
	DisableHTTP2         bool
	HideBanner           bool
	HidePort             bool
	ListenerNetwork      string
	Debug                bool     // whether to enable debug mode
	MultipartMemoryLimit int64    // file upload size limit
	PrettyIndent         string   // json/xml formatting indentation
	JSONPCallbacks       []string // jsonp callback functions
	IPExtractor          IPExtractor
}

func New() *Slim {
	s := &Slim{
		routers:              make(map[string]Router),
		negotiator:           NewNegotiator(10, nil),
		Server:               new(http.Server),
		TLSServer:            new(http.Server),
		AutoTLSManager:       autocert.Manager{Prompt: autocert.AcceptTOS},
		ListenerNetwork:      "tcp",
		StdLogger:            log.Default(),
		NewContextFunc:       nil,
		ErrorHandler:         DefaultErrorHandler,
		Filesystem:           os.DirFS("."),
		Binder:               &DefaultBinder{},
		Validator:            nil,
		Renderer:             nil,
		JSONCodec:            JSONCodec{},
		XMLCodec:             XMLCodec{},
		Debug:                true,
		MultipartMemoryLimit: 32 << 20, // 32 MB
		PrettyIndent:         "  ",
		JSONPCallbacks:       []string{"jsonp", "callback"},
	}
	s.Server.Handler = s
	s.TLSServer.Handler = s
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

// Router returns the default router
func (s *Slim) Router() Router {
	return s.router
}

// Routers returns vhost's `host => router` mapping
func (s *Slim) Routers() map[string]Router {
	return s.routers
}

// RouterFor returns the router associated with the specified `host`
func (s *Slim) RouterFor(host string) Router {
	return s.routers[host]
}

// ResetRouterCreator resets the router creator function.
// Note: will immediately recreate the default router, and vhost routers will be cleared.
func (s *Slim) ResetRouterCreator(creator func(s *Slim) Router) {
	s.routerCreator = creator
	s.router = s.NewRouter()
	clear(s.routers)
}

// Use adds middleware to the chain which is run before router.
func (s *Slim) Use(middleware ...MiddlewareFunc) {
	s.middleware = append(s.middleware, middleware...)
}

// Host creates a router instance corresponding to `host` through provided name and middleware functions
func (s *Slim) Host(name string, middleware ...MiddlewareFunc) Router {
	router := s.NewRouter()
	router.Use(middleware...)
	s.routers[name] = router
	return router
}

// Group implements route group registration, actually calls `RouteCollector.Route` to implement
func (s *Slim) Group(fn func(sub RouteCollector)) {
	s.router.Group(fn)
}

// Route implements route group registration with specified prefix
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

// Negotiator returns content negotiation tool
func (s *Slim) Negotiator() *Negotiator {
	return s.negotiator
}

// SetNegotiator sets custom content negotiation tool
func (s *Slim) SetNegotiator(negotiator *Negotiator) {
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

// findRouterByRequest gets the corresponding router through `*http.Request` instance
func (s *Slim) findRouterByRequest(r *http.Request) Router {
	if len(s.routers) == 0 {
		return s.router
	}

	// Under normal circumstances, we use reverse proxy servers like load balancers to reverse proxy our programs to provide external services,
	// so the reverse proxy's domain name or port number may be different from the source server handling the request, in this case,
	// we can use the X-Forwarded-Host header to determine which domain name was originally used for access.
	// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/X-Forwarded-Host
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		// The X-Forwarded-Host header does not belong to any existing specification, so it may not be possible to obtain data,
		// at this time, use another header Forwarded defined by RFC 7239 standard to obtain information containing the proxy server's
		// client information; here, the reason why RFC standard lags behind X-Forwarded-Host is because
		// the latter has already become a de facto standard.
		// https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Headers/Forwarded
		if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
			for forwardedPair := range strings.SplitSeq(forwarded, ";") {
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

// findRouter finds router based on host
// Note: Before calling this method, need to convert the parameter to lowercase.
func (s *Slim) findRouter(host string) Router {
	if len(s.routers) > 0 && strings.Contains(host, ".") && host != "." {
		// Priority is given to exact matching, such as:
		// * Real domain name blog.example.com；
		// * Wildcard domain *.example.com.
		if router, ok := s.routers[host]; ok {
			return router
		}

		// We only support simple forms of host expressions (such as second-level domain *.example.com or
		// third-level domain *.foo.example.com, etc., do not support complex ones like *.*.example.com),
		// so for already wildcard domains, use the default router.
		if host[:2] == "*." {
			goto fallback
		}

		i := strings.IndexByte(host, '.')
		j := strings.LastIndexByte(host, '.')
		// The parameter host must be at least a second-level domain, so for non-domain names or
		// first-level domains, we also use the default router.
		if i == -1 || i == j {
			goto fallback
		}

		// Convert host to the form of *.example.com or *.foo.example.com,
		// then query the virtual host table for associated routers.
		// Note: should use the substring after the first dot to match such as foo.example.com -> *.example.com
		if router, ok := s.routers["*."+host[i+1:]]; ok {
			return router
		}
	}

fallback:
	// If no virtual host is registered, just return the default router,
	// so for non-SAAS systems, try not to enable virtual host functionality.
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

// handleError handles route execution errors
func (s *Slim) handleError(c Context, err error) {
	if err == nil {
		return
	}

	// FIXME: There's an issue to consider here:
	//  If the error occurs in middleware, it doesn't need the subsequent RouterCollector's
	//  error handler to handle it, but should be submitted to the upper level for handling.
	if info := c.RouteInfo(); info != nil {
		// Priority is given to handling errors with error handlers defined in the route collector
		collector := info.Collector()
		for collector != nil {
			if i, ok := collector.(ErrorHandler); ok {
				i.HandleError(c, err)
				return
			}
			collector = collector.Parent()
		}
		// Error handlers defined in the router are secondary
		router := info.Router()
		if i, ok := router.(ErrorHandler); ok {
			i.HandleError(c, err)
			return
		}
	}

	// Finally use the context's error handler.
	c.Error(err)
}

// Start starts an HTTP server.
func (s *Slim) Start(address string) error {
	s.startupMutex.Lock()
	s.Server.Addr = address
	if err := s.configureServer(s.Server); err != nil {
		s.startupMutex.Unlock()
		return err
	}
	s.startupMutex.Unlock()
	return s.Server.Serve(s.Listener)
}

// StartTLS starts an HTTPS server.
// If `certFile` or `keyFile` is `string`, the values are treated as file paths.
// If `certFile` or `keyFile` is `[]byte`, the values are treated as the certificate or key as-is.
func (s *Slim) StartTLS(address string, certFile, keyFile any) (err error) {
	s.startupMutex.Lock()
	var cert []byte
	if cert, err = filepathOrContent(certFile); err != nil {
		s.startupMutex.Unlock()
		return
	}

	var key []byte
	if key, err = filepathOrContent(keyFile); err != nil {
		s.startupMutex.Unlock()
		return
	}

	srv := s.TLSServer
	srv.TLSConfig = new(tls.Config)
	srv.TLSConfig.Certificates = make([]tls.Certificate, 1)
	if srv.TLSConfig.Certificates[0], err = tls.X509KeyPair(cert, key); err != nil {
		s.startupMutex.Unlock()
		return
	}

	s.configureTLS(address)
	if err := s.configureServer(srv); err != nil {
		s.startupMutex.Unlock()
		return err
	}
	s.startupMutex.Unlock()
	return srv.Serve(s.TLSListener)
}

func filepathOrContent(fileOrContent any) (content []byte, err error) {
	switch v := fileOrContent.(type) {
	case string:
		return os.ReadFile(v)
	case []byte:
		return v, nil
	default:
		return nil, ErrInvalidCertOrKeyType
	}
}

// StartAutoTLS starts an HTTPS server using certificates automatically installed from https://letsencrypt.org.
func (s *Slim) StartAutoTLS(address string) error {
	s.startupMutex.Lock()
	srv := s.TLSServer
	srv.TLSConfig = new(tls.Config)
	srv.TLSConfig.GetCertificate = s.AutoTLSManager.GetCertificate
	srv.TLSConfig.NextProtos = append(srv.TLSConfig.NextProtos, acme.ALPNProto)

	s.configureTLS(address)
	if err := s.configureServer(srv); err != nil {
		s.startupMutex.Unlock()
		return err
	}
	s.startupMutex.Unlock()
	return srv.Serve(s.TLSListener)
}

func (s *Slim) configureTLS(address string) {
	srv := s.TLSServer
	srv.Addr = address
	if !s.DisableHTTP2 {
		srv.TLSConfig.NextProtos = append(srv.TLSConfig.NextProtos, "h2")
	}
}

// StartServer starts a custom http server.
func (s *Slim) StartServer(srv *http.Server) (err error) {
	s.startupMutex.Lock()
	if err := s.configureServer(srv); err != nil {
		s.startupMutex.Unlock()
		return err
	}
	if srv.TLSConfig != nil {
		s.startupMutex.Unlock()
		return srv.Serve(s.TLSListener)
	}
	s.startupMutex.Unlock()
	return srv.Serve(s.Listener)
}

func (s *Slim) output() io.Writer {
	if s.StdLogger != nil {
		return s.StdLogger.Writer()
	}
	return io.Discard
}

func (s *Slim) configureServer(srv *http.Server) error {
	// Setup
	w := s.output()
	srv.ErrorLog = s.StdLogger
	srv.Handler = s

	if !s.HideBanner {
		fmt.Fprintf(w, banner, color.HiRedString("v"+Version), color.HiBlueString(website))
	}

	if srv.TLSConfig == nil {
		if s.Listener == nil {
			l, err := newListener(srv.Addr, s.ListenerNetwork)
			if err != nil {
				return err
			}
			s.Listener = l
		}
		if !s.HidePort {
			fmt.Fprintf(w, "⇨ http server started on %s\n", color.HiGreenString(s.Listener.Addr().String()))
		}
		return nil
	}
	if s.TLSListener == nil {
		l, err := newListener(srv.Addr, s.ListenerNetwork)
		if err != nil {
			return err
		}
		s.TLSListener = tls.NewListener(l, srv.TLSConfig)
	}
	if !s.HidePort {
		fmt.Fprintf(w, "⇨ http server started on %s\n", color.HiGreenString(s.TLSListener.Addr().String()))
	}
	return nil
}

// ListenerAddr returns net.Addr for Listener
func (s *Slim) ListenerAddr() net.Addr {
	s.startupMutex.RLock()
	defer s.startupMutex.RUnlock()
	if s.Listener == nil {
		return nil
	}
	return s.Listener.Addr()
}

// TLSListenerAddr returns net.Addr for TLSListener
func (s *Slim) TLSListenerAddr() net.Addr {
	s.startupMutex.RLock()
	defer s.startupMutex.RUnlock()
	if s.TLSListener == nil {
		return nil
	}
	return s.TLSListener.Addr()
}

// StartH2CServer starts a custom http/2 server with h2c (HTTP/2 Cleartext).
func (s *Slim) StartH2CServer(address string, h2s *http2.Server) error {
	s.startupMutex.Lock()
	// Setup
	w := s.output()
	srv := s.Server
	srv.Addr = address
	srv.ErrorLog = s.StdLogger
	srv.Handler = h2c.NewHandler(s, h2s)

	if !s.HideBanner {
		fmt.Fprintf(w, banner, color.HiRedString("v"+Version), color.HiBlueString(website))
	}

	if s.Listener == nil {
		l, err := newListener(srv.Addr, s.ListenerNetwork)
		if err != nil {
			s.startupMutex.Unlock()
			return err
		}
		s.Listener = l
	}
	if !s.HidePort {
		fmt.Fprintf(w, "⇨ http server started on %s\n", color.HiGreenString(s.Listener.Addr().String()))
	}
	s.startupMutex.Unlock()
	return srv.Serve(s.Listener)
}

// Close immediately stops the server.
// It internally calls `http.Server#Close()`.
func (s *Slim) Close() error {
	s.startupMutex.Lock()
	defer s.startupMutex.Unlock()
	if err := s.TLSServer.Close(); err != nil {
		return err
	}
	return s.Server.Close()
}

// Shutdown stops the server gracefully.
// It internally calls `http.Server#Shutdown()`.
func (s *Slim) Shutdown(ctx context.Context) error {
	s.startupMutex.Lock()
	defer s.startupMutex.Unlock()
	if err := s.TLSServer.Shutdown(ctx); err != nil {
		return err
	}
	return s.Server.Shutdown(ctx)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g., closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	if c, err = ln.AcceptTCP(); err != nil {
		return
	} else if err = c.(*net.TCPConn).SetKeepAlive(true); err != nil {
		return
	}
	// Ignore error from setting the KeepAlivePeriod as some systems, such as
	// OpenBSD, do not support setting TCP_USER_TIMEOUT on IPPROTO_TCP
	_ = c.(*net.TCPConn).SetKeepAlivePeriod(3 * time.Minute)
	return
}

func newListener(address, network string) (*tcpKeepAliveListener, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, ErrInvalidListenerNetwork
	}
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &tcpKeepAliveListener{l.(*net.TCPListener)}, nil
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

// DefaultErrorHandler default error handling function
func DefaultErrorHandler(c Context, err error) {
	if c.Written() {
		fmt.Fprintf(c.Slim().output(), "Error: %v\n", err)
		return
	}
	// TODO(hupeh): Return corresponding format based on Accept header
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
