package middleware

// The original work was derived from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"go-slim.dev/slim"
)

// RecoveryConfig defines the config for Recovery middleware.
type RecoveryConfig struct {
	// Size of the stack to be printed.
	// Optional. Default value 4KB.
	StackSize int
	// DisableStackAll disables formatting stack traces of all other goroutines
	// into buffer after the trace for the current goroutine.
	// Optional. Default value is false.
	DisableStackAll bool
	// DisablePrintStack disables printing stack trace.
	// Optional. Default value as false.
	DisablePrintStack bool
}

// DefaultRecoveryConfig is the default Recovery middleware config.
var DefaultRecoveryConfig = RecoveryConfig{
	StackSize:         4 << 10, // 4 KB
	DisableStackAll:   false,
	DisablePrintStack: false,
}

// Recovery returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized ErrorHandler.
func Recovery() slim.MiddlewareFunc {
	return RecoveryWithConfig(DefaultRecoveryConfig)
}

// RecoveryWithConfig returns Recovery middleware with config or panics on invalid configuration.
func RecoveryWithConfig(config RecoveryConfig) slim.MiddlewareFunc {
	return config.ToMiddleware()
}

// ToMiddleware converts RecoveryConfig to middleware or returns an error for invalid configuration
func (config RecoveryConfig) ToMiddleware() slim.MiddlewareFunc {
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoveryConfig.StackSize
	}

	// the middleware that recovers from panics, logs the panic (and a
	// backtrace), and returns an HTTP 500 (Internal Server Error) status if
	// possible. Recovery prints a request ID if one is provided.
	//
	// Alternatively, look at https://github.com/go-chi/httplog middleware pkgs.
	return func(c slim.Context, next slim.HandlerFunc) (err error) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}

				if !config.DisablePrintStack {
					stack := make([]byte, config.StackSize)
					for {
						n := runtime.Stack(stack, !config.DisableStackAll)
						if n < len(stack) {
							stack = stack[:n]
							break
						}
						stack = make([]byte, 2*len(stack))
					}

					logEntry := GetLogEntry(c)
					if logEntry != nil {
						logEntry.Panic(rvr, stack)
					} else {
						PrintPrettyStack(rvr, stack)
					}
				}

				if c.Header("Connection") != "Upgrade" && !c.Response().Written() {
					c.Response().WriteHeader(http.StatusInternalServerError)
				}
			}
		}()
		err = next(c)
		return
	}
}

// for ability to test the PrintPrettyStack function
var recovererErrorWriter io.Writer = os.Stderr

func PrintPrettyStack(rvr any, debugStack []byte) {
	debugStack = debug.Stack()
	s := prettyStack{}
	out, err := s.parse(debugStack, rvr)
	if err == nil {
		recovererErrorWriter.Write(out)
	} else {
		// print stdlib output as a fallback
		os.Stderr.Write(debugStack)
	}
}

type prettyStack struct{}

func (s prettyStack) parse(debugStack []byte, rvr any) ([]byte, error) {
	var err error
	useColor := true
	buf := &bytes.Buffer{}

	cW(buf, false, bRed, "\n")
	cW(buf, useColor, bCyan, " panic: ")
	cW(buf, useColor, bBlue, "%v", rvr)
	cW(buf, false, bWhite, "\n \n")

	// process debug stack info
	stack := strings.Split(string(debugStack), "\n")
	lines := []string{}

	// locate panic line, as we may have nested panics
	for i := len(stack) - 1; i > 0; i-- {
		lines = append(lines, stack[i])
		if strings.HasPrefix(stack[i], "panic(") {
			lines = lines[0 : len(lines)-2] // remove boilerplate
			break
		}
	}

	// reverse
	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}

	// decorate
	for i, line := range lines {
		lines[i], err = s.decorateLine(line, useColor, i)
		if err != nil {
			return nil, err
		}
	}

	for _, l := range lines {
		fmt.Fprintf(buf, "%s", l)
	}
	return buf.Bytes(), nil
}

func (s prettyStack) decorateLine(line string, useColor bool, num int) (string, error) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "\t") || strings.Contains(line, ".go:") {
		return s.decorateSourceLine(line, useColor, num)
	}
	if strings.HasSuffix(line, ")") {
		return s.decorateFuncCallLine(line, useColor, num)
	}
	if strings.HasPrefix(line, "\t") {
		return strings.Replace(line, "\t", "      ", 1), nil
	}
	return fmt.Sprintf("    %s\n", line), nil
}

func (s prettyStack) decorateFuncCallLine(line string, useColor bool, num int) (string, error) {
	idx := strings.LastIndex(line, "(")
	if idx < 0 {
		return "", errors.New("not a func call line")
	}

	buf := &bytes.Buffer{}
	pkg := line[0:idx]
	// addr := line[idx:]
	method := ""

	if idx := strings.LastIndex(pkg, string(os.PathSeparator)); idx < 0 {
		if idx := strings.Index(pkg, "."); idx > 0 {
			method = pkg[idx:]
			pkg = pkg[0:idx]
		}
	} else {
		method = pkg[idx+1:]
		pkg = pkg[0 : idx+1]
		if idx := strings.Index(method, "."); idx > 0 {
			pkg += method[0:idx]
			method = method[idx:]
		}
	}
	pkgColor := nYellow
	methodColor := bGreen

	if num == 0 {
		cW(buf, useColor, bRed, " -> ")
		pkgColor = bMagenta
		methodColor = bRed
	} else {
		cW(buf, useColor, bWhite, "    ")
	}
	cW(buf, useColor, pkgColor, "%s", pkg)
	cW(buf, useColor, methodColor, "%s\n", method)
	// cW(buf, useColor, nBlack, "%s", addr)
	return buf.String(), nil
}

func (s prettyStack) decorateSourceLine(line string, useColor bool, num int) (string, error) {
	idx := strings.LastIndex(line, ".go:")
	if idx < 0 {
		return "", errors.New("not a source line")
	}

	buf := &bytes.Buffer{}
	path := line[0 : idx+3]
	lineno := line[idx+3:]

	idx = strings.LastIndex(path, string(os.PathSeparator))
	dir := path[0 : idx+1]
	file := path[idx+1:]

	idx = strings.Index(lineno, " ")
	if idx > 0 {
		lineno = lineno[0:idx]
	}
	fileColor := bCyan
	lineColor := bGreen

	if num == 1 {
		cW(buf, useColor, bRed, " ->   ")
		fileColor = bRed
		lineColor = bMagenta
	} else {
		cW(buf, false, bWhite, "      ")
	}
	cW(buf, useColor, bWhite, "%s", dir)
	cW(buf, useColor, fileColor, "%s", file)
	cW(buf, useColor, lineColor, "%s", lineno)
	if num == 1 {
		cW(buf, false, bWhite, "\n")
	}
	cW(buf, false, bWhite, "\n")

	return buf.String(), nil
}
