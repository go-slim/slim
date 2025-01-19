package slim

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"zestack.dev/slim/nego"
)

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	// Tags to construct the logger format.
	//
	// - time_unix
	// - time_unix_milli
	// - time_unix_micro
	// - time_unix_nano
	// - time_rfc3339
	// - time_rfc3339_nano
	// - time_custom
	// - id (Request ID)
	// - remote_ip
	// - uri
	// - host
	// - method
	// - path
	// - route
	// - protocol
	// - referer
	// - user_agent
	// - status
	// - error
	// - latency (In nanoseconds)
	// - latency_human (Human readable)
	// - bytes_in (Bytes received)
	// - bytes_out (Bytes sent)
	// - header:<NAME>
	// - query:<NAME>
	// - form:<NAME>
	// - custom (see CustomTagFunc field)
	//
	// Example "${remote_ip} ${status}"
	//
	// Optional. Default value DefaultLoggerConfig.Format.
	Format string `yaml:"format"`

	// Optional. Default value DefaultLoggerConfig.CustomTimeFormat.
	CustomTimeFormat string `yaml:"custom_time_format"`

	// CustomTagFunc is function called for `${custom}` tag to output user implemented text by writing it to buf.
	// Make sure that outputted text creates valid JSON string with other logged tags.
	// Optional.
	CustomTagFunc func(c Context, buf *bytes.Buffer) (int, error)

	// Output is a writer where logs in JSON format are written.
	// Optional. Default value os.Stdout.
	Output io.Writer

	pool *sync.Pool
}

// DefaultLoggerConfig is the default Logger middleware config.
var DefaultLoggerConfig = LoggerConfig{
	Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
		`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
		`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
		`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	CustomTimeFormat: "2006-01-02 15:04:05.00000",
}

// Logging returns a middleware that logs HTTP requests.
func Logging() MiddlewareFunc {
	return LoggingWithConfig(DefaultLoggerConfig)
}

// LoggingWithConfig returns a Logger middleware with config.
// See: `Logging()`.
func LoggingWithConfig(config LoggerConfig) MiddlewareFunc {
	if config.Format == "" {
		config.Format = DefaultLoggerConfig.Format
	}
	if config.Output == nil {
		config.Output = DefaultLoggerConfig.Output
	}

	config.pool = &sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 256))
		},
	}

	return func(c Context, next HandlerFunc) (err error) {
		req := c.Request()
		res := c.Response()
		start := time.Now()
		if err = next(c); err != nil {
			c.Error(err)
		}
		stop := time.Now()
		buf := config.pool.Get().(*bytes.Buffer)
		buf.Reset()
		defer config.pool.Put(buf)

		if _, err = ExecuteFunc(config.Format, "${", "}", buf, func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "custom":
				if config.CustomTagFunc == nil {
					return 0, nil
				}
				return config.CustomTagFunc(c, buf)
			case "time_unix":
				return buf.WriteString(strconv.FormatInt(time.Now().Unix(), 10))
			case "time_unix_milli":
				return buf.WriteString(strconv.FormatInt(time.Now().UnixMilli(), 10))
			case "time_unix_micro":
				return buf.WriteString(strconv.FormatInt(time.Now().UnixMicro(), 10))
			case "time_unix_nano":
				return buf.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
			case "time_rfc3339":
				return buf.WriteString(time.Now().Format(time.RFC3339))
			case "time_rfc3339_nano":
				return buf.WriteString(time.Now().Format(time.RFC3339Nano))
			case "time_custom":
				return buf.WriteString(time.Now().Format(config.CustomTimeFormat))
			case "id":
				id := req.Header.Get(nego.HeaderXRequestID)
				if id == "" {
					id = res.Header().Get(nego.HeaderXRequestID)
				}
				return buf.WriteString(id)
			case "remote_ip":
				return buf.WriteString(c.RealIP())
			case "host":
				return buf.WriteString(req.Host)
			case "uri":
				return buf.WriteString(req.RequestURI)
			case "method":
				return buf.WriteString(req.Method)
			case "path":
				p := req.URL.Path
				if p == "" {
					p = "/"
				}
				return buf.WriteString(p)
			case "route":
				return buf.WriteString(c.RouteInfo().String())
			case "protocol":
				return buf.WriteString(req.Proto)
			case "referer":
				return buf.WriteString(req.Referer())
			case "user_agent":
				return buf.WriteString(req.UserAgent())
			case "status":
				n := res.Status()
				// s := config.colorer.Green(n)
				// switch {
				// case n >= 500:
				// 	s = config.colorer.Red(n)
				// case n >= 400:
				// 	s = config.colorer.Yellow(n)
				// case n >= 300:
				// 	s = config.colorer.Cyan(n)
				// }
				s := strconv.Itoa(n)
				return buf.WriteString(s)
			case "error":
				if err != nil {
					// Error may contain invalid JSON e.g. `"`
					b, _ := json.Marshal(err.Error())
					b = b[1 : len(b)-1]
					return buf.Write(b)
				}
			case "latency":
				l := stop.Sub(start)
				return buf.WriteString(strconv.FormatInt(int64(l), 10))
			case "latency_human":
				return buf.WriteString(stop.Sub(start).String())
			case "bytes_in":
				cl := req.Header.Get(nego.HeaderContentLength)
				if cl == "" {
					cl = "0"
				}
				return buf.WriteString(cl)
			case "bytes_out":
				return buf.WriteString(strconv.FormatInt(int64(res.Size()), 10))
			default:
				switch {
				case strings.HasPrefix(tag, "header:"):
					return buf.Write([]byte(c.Request().Header.Get(tag[7:])))
				case strings.HasPrefix(tag, "query:"):
					return buf.Write([]byte(c.QueryParam(tag[6:])))
				case strings.HasPrefix(tag, "form:"):
					return buf.Write([]byte(c.FormValue(tag[5:])))
				case strings.HasPrefix(tag, "cookie:"):
					cookie, err := c.Cookie(tag[7:])
					if err == nil {
						return buf.Write([]byte(cookie.Value))
					}
				}
			}
			return 0, nil
		}); err != nil {
			return
		}

		if config.Output == nil {
			_, err = c.Logger().Writer().Write(buf.Bytes())
			return
		}
		_, err = config.Output.Write(buf.Bytes())
		return
	}
}

func SliceByteToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StringToSliceByte(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// TagFunc can be used as a substitution value in the map passed to Execute*.
// Execute* functions pass tag (placeholder) name in 'tag' argument.
//
// TagFunc must be safe to call from concurrently running goroutines.
//
// TagFunc must write contents to w and return the number of bytes written.
type TagFunc func(w io.Writer, tag string) (int, error)

// ExecuteFunc calls f on each template tag (placeholder) occurrence.
//
// Returns the number of bytes written to w.
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteFunc for frozen templates.
func ExecuteFunc(template, startTag, endTag string, w io.Writer, f TagFunc) (int64, error) {
	s := StringToSliceByte(template)
	a := StringToSliceByte(startTag)
	b := StringToSliceByte(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		ni, err = f(w, SliceByteToString(s[:n]))
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
}
