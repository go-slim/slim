package middleware

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"go-slim.dev/slim"
	"go-slim.dev/slim/nego"
)

var (
	// logEntryCtxKey is the context.Context key to store the request log entry.
	logEntryCtxKey = &contextKey{"LogEntry"}

	// logPayloadCtxKey is the context.Context key to store the request log payload.
	logPayloadCtxKey = &contextKey{"LogCollector"}
)

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	TimeLayout string
	// NewEntry is called by the Logger middleware handler to log each request.
	NewEntry func(slim.Context) LogEntry
}

// DefaultLoggerConfig is the default Logger middleware config.
var DefaultLoggerConfig = LoggerConfig{
	TimeLayout: "2006/01/02 15:04:05.000",
	NewEntry: func(c slim.Context) LogEntry {
		return NewLogEntry(c.Logger().Output())
	},
}

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return. When standard output is a TTY, Logger will
// print in color, otherwise it will print in black and white. Logger prints a
// request ID if one is provided.
//
// Alternatively, look at https://github.com/goware/httplog for a more in-depth
// http logger with structured logging support.
//
// IMPORTANT NOTE: Logger should go before any other middleware that may change
// the response, such as slim.Recovery. Example:
//
//	r := slim.New()
//	r.ErrorHandler = func (c slim.Context, err error) {
//		middleware.LogEnd(c, err)       // <--<< Logger should come at error handling
//	}
//	r.Use(middleware.Logger())        // <--<< Logger should come before Recovery
//	r.Use(slim.Recovery())
//	r.Get("/", handler)
func Logger() slim.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with config.
// See: `Logger()`.
func LoggerWithConfig(config LoggerConfig) slim.MiddlewareFunc {
	if config.NewEntry == nil {
		config.NewEntry = DefaultLoggerConfig.NewEntry
	}
	return config.ToMiddleware()
}

func (l LoggerConfig) ToMiddleware() slim.MiddlewareFunc {
	if l.NewEntry == nil {
		l.NewEntry = DefaultLoggerConfig.NewEntry
	}

	return func(c slim.Context, next slim.HandlerFunc) error {
		e := l.NewEntry(c)
		e.SetTimeLayout(l.TimeLayout)
		ProvideLogEntry(c, e)

		LogBegin(c)
		if err := next(c); err != nil {
			// 需要再 ErrorHandler 里面完成
			return err
		}
		LogEnd(c, nil)

		return nil
	}
}

// LogEntry records the final log when a request completes.
// See defaultLogEntry for an example implementation.
type LogEntry interface {
	Begin(c slim.Context) map[string]any
	End(c slim.Context, err error) map[string]any
	SetTimeLayout(layout string)
	SetColorable(colorable bool)
	Print(c slim.Context, p LogPayload)
	Panic(v any, stack []byte)
}

// ProvideLogEntry sets the in-context LogEntry for a slim context.
func ProvideLogEntry(c slim.Context, entry LogEntry) {
	valueIntoContext(c, logEntryCtxKey, entry)
}

// GetLogEntry returns the in-context LogEntry for a slim context.
func GetLogEntry(c slim.Context) LogEntry {
	entry, _ := c.Value(logEntryCtxKey).(LogEntry)
	return entry
}

type LogPayload struct {
	StartTime  time.Time
	Proto      string
	RequestURI string
	RawQuery   string
	Method     string
	RemoteAddr string
	Extra      map[string]any
	Error      error
	StatusCode int
	Written    int
	Elapsed    time.Duration
}

// ProvideLogPayload sets the in-context LogPayload for a slim context.
func ProvideLogPayload(c slim.Context, p LogPayload) {
	valueIntoContext(c, logPayloadCtxKey, p)
}

// GetLogPayload returns the in-context LogPayload for a slim context.
func GetLogPayload(c slim.Context) (LogPayload, bool) {
	p, ok := c.Value(logPayloadCtxKey).(LogPayload)
	return p, ok
}

func LogBegin(c slim.Context) {
	p, ok := GetLogPayload(c)
	if !ok || p.StartTime.IsZero() {
		req := c.Request()
		p.StartTime = time.Now()
		p.Proto = req.Proto
		p.RequestURI = cmp.Or(p.RequestURI, fmt.Sprintf("%s://%s%s", c.Scheme(), req.Host, req.URL.Path))
		p.RawQuery = cmp.Or(p.RawQuery, req.URL.RawQuery)
		p.Method = cmp.Or(p.Method, req.Method)
		p.RemoteAddr = cmp.Or(p.RemoteAddr, req.RemoteAddr)
	}
	entry, ok := c.Value(logEntryCtxKey).(LogEntry)
	if ok && entry != nil {
		p.Extra = entry.Begin(c)
	}
	valueIntoContext(c, logPayloadCtxKey, p)
}

func LogEnd(c slim.Context, err error) {
	p, ok := GetLogPayload(c)
	if !ok || p.StartTime.IsZero() {
		return // no log payload, not call LogBegin()
	}

	res := c.Response()

	p.Error = err
	p.StatusCode = res.Status()
	p.Written = res.Size()
	p.Elapsed = time.Since(p.StartTime)

	if p.Extra == nil {
		p.Extra = make(map[string]any)
	}

	entry, ok := c.Value(logEntryCtxKey).(LogEntry)
	if !ok || entry == nil {
		entry = DefaultLoggerConfig.NewEntry(c)
	}
	for key, val := range entry.End(c, err) {
		p.Extra[key] = val
	}

	// prints the log payload.
	entry.Print(c, p)
}

var _ LogEntry = (*defaultLogEntry)(nil)

type defaultLogEntry struct {
	colorable  bool
	timeLayout string
	w          io.Writer
}

func NewLogEntry(w io.Writer) LogEntry {
	l := &defaultLogEntry{w: w}
	if c, ok := w.(interface{ Colorable() bool }); ok {
		l.colorable = c.Colorable()
	}
	return l
}

func (d *defaultLogEntry) Begin(c slim.Context) map[string]any {
	return nil
}

func (d *defaultLogEntry) End(c slim.Context, err error) map[string]any {
	return nil
}

func (d *defaultLogEntry) SetColorable(colorable bool) {
	d.colorable = colorable
}

func (d *defaultLogEntry) SetTimeLayout(layout string) {
	d.timeLayout = layout
}

// LogCategoryMark
// zeroWidthNonJoiner := "\u200C" // 零宽非连接符
// zeroWidthSpace := "\u200B" // 零宽空格
// zeroWidthJoiner := "\u200D" // 零宽连接符
const LogCategoryMark = "\u200C\u200B\u200D"

var UseLogCategoryMark = false

func (d *defaultLogEntry) Print(c slim.Context, p LogPayload) {
	buf := getBuffer()
	defer freeBuffer(buf)

	useColor := d.colorable

	if d.timeLayout != "" {
		cP(buf, useColor, nCyan, "%s ", p.StartTime.Format("2006/01/02 15:04:05.000"))
	}

	reqID := c.Header(nego.HeaderXRequestID)
	if reqID != "" {
		cP(buf, useColor, nYellow, "[%s] ", reqID)
	}
	cP(buf, useColor, nCyan, "\"")
	cP(buf, useColor, bMagenta, "%s ", p.Method)
	cP(buf, useColor, nCyan, "%s %s\" ", p.RequestURI, p.Proto)
	*buf = append(*buf, "from "...)
	*buf = append(*buf, p.RemoteAddr...)
	*buf = append(*buf, ' ', '-', ' ')

	switch status := p.StatusCode; {
	case status < 200:
		cP(buf, useColor, bBlue, "%03d", status)
	case status < 300:
		cP(buf, useColor, bGreen, "%03d", status)
	case status < 400:
		cP(buf, useColor, bCyan, "%03d", status)
	case status < 500:
		cP(buf, useColor, bYellow, "%03d", status)
	default:
		cP(buf, useColor, bRed, "%03d", status)
	}

	cP(buf, useColor, bBlue, " %dB", p.Written)

	*buf = append(*buf, " in "...)
	if elapsed := p.Elapsed; elapsed < 500*time.Millisecond {
		cP(buf, useColor, nGreen, "%s", elapsed)
	} else if elapsed < 5*time.Second {
		cP(buf, useColor, nYellow, "%s", elapsed)
	} else {
		cP(buf, useColor, nRed, "%s", elapsed)
	}

	if p.Error != nil {
		lines := formatError(p.Error)
		printCategory(buf, useColor, "Spot a mistake:", lines)
	}

	if extra := formatExtra(p.Extra); len(extra) > 0 {
		lines := bytes.Split(extra, []byte("\n"))
		printCategory(buf, useColor, "Additional data:", lines)
	}

	d.w.Write(*buf)
}

func (d *defaultLogEntry) Panic(v any, stack []byte) {
	PrintPrettyStack(v, stack)
}

func printCategory(buf *[]byte, useColor bool, title string, lines [][]byte) {
    if len(lines) == 0 {
        return
    }
    *buf = append(*buf, '\n', '\n')
    if UseLogCategoryMark {
        *buf = append(*buf, LogCategoryMark...)
    }
    cP(buf, useColor, dim, "%s\n", title)
    for _, line := range lines {
        *buf = fmt.Append(*buf, "\n  ")
        cP(buf, useColor, nBlue, "%s", line)
    }
}

func formatError(err error) [][]byte {
	bts := fmt.Appendf(nil, "%+v", err)
	if len(bts) == 0 {
		bts = fmt.Appendf(nil, "%s", err)
	}
	if len(bts) == 0 {
		bts = fmt.Append(nil, err.Error())
	}
	if len(bts) == 0 {
		bts = []byte("error occurred")
	}
	return bytes.Split(bts, []byte{'\n'})
}

func formatExtra(p map[string]any) []byte {
	if len(p) == 0 {
		return nil
	}
	cw := getCutoffWriter()
	defer freeCutoffWriter(cw)
	enc := json.NewEncoder(cw)
	enc.SetEscapeHTML(false)
	err := enc.Encode(p)
	if err == nil && !cw.cutoffHit {
		return cw.buf[:] // 可接受单行输出
	}
	pretty, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Appendf(nil, "%#v\n", p)
	}
	return pretty
}

var cwPool = sync.Pool{
	New: func() any {
		return &cutoffWriter{}
	},
}

func getCutoffWriter() *cutoffWriter {
	return cwPool.Get().(*cutoffWriter)
}

func freeCutoffWriter(w *cutoffWriter) {
	w.cutoffHit = false
	w.buf = w.buf[:0]
	cwPool.Put(w)
}

type cutoffWriter struct {
	buf       []byte
	cutoffHit bool
}

func (w *cutoffWriter) Write(p []byte) (int, error) {
	if len(w.buf)+len(p) > 80 {
		w.cutoffHit = true
		return 0, errors.New("output exceeds max length")
	}
	w.buf = append(w.buf, p...)
	return len(p), nil
}
