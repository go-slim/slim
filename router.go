package slim

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
)

// Router is interface for routing requests to registered routes.
type Router interface {
	// MiddlewareRegistrar implements middleware registration interface
	MiddlewareRegistrar
	// MiddlewareComposer implements middleware composer interface
	MiddlewareComposer
	// ErrorHandlerRegistrar implements error handler registration interface
	ErrorHandlerRegistrar
	// RouteRegistrar implements route registration interface
	RouteRegistrar
	// RouteMatcher implements route matcher interface
	RouteMatcher
	// Add registers a new route for method and path with matching handler.
	Add([]string, string, HandlerFunc) (Route, error)
	// Remove removes route
	Remove(methods []string, path string) error
	// Routes returns registered routes
	Routes() []Route
	// URI generates a URI from handler.
	URI(h HandlerFunc, params ...any) string
	// Reverse generates a URL from route name and provided parameters.
	Reverse(name string, params ...any) string
}

// RouteMatchType describes possible states that request could be in perspective of routing
type RouteMatchType uint8

const (
	// RouteMatchUnknown is state before routing is done. Default state for fresh context.
	RouteMatchUnknown RouteMatchType = iota
	// RouteMatchNotFound is state when router did not find matching route for current request
	RouteMatchNotFound
	// RouteMatchMethodNotAllowed is state when router did not find route with matching path + method for current request.
	// Although router had a matching route with that path but different method.
	RouteMatchMethodNotAllowed
	// RouteMatchFound is state when router found exact match for path + method combination
	RouteMatchFound
)

// RouteMatch is result object for Router.Match. Its main purpose is to avoid allocating memory for PathParams inside router.
type RouteMatch struct {
	// Type contains a result as enumeration of Router.Match and helps to understand did Router actually matched Route or
	// what kind of error case (404/405) we have at the end of the handler chain.
	Type RouteMatchType
	// AllowMethods list of acceptable request methods, mainly
	// used when Type value is RouteMatchMethodNotAllowed.
	AllowMethods []string
	// Handler is function(chain) that was matched by router. In case of no match could result to ErrNotFound or ErrMethodNotAllowed.
	Handler HandlerFunc
	// RouteInfo is information about the route we just matched
	RouteInfo RouteInfo
}

// RouteMatcher route matcher interface
type RouteMatcher interface {
	// Match matches route
	Match(r *http.Request, p *PathParams) RouteMatch
}

// RouteCollector route collector interface
type RouteCollector interface {
	// MiddlewareRegistrar 实现中间件注册接口
	MiddlewareRegistrar
	// MiddlewareComposer 实现中间件合成器接口
	MiddlewareComposer
	// ErrorHandlerRegistrar 实现错误处理器注册接口
	ErrorHandlerRegistrar
	// RouteRegistrar 实现路由注册器接口
	RouteRegistrar
	// Prefix returns the common prefix of routes
	Prefix() string
	// Parent returns the parent route collector
	Parent() RouteCollector
	// Router returns the belonging router
	Router() Router
}

// RouteRegistrar route registration interface
//
// RouteRegistrar.Some, RouteRegistrar.Any, and
// RouteRegistrar.Handle provide extension capabilities
// for custom non-standard HTTP request methods.
type RouteRegistrar interface {
	// Group groups routes to conveniently apply one or more middleware
	// to routes in the same group, and make them use the same handling
	// for errors and panics
	Group(fn func(sub RouteCollector))
	// Route specifies the same prefix for routes in the same group,
	// with usage and internal logic consistent with the Group method
	Route(prefix string, fn func(sub RouteCollector))
	// Some registers a new route for multiple HTTP methods and path with matching
	// handler in the router. Panics on error.
	Some(methods []string, pattern string, h HandlerFunc) Route
	// Any registers a new route for all supported HTTP methods and path with matching
	// handler in the router. Panics on error.
	Any(pattern string, h HandlerFunc) Route
	// CONNECT registers a new CONNECT route for a path with matching handler in
	// the router. Panics on error.
	CONNECT(pattern string, h HandlerFunc) Route
	// DELETE registers a new DELETE route for a path with matching handler in
	// the router. Panics on error.
	DELETE(pattern string, h HandlerFunc) Route
	// GET registers a new GET route for a path with matching handler in
	// the router. Panics on error.
	GET(pattern string, h HandlerFunc) Route
	// HEAD registers a new HEAD route for a path with matching handler in
	// the router. Panics on error.
	HEAD(pattern string, h HandlerFunc) Route
	// OPTIONS registers a new OPTIONS route for a path with matching handler
	// in the router. Panics on error.
	OPTIONS(pattern string, h HandlerFunc) Route
	// PATCH registers a new PATCH route for a path with matching handler in
	// the router. Panics on error.
	PATCH(pattern string, h HandlerFunc) Route
	// POST registers a new POST route for a path with matching handler in
	// the router. Panics on error.
	POST(pattern string, h HandlerFunc) Route
	// PUT registers a new PUT route for a path with matching handler in
	// the router. Panics on error.
	PUT(pattern string, h HandlerFunc) Route
	// TRACE registers a new TRACE route for a path with matching handler in
	// the router. Panics on error.
	TRACE(pattern string, h HandlerFunc) Route
	// Handle registers a route that supports specified request methods
	Handle(method, pattern string, h HandlerFunc) Route
	// Static registers a new route with path prefix to serve static files
	// from the provided root directory. Panics on error.
	Static(prefix, root string) Route
	// File registers a new route with a path to serve a static file.
	// Panics on error.
	File(pattern, file string) Route
}

// Route route interface
type Route interface {
	// MiddlewareRegistrar implements middleware registration interface
	MiddlewareRegistrar
	// MiddlewareComposer implements middleware composer interface
	MiddlewareComposer
	// Router returns the belonging router
	Router() Router
	// Collector returns the belonging collector
	Collector() RouteCollector
	// Name returns the route name
	Name() string
	// SetName sets the route name
	SetName(name string) Route
	// Title returns the route title
	Title() string
	// SetTitle sets the route title
	SetTitle(name string) Route
	// Pattern route path expression
	Pattern() string
	// Methods returns the supported HTTP request methods
	Methods() []string
	// Handler returns the registered request handler function
	Handler() HandlerFunc
	// Params returns the supported route parameter list
	Params() []string
	// RouteInfo returns the route description interface implementation
	RouteInfo() RouteInfo
}

// RouteInfo route description interface
type RouteInfo interface {
	// Router returns the belonging router
	Router() Router
	// Collector returns the belonging collector
	Collector() RouteCollector
	// Name returns the route name
	Name() string
	// Title returns the route title
	Title() string
	// Methods returns the supported request method list
	Methods() []string
	// Pattern route path expression
	Pattern() string
	// Params returns the supported route parameter list
	Params() []string
	// Reverse reverses the route expression through provided parameters, returning a real request path.
	// If the parameters are empty or nil, it tries to use default values, and if the parameters
	// cannot be resolved, it will panic with an error
	Reverse(params ...any) string
	// String returns string form
	String() string
}

type RouterConfig struct {
	AllowOverwritingRoute    bool
	UnescapePathParamValues  bool
	UseEscapedPathForRouting bool
	RoutingTrailingSlash     bool
	RouteCollector           RouteCollector
}

func NewRouter(config RouterConfig) Router {
	r := &routerImpl{
		collector:                config.RouteCollector,
		tree:                     &node{},
		routes:                   make([]Route, 0),
		middleware:               make([]MiddlewareFunc, 0),
		allowOverwritingRoute:    config.AllowOverwritingRoute,
		unescapePathParamValues:  config.UnescapePathParamValues,
		useEscapedPathForRouting: config.UseEscapedPathForRouting,
		routingTrailingSlash:     config.RoutingTrailingSlash,
	}
	if r.collector == nil {
		r.collector = NewRouteCollector("", nil, r)
	}
	return r
}

func NewRouteCollector(prefix string, parent RouteCollector, router Router) RouteCollector {
	if router == nil {
		if parent != nil {
			router = parent.Router()
		}
	} else if parent != nil {
		if parent.Router() != router {
			panic("slim: invalid router for the given parent")
		}
	}
	if router == nil {
		panic("slim: no router found")
	}
	return &routeCollectorImpl{
		prefix: prefix,
		parent: parent,
		router: router,
	}
}

var nextRouteId uint32

var _ Router = (*routerImpl)(nil)

type routerImpl struct {
	collector    RouteCollector   // route collector
	tree         *node            // route node tree, same as root node tree
	routes       []Route          // actual type is `[]*routeImpl`
	middleware   []MiddlewareFunc // middleware list
	errorHandler ErrorHandler     // route-level error handler
	slim         *Slim

	allowOverwritingRoute    bool
	unescapePathParamValues  bool
	useEscapedPathForRouting bool
	routingTrailingSlash     bool
}

func (r *routerImpl) UseErrorHandler(h ErrorHandler) {
	r.errorHandler = h
}

func (r *routerImpl) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *routerImpl) Middleware() []MiddlewareFunc {
	return r.middleware
}

func (r *routerImpl) Compose() MiddlewareFunc {
	return Compose(r.middleware...)
}

func (r *routerImpl) Add(methods []string, pattern string, h HandlerFunc) (Route, error) {
	segments, trailingSlash := split(pattern)
	params := make([]string, 0)
	tail, _ := r.tree.insert(segments, &params, 0)
	route := &routeImpl{
		id:      atomic.AddUint32(&nextRouteId, 1),
		name:    handlerName(h),
		pattern: strings.Join(segments, ""),
		methods: methods,
		params:  params,
		handler: h,
	}
	for _, method := range methods {
		if e := tail.leaf.endpoint(method); e != nil {
			if !r.allowOverwritingRoute {
				panic(errors.New("slim: adding duplicate route (same method+path) is not allowed"))
			}
			r.routes = slices.DeleteFunc(r.routes, func(route Route) bool {
				return route.(*routeImpl).id == e.routeId
			})
			e.trailingSlash = trailingSlash
			e.routeId = route.id
		} else {
			tail.leaf.endpoints = append(tail.leaf.endpoints, &endpoint{
				method:        method,
				pattern:       route.pattern,
				trailingSlash: trailingSlash,
				routeId:       route.id,
			})
		}
	}
	sort.Sort(tail.leaf.endpoints) // Sort endpoints
	r.routes = append(r.routes, route)
	// TODO(hupeh): How to handle remove
	if r.slim.contextPathParamAllocSize < tail.leaf.paramsCount {
		r.slim.contextPathParamAllocSize = tail.leaf.paramsCount
	}
	return route, nil
}

func (r *routerImpl) Group(fn func(sub RouteCollector)) {
	r.collector.Group(fn)
}

func (r *routerImpl) Route(prefix string, fn func(sub RouteCollector)) {
	r.collector.Route(prefix, fn)
}

func (r *routerImpl) Some(methods []string, pattern string, h HandlerFunc) Route {
	return r.collector.Some(methods, pattern, h)
}

func (r *routerImpl) Any(pattern string, h HandlerFunc) Route {
	return r.collector.Any(pattern, h)
}

func (r *routerImpl) CONNECT(pattern string, h HandlerFunc) Route {
	return r.collector.CONNECT(pattern, h)
}

func (r *routerImpl) DELETE(pattern string, h HandlerFunc) Route {
	return r.collector.DELETE(pattern, h)
}

func (r *routerImpl) GET(pattern string, h HandlerFunc) Route {
	return r.collector.GET(pattern, h)
}

func (r *routerImpl) HEAD(pattern string, h HandlerFunc) Route {
	return r.collector.HEAD(pattern, h)
}

func (r *routerImpl) OPTIONS(pattern string, h HandlerFunc) Route {
	return r.collector.OPTIONS(pattern, h)
}

func (r *routerImpl) PATCH(pattern string, h HandlerFunc) Route {
	return r.collector.PATCH(pattern, h)
}

func (r *routerImpl) POST(pattern string, h HandlerFunc) Route {
	return r.collector.POST(pattern, h)
}

func (r *routerImpl) PUT(pattern string, h HandlerFunc) Route {
	return r.collector.PUT(pattern, h)
}

func (r *routerImpl) TRACE(pattern string, h HandlerFunc) Route {
	return r.collector.TRACE(pattern, h)
}

func (r *routerImpl) Handle(method, pattern string, h HandlerFunc) Route {
	return r.collector.Handle(method, pattern, h)
}

func (r *routerImpl) Static(prefix, root string) Route {
	return r.collector.Static(prefix, root)
}

func (r *routerImpl) File(pattern, file string) Route {
	return r.collector.File(pattern, file)
}

// Remove service endpoint through combination of `method+pattern`
func (r *routerImpl) Remove(methods []string, path string) error {
	segments, trailingSlash := split(path)
	routes, ok := r.tree.remove(methods, trailingSlash, r.routingTrailingSlash, segments, 0)
	if !ok {
		return nil
	}
	for _, rid := range routes {
		i := slices.IndexFunc(r.routes, func(rt Route) bool {
			return rt.(*routeImpl).id == rid
		})
		if i == -1 {
			return errors.New("route not found")
		}
		r.routes[i].(*routeImpl).Remove()
	}
	return nil
}

func (r *routerImpl) Routes() []Route {
	return r.routes
}

func (r *routerImpl) Match(req *http.Request, pathParams *PathParams) RouteMatch {
	*pathParams = (*pathParams)[0:cap(*pathParams)]
	path := req.URL.Path
	if r.useEscapedPathForRouting && req.URL.RawPath != "" {
		path = req.URL.RawPath
	}
	segments, tailingSlash := split(path)
	tail := r.tree.match(segments, 0)
	result := RouteMatch{Type: RouteMatchNotFound, Handler: NotFoundHandler}
	if tail == nil {
		*pathParams = (*pathParams)[0:0]
		return result
	}
	// Reallocate length based on leaf parameter count
	*pathParams = (*pathParams)[0:tail.leaf.paramsCount]
	var ep *endpoint
	result.AllowMethods, ep = tail.leaf.match(req.Method)
	if ep == nil || (ep.trailingSlash != tailingSlash && !r.routingTrailingSlash) {
		// TODO(hupeh): In case of OPTIONS method, can respond using the method list owned by this node.
		// FIXME: See https://httpwg.org/specs/rfc7231.html#OPTIONS
		result.Type = RouteMatchMethodNotAllowed
		result.Handler = MethodNotAllowedHandler
		return result
	}
	// Find route
	var route *routeImpl
	for _, rr := range r.routes {
		dr := rr.(*routeImpl)
		if dr.id == ep.routeId {
			route = dr
			break
		}
	}
	// Internal error if not found
	if route == nil {
		panic(fmt.Errorf(
			"slim: route %s@%s#%d not found",
			req.Method, ep.pattern, ep.routeId,
		))
	}
	pattern := route.Pattern()
	var valueIndex int
	var paramIndex int
	for i, l := 0, len(pattern); i < l; i++ {
		if pattern[i] == paramLabel || pattern[i] == anyLabel {
			j := i
			for ; j < l && pattern[j] != pathSeparator; j++ {
			}
			if paramIndex >= tail.leaf.paramsCount {
				panic(fmt.Errorf(
					"slim: invalid param count for routing  %s@%s#%d",
					req.Method, ep.pattern, ep.routeId,
				))
			}
			key := pattern[i+1 : j]
			value := segments[valueIndex][1:]
			if pattern[i] == anyLabel {
				if key == "" {
					key = string(anyLabel)
				}
				value = strings.Join(segments[valueIndex:], "")[1:]
				if tailingSlash {
					value += "/"
				}
				i = l
			} else {
				i = j
			}
			//  there are cases when path parameter needs to be unescaped
			tmpVal, err := url.PathUnescape(value)
			if err == nil { // handle problems by ignoring them.
				value = tmpVal
			}
			(*pathParams)[paramIndex].Name = key
			(*pathParams)[paramIndex].Value = value
			paramIndex++
			valueIndex++
		} else if pattern[i] == pathSeparator && i > 0 {
			valueIndex++
		}
	}
	result.Type = RouteMatchFound
	result.Handler = ComposeChainHandler(route)
	result.RouteInfo = route.RouteInfo()
	return result
}

func (r *routerImpl) HandleError(c Context, err error) {
	if r.errorHandler != nil {
		r.errorHandler.HandleError(c, err)
	} else if eh := r.slim.ErrorHandler; eh != nil {
		// Do not use c.Error(err) to handle errors here, otherwise
		// it will cause an infinite loop, because the c.Error method
		// recursively searches for error handlers from the route up
		// to handle errors.
		eh.HandleError(c, err)
	}
}

func (r *routerImpl) URI(h HandlerFunc, params ...any) string {
	for _, route := range r.routes {
		if handlerName(route.Handler()) == handlerName(h) {
			return route.RouteInfo().Reverse(params...)
		}
	}
	return ""
}

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

func (r *routerImpl) Reverse(name string, params ...any) string {
	for _, route := range r.routes {
		if route.Name() == name {
			return route.RouteInfo().Reverse(params...)
		}
	}
	return ""
}

// ComposeChainHandler composes middleware from route collector and route middleware
func ComposeChainHandler(route Route) HandlerFunc {
	return func(c Context) error {
		stack := make([]MiddlewareFunc, 0)
		collector := route.Collector()
		for collector != nil {
			if mw := collector.Compose(); mw != nil {
				stack = append(stack, mw)
			}
			collector = collector.Parent()
		}
		// The above is reverse, so need to reverse here
		slices.Reverse(stack)
		mw := Compose(stack...)
		h := HandlerFunc(func(c Context) error {
			h2 := route.Handler()
			mw2 := Compose(route.Middleware()...)
			if mw2 != nil {
				return mw2(c, h2)
			}
			return h2(c)
		})
		if mw == nil {
			return h(c)
		}
		return mw(c, h)
	}
}

var _ RouteCollector = (*routeCollectorImpl)(nil)

type routeCollectorImpl struct {
	prefix       string           // route prefix
	parent       RouteCollector   // parent route collector
	router       Router           // parent router
	middleware   []MiddlewareFunc // middleware list
	errorHandler ErrorHandler
}

func (rc *routeCollectorImpl) UseErrorHandler(h ErrorHandler) {
	rc.errorHandler = h
}

func (rc *routeCollectorImpl) HandleError(c Context, err error) {
	if rc.errorHandler != nil {
		rc.errorHandler.HandleError(c, err)
	} else if eh, ok := geteh(c, rc.parent, rc.router); ok {
		eh.HandleError(c, err)
	}
}

func geteh(c Context, vs ...any) (ErrorHandler, bool) {
	for _, v := range vs {
		if v != nil {
			if eh, ok := v.(ErrorHandler); ok {
				return eh, true
			}
		}
	}
	if eh := c.Slim().ErrorHandler; eh != nil {
		return eh, true
	}
	return nil, false
}

func (rc *routeCollectorImpl) Prefix() string {
	return rc.prefix
}

func (rc *routeCollectorImpl) Parent() RouteCollector {
	return rc.parent
}

func (rc *routeCollectorImpl) Router() Router {
	return rc.router
}

func (rc *routeCollectorImpl) Use(middleware ...MiddlewareFunc) {
	rc.middleware = append(rc.middleware, middleware...)
}

func (rc *routeCollectorImpl) Middleware() []MiddlewareFunc {
	return rc.middleware
}

func (rc *routeCollectorImpl) Compose() MiddlewareFunc {
	return Compose(rc.middleware...)
}

func (rc *routeCollectorImpl) Group(fn func(sub RouteCollector)) {
	if fn != nil {
		fn(NewRouteCollector("", rc, nil))
	}
}

func (rc *routeCollectorImpl) Route(prefix string, fn func(sub RouteCollector)) {
	if fn != nil {
		fn(NewRouteCollector(prefix, rc, nil))
	}
}

func (rc *routeCollectorImpl) Some(methods []string, pattern string, h HandlerFunc) Route {
	var collector RouteCollector
	collector = rc
	for collector != nil {
		pattern = collector.Prefix() + pattern
		collector = collector.Parent()
	}
	route, err := rc.Router().Add(methods, pattern, h)
	if err != nil {
		panic(err)
	}
	// TODO(hupeh): Need a more reasonable way to set route collector
	if dr, ok := route.(*routeImpl); ok {
		dr.collector = rc
	}
	return route
}

func (rc *routeCollectorImpl) Any(pattern string, h HandlerFunc) Route {
	return rc.Some([]string{"*"}, pattern, h)
}

func (rc *routeCollectorImpl) CONNECT(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodConnect, pattern, h)
}

func (rc *routeCollectorImpl) DELETE(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodDelete, pattern, h)
}

func (rc *routeCollectorImpl) GET(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodGet, pattern, h)
}

func (rc *routeCollectorImpl) HEAD(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodHead, pattern, h)
}

func (rc *routeCollectorImpl) OPTIONS(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodOptions, pattern, h)
}

func (rc *routeCollectorImpl) PATCH(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodPatch, pattern, h)
}

func (rc *routeCollectorImpl) POST(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodPost, pattern, h)
}

func (rc *routeCollectorImpl) PUT(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodPut, pattern, h)
}

func (rc *routeCollectorImpl) TRACE(pattern string, h HandlerFunc) Route {
	return rc.Handle(http.MethodTrace, pattern, h)
}

func (rc *routeCollectorImpl) Handle(method string, pattern string, h HandlerFunc) Route {
	return rc.Some([]string{method}, pattern, h)
}

func (rc *routeCollectorImpl) Static(prefix, root string) Route {
	return rc.GET(prefix+"*", StaticDirectoryHandler(root, false))
}

func (rc *routeCollectorImpl) File(pattern, file string) Route {
	if filepath.IsAbs(file) {
		rel := strings.TrimPrefix(filepath.Clean(file), string(filepath.Separator))
		return rc.GET(pattern, func(c Context) error { return c.File(rel, os.DirFS(string(filepath.Separator))) })
	}
	return rc.GET(pattern, func(c Context) error { return c.File(file) })
}

// StaticDirectoryHandler creates handler function to serve files from given a root path
func StaticDirectoryHandler(root string, disablePathUnescaping bool) HandlerFunc {
	if root == "" {
		root = "." // For security, we want to restrict to CWD.
	}
	return func(c Context) error {
		p := c.PathParam("*")
		if !disablePathUnescaping { // when router is already unescaping, we do not want to do is twice
			tmpPath, err := url.PathUnescape(p)
			if err != nil {
				return fmt.Errorf("failed to unescape path variable: %w", err)
			}
			p = tmpPath
		}

		name := filepath.Join(root, filepath.Clean("/"+p)) // "/"+ for security
		fi, err := fs.Stat(c.Filesystem(), name)
		if err != nil {
			// The access path does not exist
			return ErrNotFound
		}

		// If the request is for a directory and does not end with "/"
		p = c.Request().URL.Path // the path must not be empty.
		if fi.IsDir() && p[len(p)-1] != '/' {
			// Redirect to end with "/"
			return c.Redirect(http.StatusMovedPermanently, p+"/")
		}
		return c.File(name)
	}
}

var _ Route = (*routeImpl)(nil)
var _ RouteInfo = (*routeImpl)(nil)

type routeImpl struct {
	id         uint32
	name       string
	title      string
	collector  RouteCollector
	pattern    string
	methods    []string
	params     []string
	handler    HandlerFunc
	middleware []MiddlewareFunc
}

func (r *routeImpl) SetName(name string) Route   { r.name = name; return r }
func (r *routeImpl) SetTitle(title string) Route { r.title = title; return r }
func (r *routeImpl) Use(middleware ...MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware...)
}
func (r *routeImpl) ID() uint32                   { return r.id }
func (r *routeImpl) Router() Router               { return r.collector.Router() }
func (r *routeImpl) Collector() RouteCollector    { return r.collector }
func (r *routeImpl) Name() string                 { return r.name }
func (r *routeImpl) Title() string                { return r.title }
func (r *routeImpl) Pattern() string              { return r.pattern }
func (r *routeImpl) Methods() []string            { return r.methods[:] }
func (r *routeImpl) Handler() HandlerFunc         { return r.handler }
func (r *routeImpl) Params() []string             { return r.params[:] }
func (r *routeImpl) Middleware() []MiddlewareFunc { return r.middleware[:] }
func (r *routeImpl) Compose() MiddlewareFunc      { return Compose(r.middleware...) }
func (r *routeImpl) RouteInfo() RouteInfo         { return r }
func (r *routeImpl) Remove() {
	router := r.Router()
	if x, ok := router.(*routerImpl); ok {
		x.routes = slices.DeleteFunc(x.routes, func(route Route) bool {
			return route.(*routeImpl).id == r.id
		})
	} else {
		// Reserved interface for custom routers to remove sub-routes
		if i, yes := router.(interface{ RemoveRoute(Route) }); yes {
			i.RemoveRoute(r)
		}
	}
}
func (r *routeImpl) Reverse(params ...any) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	for i, l := 0, len(r.pattern); i < l; i++ {
		if (r.pattern[i] == paramLabel || r.pattern[i] == anyLabel) && n < ln {
			// in case of `*` wildcard or `:` (unescaped colon) param we replace everything till next slash or end of path
			for ; i < l && r.pattern[i] != pathSeparator; i++ {
			}
			if n < ln {
				fmt.Fprintf(uri, "%v", params[n])
			}
			n++
		}
		if i < l {
			uri.WriteByte(r.pattern[i])
		}
	}
	return uri.String()
}
func (r *routeImpl) String() string {
	if r.name != "" {
		return fmt.Sprintf("%s (%s)", r.name, r.pattern)
	}
	return r.pattern
}
