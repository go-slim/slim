package slim

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zestack.dev/slim/nego"
)

// Context 网络请求上下文，包含了请求数据（路径、路径参数、载荷）
// 和响应对象以及已注册的处理程序等。
type Context interface {
	// Context 实现 context.Context 接口
	context.Context
	// Request 返回当前请求的 `*http.Request` 结构体实例
	Request() *http.Request
	// SetRequest 为上下文设置新的 `*http.Request` 结构体实例。
	SetRequest(r *http.Request)
	// Response 返回当前请求的 `http.ResponseWriter` 接口实现
	Response() ResponseWriter
	// SetResponse 为上下文设置新的 `http.ResponseWriter` 实现
	SetResponse(r ResponseWriter)
	// Logger returns the `Logger` instance.
	Logger() *log.Logger
	// SetLogger Set the logger
	SetLogger(logger *log.Logger)
	// Filesystem returns `fs.FS`.
	Filesystem() fs.FS
	// SetFilesystem sets `fs.FS`
	SetFilesystem(fs.FS)
	// IsTLS 判断 HTTP 连接是否采用了 Transport Layer Security (TLS) 协议，
	// 如果是返回 true，否则返回 false。
	IsTLS() bool
	// IsWebSocket 判断 HTTP 连接是否为 WebSocket 协议，如果
	// 是就返回 true，否则返回 false。
	IsWebSocket() bool
	// Scheme 获取 HTTP 请求的协议方案，返回值为 `http` 或者 `https`
	Scheme() string
	// RealIP returns the client's network address based on `X-Forwarded-For`
	// or `X-Real-IP` request header.
	// The behavior can be configured using `Slim#IPExtractor`.
	RealIP() string
	RequestURI() string
	Is(types ...string) string
	// Accepts 返回支持的权重最高的媒体类型，若匹配失败则会返回空字符串。
	// 给出的值可以是标准的媒体类型（如 application/json），也可以是扩展名（如 json、xml 等）。
	Accepts(expect ...string) string
	// AcceptsEncodings 返回支持的权重最高的编码方式，若匹配失败则会返回空字符串。
	AcceptsEncodings(encodings ...string) string
	// AcceptsCharsets 返回支持的权重最高的字符集，若匹配失败则会返回空字符串。
	AcceptsCharsets(charsets ...string) string
	// AcceptsLanguages 返回支持的权重最高的语言，若匹配失败则会返回空字符串。
	AcceptsLanguages(languages ...string) string
	// AllowsMethods 返回允许的请求方法
	AllowsMethods() []string
	// RouteMatchType returns router match type for current context. This helps middlewares to distinguish which type
	// of match router found and how this request context handler chain could end:
	// * route match - this path + method had matching route.
	// * not found - this path did not match any routes enough to be considered match
	// * method not allowed - path had routes registered but for other method types then current request is
	// * unknown - initial state for fresh context before router tries to do routing
	//
	// Note: for slim-middleware (Slim.Use) this method result is always RouteMatchUnknown as at point router has not tried
	// to match request to route.
	RouteMatchType() RouteMatchType
	// RouteInfo returns current request route information. Method, Path, Name and params if they exist for matched route.
	// In the case of 404 (route not found) and 405 (method not allowed) RouteInfo returns generic struct for these cases.
	RouteInfo() RouteInfo
	// PathParam returns path parameter by name.
	PathParam(name string) string
	// PathParams returns path parameter values.
	PathParams() PathParams
	// SetPathParams set path parameter for during current request lifecycle.
	SetPathParams(params PathParams)
	// QueryParam returns the query param for the provided name.
	QueryParam(name string) string
	// QueryParams returns the query parameters as `url.Values`.
	QueryParams() url.Values
	// QueryString returns the URL query string.
	QueryString() string
	// FormValue returns the form field value for the provided name.
	FormValue(name string) string
	// FormParams returns the form parameters as `url.Values`.
	FormParams() (url.Values, error)
	// FormFile returns the multipart form file for the provided name.
	FormFile(name string) (*multipart.FileHeader, error)
	Header(key string) string
	SetHeader(key string, values ...string)
	// MultipartForm returns the multipart form.
	MultipartForm() (*multipart.Form, error)
	// Cookie returns the named cookie provided in the request.
	Cookie(name string) (*http.Cookie, error)
	// SetCookie adds a `Set-Cookie` header in HTTP response.
	SetCookie(cookie *http.Cookie)
	// Cookies return the HTTP cookies sent with the request.
	Cookies() []*http.Cookie
	// Get retrieves data from the context.
	Get(key string) any
	// Set saves data in the context.
	Set(key string, val any)
	// Bind binds the request body into a provided type `i`. The default binder
	// does it based on Content-Type header.
	Bind(i any) error
	// Validate validates provided `i`. It is usually called after `Context#Bind()`.
	// Validator must be registered using `Slim#Validator`.
	Validate(i any) error
	// Written returns whether the context response has been written to
	Written() bool
	// Render renders a template with data and sends a text/html response with status
	// code. Renderer must be registered using `Slim.Renderer`.
	Render(code int, name string, data any) error
	// HTML sends an HTTP response with status code.
	HTML(code int, html string) error
	// HTMLBlob sends an HTTP blob response with status code.
	HTMLBlob(code int, b []byte) error
	// String sends a string response with status code.
	String(code int, s string) error
	// JSON sends a JSON response with status code.
	JSON(code int, i any) error
	// JSONPretty sends a pretty-print JSON with status code.
	JSONPretty(code int, i any, indent string) error
	// JSONBlob sends a JSON blob response with status code.
	JSONBlob(code int, b []byte) error
	// JSONP sends a JSONP response with status code. It uses `callback` to construct
	// the JSONP payload.
	JSONP(code int, callback string, i any) error
	// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
	// to construct the JSONP payload.
	JSONPBlob(code int, callback string, b []byte) error
	// XML sends an XML response with status code.
	XML(code int, i any) error
	// XMLPretty sends a pretty-print XML with status code.
	XMLPretty(code int, i any, indent string) error
	// XMLBlob sends an XML blob response with status code.
	XMLBlob(code int, b []byte) error
	// Blob sends a blob response with a status code and content type.
	Blob(code int, contentType string, b []byte) error
	// Stream sends a streaming response with status code and content type.
	Stream(code int, contentType string, r io.Reader) error
	// File sends a response with the content of the file.
	File(file string, filesystem ...fs.FS) error
	// Attachment sends a response as attachment, prompting client to save the
	// file.
	Attachment(file string, name string) error
	// Inline sends a response as inline, opening the file in the browser.
	Inline(file string, name string) error
	// NoContent sends a response with nobody and a status code.
	NoContent(code ...int) error
	// Redirect redirects the request to a provided URL with status code.
	Redirect(code int, url string) error
	// Error invokes the registered HTTP error handler.
	// NB: Avoid using this method. It is better to return errors, so middlewares up in a chain could act on returned error.
	Error(err error)
	// Slim 返回 Slim 实例
	Slim() *Slim
}

type EditableContext interface {
	Context
	// RawPathParams returns raw path pathParams value.
	RawPathParams() *PathParams
	// SetRawPathParams replaces any existing param values with new values for this context lifetime (request).
	SetRawPathParams(params *PathParams)
	// SetRouteMatchType sets the RouteMatchType of router match for this request.
	SetRouteMatchType(t RouteMatchType)
	SetAllowsMethods(methods []string)
	// SetRouteInfo sets the route info of this request to the context.
	SetRouteInfo(ri RouteInfo)
	// Reset resets the context after request completes. It must be called along
	// with `Slim#AcquireContext()` and `Slim#ReleaseContext()`.
	// See `Slim#ServeHTTP()`
	Reset(w http.ResponseWriter, r *http.Request)
}

var _ EditableContext = &contextImpl{}

type BytesGetter interface {
	Bytes() []byte
}

type contextImpl struct {
	request       *http.Request
	response      ResponseWriter
	matchType     RouteMatchType
	allowsMethods []string
	route         RouteInfo
	filesystem    fs.FS
	// pathParams holds path/uri parameters determined by Router.
	// The Lifecycle is handled by Slim to reduce allocations.
	pathParams *PathParams
	// currentParams hold path parameters set by non-Slim implementation (custom middlewares, handlers) during the lifetime of Request.
	// Lifecycle is not handle by Slim and could have excess allocations per served Request
	currentParams PathParams
	negotiator    *nego.Negotiator
	logger        *log.Logger
	query         url.Values
	store         map[string]any
	slim          *Slim
	mu            sync.RWMutex
}

func (x *contextImpl) Deadline() (deadline time.Time, ok bool) {
	return x.request.Context().Deadline()
}

func (x *contextImpl) Done() <-chan struct{} {
	return x.request.Context().Done()
}

func (x *contextImpl) Err() error {
	return x.request.Context().Err()
}

func (x *contextImpl) Value(key any) any {
	x.mu.RLock()
	defer x.mu.RUnlock()

	if k, ok := key.(*contextKey); ok {
		switch k.name {
		case SlimContextKey.name:
			return x.slim
		case RequestContextKey.name:
			return x.request
		case ResponseContextKey.name:
			return x.response
		case ContextKey.name:
			return x
		}
	}

	if ks, ok := key.(string); ok {
		value, has := x.store[ks]
		if has {
			return value
		}
	}

	return x.request.Context().Value(key)
}

// Reset resets the context after request completes. It must be called along
// with `Slim.AcquireContext()` and `Slim.ReleaseContext()`.
// See `Slim.ServeHTTP()`
func (x *contextImpl) Reset(w http.ResponseWriter, r *http.Request) {
	x.request = x.wrap(r)
	x.response = NewResponseWriter(r.Method, w) // todo x.response.reset
	x.matchType = RouteMatchUnknown
	x.allowsMethods = x.allowsMethods[:0]
	x.route = nil
	x.filesystem = nil
	// NOTE: Don't reset because it has to have length c.slim.contextPathParamAllocSize at all times
	*x.pathParams = (*x.pathParams)[:0]
	x.currentParams = nil
	x.logger = nil
	x.query = nil
	x.store = nil
}

// Request returns `*http.Request`.
func (x *contextImpl) Request() *http.Request {
	return x.request
}

// SetRequest sets `*http.Request`.
func (x *contextImpl) SetRequest(r *http.Request) {
	x.request = x.wrap(r)
}

func (x *contextImpl) wrap(r *http.Request) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, SlimContextKey, x.slim)
	ctx = context.WithValue(ctx, ContextKey, x)
	return r.WithContext(ctx)
}

// Response returns `slim.ResponseWriter`.
func (x *contextImpl) Response() ResponseWriter {
	return x.response
}

// SetResponse sets `slim.ResponseWriter`.
func (x *contextImpl) SetResponse(w ResponseWriter) {
	x.response = w
}

func (x *contextImpl) Logger() *log.Logger {
	if x.logger != nil {
		return x.logger
	}
	if x.slim.Logger == nil {
		panic(errors.New("logger not registered"))
	}
	return x.slim.Logger
}

func (x *contextImpl) SetLogger(l *log.Logger) {
	x.logger = l
}

// Filesystem returns `fs.FS`.
func (x *contextImpl) Filesystem() fs.FS {
	if x.filesystem != nil {
		return x.filesystem
	}
	return x.slim.Filesystem
}

// SetFilesystem sets `fs.FS`
func (x *contextImpl) SetFilesystem(filesystem fs.FS) {
	x.filesystem = filesystem
}

// IsTLS returns true if HTTP connection is TLS otherwise false.
func (x *contextImpl) IsTLS() bool {
	return x.request.TLS != nil
}

// IsWebSocket returns true if HTTP connection is WebSocket otherwise false.
func (x *contextImpl) IsWebSocket() bool {
	upgrade := x.request.Header.Get(nego.HeaderUpgrade)
	return strings.EqualFold(upgrade, "websocket")
}

func (x *contextImpl) Scheme() string {
	// Can't use `r.Request.URL.Scheme`
	// See: https://groups.google.com/forum/#!topic/golang-nuts/pMUkBlQBDF0
	if x.IsTLS() {
		return "https"
	}
	if scheme := x.request.Header.Get(nego.HeaderXForwardedProto); scheme != "" {
		return scheme
	}
	if scheme := x.request.Header.Get(nego.HeaderXForwardedProtocol); scheme != "" {
		return scheme
	}
	if ssl := x.request.Header.Get(nego.HeaderXForwardedSsl); ssl == "on" {
		return "https"
	}
	if scheme := x.request.Header.Get(nego.HeaderXUrlScheme); scheme != "" {
		return scheme
	}
	return "http"
}

func (x *contextImpl) RealIP() string {
	if x.slim.IPExtractor != nil {
		return x.slim.IPExtractor(x.request)
	}
	// Fall back to legacy behavior
	if ip := x.request.Header.Get(nego.HeaderXForwardedFor); ip != "" {
		i := strings.IndexAny(ip, ",")
		if i > 0 {
			xffip := strings.TrimSpace(ip[:i])
			xffip = strings.TrimPrefix(xffip, "[")
			xffip = strings.TrimSuffix(xffip, "]")
			return xffip
		}
		return ip
	}
	if ip := x.request.Header.Get(nego.HeaderXRealIP); ip != "" {
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")
		return ip
	}
	ra, _, _ := net.SplitHostPort(x.request.RemoteAddr)
	return ra
}

func (x *contextImpl) RequestURI() string {
	return x.request.RequestURI
}

func (x *contextImpl) Is(types ...string) string {
	typ, _ := nego.TypeIs(x.Header("Content-Type"), types...)
	return typ
}

func (x *contextImpl) Accepts(expect ...string) string {
	return x.slim.negotiator.Type(x.request, expect...)
}

func (x *contextImpl) AcceptsEncodings(encodings ...string) string {
	return x.slim.negotiator.Encoding(x.request, encodings...)
}

func (x *contextImpl) AcceptsCharsets(charsets ...string) string {
	return x.slim.negotiator.Charset(x.request, charsets...)
}

func (x *contextImpl) AcceptsLanguages(languages ...string) string {
	return x.slim.negotiator.Language(x.request, languages...)
}

func (x *contextImpl) AllowsMethods() []string {
	return x.allowsMethods[:]
}

func (x *contextImpl) SetAllowsMethods(methods []string) {
	x.allowsMethods = methods
}

// RouteMatchType returns router match type for current context. This helps middlewares to distinguish which type
// of match router found and how this request context handler chain could end:
// * route match - this path + method had matching route.
// * not found - this path did not match any routes enough to be considered match
// * method not allowed - path had routes registered but for other method types then current request is
// * unknown - initial state for fresh context before router tries to do routing
func (x *contextImpl) RouteMatchType() RouteMatchType {
	return x.matchType
}

// SetRouteMatchType sets the RouteMatchType of router match for this request.
func (x *contextImpl) SetRouteMatchType(t RouteMatchType) {
	x.matchType = t
}

// RouteInfo returns current request route information. Method, Path, Name and params if they exist for matched route.
// In the case of 404 (route not found) and 405 (method not allowed) RouteInfo returns generic struct for these cases.
func (x *contextImpl) RouteInfo() RouteInfo {
	return x.route
}

// SetRouteInfo sets the route info of this request to the context.
func (x *contextImpl) SetRouteInfo(ri RouteInfo) {
	x.route = ri
}

// RawPathParams returns raw path pathParams value.
func (x *contextImpl) RawPathParams() *PathParams {
	return x.pathParams
}

// SetRawPathParams replaces any existing param values with new values for this context lifetime (request).
func (x *contextImpl) SetRawPathParams(params *PathParams) {
	x.pathParams = params
}

// PathParam returns the corresponding path parameter value from the request
// routing context.
func (x *contextImpl) PathParam(name string) string {
	if x.currentParams != nil {
		return x.currentParams.Get(name, "")
	}
	return x.pathParams.Get(name, "")
}

// PathParams returns path parameter values.
func (x *contextImpl) PathParams() PathParams {
	if x.currentParams != nil {
		return x.currentParams
	}
	result := make(PathParams, len(*x.pathParams))
	copy(result, *x.pathParams)
	return result
}

// SetPathParams set path parameter for during current request lifecycle.
func (x *contextImpl) SetPathParams(params PathParams) {
	x.currentParams = params
}

func (x *contextImpl) QueryParam(name string) string {
	return x.QueryParams().Get(name)
}

func (x *contextImpl) QueryParams() url.Values {
	if x.query == nil {
		x.query = x.request.URL.Query()
	}
	return x.query
}

func (x *contextImpl) QueryString() string {
	return x.request.URL.RawQuery
}

func (x *contextImpl) FormValue(name string) string {
	return x.request.FormValue(name)
}

func (x *contextImpl) FormParams() (url.Values, error) {
	if strings.HasPrefix(x.request.Header.Get(nego.HeaderContentType), nego.MIMEMultipartForm) {
		err := x.request.ParseMultipartForm(x.slim.MultipartMemoryLimit)
		if err != nil {
			return nil, err
		}
	} else {
		err := x.request.ParseForm()
		if err != nil {
			return nil, err
		}
	}
	return x.request.Form, nil
}

func (x *contextImpl) FormFile(name string) (*multipart.FileHeader, error) {
	f, fh, err := x.request.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, nil
}

func (x *contextImpl) MultipartForm() (*multipart.Form, error) {
	err := x.request.ParseMultipartForm(x.slim.MultipartMemoryLimit)
	return x.request.MultipartForm, err
}

func (x *contextImpl) Header(key string) string {
	return x.request.Header.Get(key)
}

func (x *contextImpl) SetHeader(key string, values ...string) {
	header := x.response.Header()
	for i, value := range values {
		if i == 0 {
			header.Set(key, value)
		} else {
			header.Add(key, value)
		}
	}
}

func (x *contextImpl) Cookie(name string) (*http.Cookie, error) {
	return x.request.Cookie(name)
}

func (x *contextImpl) SetCookie(cookie *http.Cookie) {
	http.SetCookie(x.response, cookie)
}

func (x *contextImpl) Cookies() []*http.Cookie {
	return x.request.Cookies()
}

func (x *contextImpl) Get(key string) any {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.store[key]
}

func (x *contextImpl) Set(key string, val any) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.store == nil {
		x.store = make(map[string]any)
	}
	x.store[key] = val
}

// Bind binds the request body into a provided type `i`. The default binder
// does it based on Content-Type header.
func (x *contextImpl) Bind(i any) error {
	return x.slim.Binder.Bind(x, i)
}

// Validate validates provided `i`. It is usually called after `Context#Bind()`.
// Validator must be registered using `Slim#Validator`.
func (x *contextImpl) Validate(i any) error {
	if x.slim.Validator == nil {
		return ErrValidatorNotRegistered
	}
	return x.slim.Validator.Validate(i)
}

// Written returns whether the context response has been written to
func (x *contextImpl) Written() bool {
	return x.response.Written()
}

func (x *contextImpl) writeContentType(value string) {
	if value != "" {
		header := x.response.Header()
		if header.Get(nego.HeaderContentType) == "" {
			header.Set(nego.HeaderContentType, value)
		}
	}
}

func (x *contextImpl) prettyIndent() string {
	_, pretty := x.QueryParams()["pretty"]
	if x.slim.Debug || pretty {
		return x.slim.PrettyIndent
	}
	return ""
}

// Render renders a template with data and sends a text/html response with status
// code. Renderer must be registered using `Slim.Renderer`.
func (x *contextImpl) Render(code int, name string, data any) error {
	if x.slim.Renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	err := x.slim.Renderer.Render(x, buf, name, data)
	if err != nil {
		return err
	}
	return x.HTMLBlob(code, buf.Bytes())
}

// HTML sends an HTTP response with status code.
func (x *contextImpl) HTML(code int, html string) error {
	return x.HTMLBlob(code, []byte(html))
}

// HTMLBlob sends an HTTP blob response with status code.
func (x *contextImpl) HTMLBlob(code int, b []byte) error {
	return x.Blob(code, nego.MIMETextHTMLCharsetUTF8, b)
}

// String sends a string response with status code.
func (x *contextImpl) String(code int, s string) error {
	return x.Blob(code, nego.MIMETextPlainCharsetUTF8, []byte(s))
}

// JSON sends a JSON response with status code.
func (x *contextImpl) JSON(code int, i any) error {
	return x.JSONPretty(code, i, x.prettyIndent())
}

// JSONPretty sends a pretty-print JSON with status code.
func (x *contextImpl) JSONPretty(code int, i any, indent string) error {
	x.writeContentType(nego.MIMEApplicationJSONCharsetUTF8)
	x.response.WriteHeader(code)
	return x.slim.JSONSerializer.Serialize(x.response, i, indent)
}

// JSONBlob sends a JSON blob response with status code.
func (x *contextImpl) JSONBlob(code int, b []byte) error {
	return x.Blob(code, nego.MIMEApplicationJSONCharsetUTF8, b)
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct
// the JSONP payload.
func (x *contextImpl) JSONP(code int, callback string, i any) error {
	x.writeContentType(nego.MIMEApplicationJavaScriptCharsetUTF8)
	x.response.WriteHeader(code)
	if _, err := x.response.Write([]byte(callback + "(")); err != nil {
		return err
	}
	if err := x.slim.JSONSerializer.Serialize(x.response, i, x.prettyIndent()); err != nil {
		return err
	}
	if _, err := x.response.Write([]byte(");")); err != nil {
		return err
	}
	return nil
}

// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
// to construct the JSONP payload.
func (x *contextImpl) JSONPBlob(code int, callback string, b []byte) error {
	x.writeContentType(nego.MIMEApplicationJavaScriptCharsetUTF8)
	x.response.WriteHeader(code)
	if _, err := x.response.Write([]byte(callback + "(")); err != nil {
		return err
	}
	if _, err := x.response.Write(b); err != nil {
		return err
	}
	_, err := x.response.Write([]byte(");"))
	return err
}

// XML sends an XML response with status code.
func (x *contextImpl) XML(code int, i any) error {
	return x.XMLPretty(code, i, x.prettyIndent())
}

// XMLPretty sends a pretty-print XML with status code.
func (x *contextImpl) XMLPretty(code int, i any, indent string) error {
	x.writeContentType(nego.MIMEApplicationXMLCharsetUTF8)
	x.response.WriteHeader(code)
	if _, err := x.response.Write([]byte(xml.Header)); err != nil {
		return err
	}
	return x.slim.XMLSerializer.Serialize(x.response, i, indent)
}

// XMLBlob sends an XML blob response with status code.
func (x *contextImpl) XMLBlob(code int, b []byte) error {
	x.writeContentType(nego.MIMEApplicationXMLCharsetUTF8)
	x.response.WriteHeader(code)
	_, err := x.response.Write([]byte(xml.Header))
	if err == nil {
		_, err = x.response.Write(b)
	}
	return err
}

// Blob sends a blob response with a status code and content type.
func (x *contextImpl) Blob(code int, contentType string, b []byte) error {
	x.writeContentType(contentType)
	x.response.WriteHeader(code)
	_, err := x.response.Write(b)
	return err
}

// Stream sends a streaming response with status code and content type.
func (x *contextImpl) Stream(code int, contentType string, r io.Reader) error {
	x.writeContentType(contentType)
	x.response.WriteHeader(code)
	_, err := io.Copy(x.response, r)
	return err
}

// The File sends a response with the content of the file.
func (x *contextImpl) File(file string, filesystem ...fs.FS) error {
	var lfs fs.FS
	for _, i := range filesystem {
		if i != nil {
			lfs = i
			break
		}
	}
	if lfs == nil {
		lfs = x.Filesystem()
		if lfs == nil {
			return ErrFilesystemNotRegistered
		}
	}
	f, err := lfs.Open(file)
	if err != nil {
		if !x.slim.Debug || os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	defer f.Close()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, "index.html")
		f, err = lfs.Open(file)
		if err != nil {
			if !x.slim.Debug || os.IsNotExist(err) {
				return ErrNotFound
			}
			return err
		}
		defer f.Close()
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}
	ff, ok := f.(io.ReadSeeker)
	if !ok {
		return errors.New("slim: file does not implement io.ReadSeeker")
	}
	http.ServeContent(x.response, x.request, fi.Name(), fi.ModTime(), ff)
	return nil
}

// Attachment sends a response as attachment, prompting a client to save the file.
func (x *contextImpl) Attachment(file string, name string) error {
	return x.contentDisposition(file, name, "attachment")
}

// Inline sends a response as inline, opening the file in the browser.
func (x *contextImpl) Inline(file string, name string) error {
	return x.contentDisposition(file, name, "inline")
}

func (x *contextImpl) contentDisposition(file, name, dispositionType string) error {
	x.SetHeader(nego.HeaderContentDisposition, fmt.Sprintf("%s; filename=%q", dispositionType, name))
	return x.File(file)
}

// NoContent sends a response with nobody and a status code.
func (x *contextImpl) NoContent(code ...int) error {
	for _, status := range code {
		x.response.WriteHeader(status)
		return nil
	}
	x.response.WriteHeader(http.StatusNoContent)
	return nil
}

// Redirect redirects the request to a provided URL with status code.
func (x *contextImpl) Redirect(code int, location string) error {
	if code < 300 || code > 308 {
		return ErrInvalidRedirectCode
	}
	http.Redirect(x.response, x.request, location, code)
	return nil
}

// Error invokes the registered HTTP error handler.
// NB: Avoid using this method. It is better to return errors, so middlewares up in chain could act on returned error.
func (x *contextImpl) Error(err error) {
	x.slim.ErrorHandler(x, err)
}

func (x *contextImpl) Slim() *Slim {
	return x.slim
}

// PathParams 路由参数列表
type PathParams []PathParam

// PathParam 路径参数
type PathParam struct {
	Name  string // 参数名
	Value string // 参数值
}

// Get 获取与 name 对应的值，若不存在，则返回提供的默认值
func (p PathParams) Get(name string, defaultValue ...string) string {
	if value, ok := p.Lookup(name); ok && value != "" {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Lookup 检查并返回参数值，
// 第一个返回值是与 name 对应的值，第二个返回值表示是否存在。
func (p PathParams) Lookup(name string) (string, bool) {
	for _, param := range p {
		if param.Name == name {
			return param.Value, true
		}
	}
	return "", false
}
