package slim

import "io"

// Logger defines the logging interface.
type Logger interface {
	Output() io.Writer
	SetOutput(w io.Writer)
	Prefix() string
	SetPrefix(p string)
	Flags() int
	SetFlags(flag int)
	Level() int
	SetLevel(v int)
	StacktraceLevel() int
	SetStacktraceLevel(v int)
	Print(i ...any)
	Printf(format string, args ...any)
	Printj(j map[string]any)
	Debug(i ...any)
	Debugf(format string, args ...any)
	Debugj(j map[string]any)
	Info(i ...any)
	Infof(format string, args ...any)
	Infoj(j map[string]any)
	Warn(i ...any)
	Warnf(format string, args ...any)
	Warnj(j map[string]any)
	Error(i ...any)
	Errorf(format string, args ...any)
	Errorj(j map[string]any)
	Panic(i ...any)
	Panicj(j map[string]any)
	Panicf(format string, args ...any)
	Fatal(i ...any)
	Fatalj(j map[string]any)
	Fatalf(format string, args ...any)
}
