package slim

import (
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
	output *outputVar
	level  *slog.LevelVar
}

type LoggerOptions struct {
	Output io.Writer

	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// The attribute's value has been resolved (see [Value.Resolve]).
	// If ReplaceAttr returns a zero Attr, the attribute is discarded.
	//
	// The built-in attributes with keys "time", "level", "source", and "msg"
	// are passed to this function, except that time is omitted
	// if zero, and source is omitted if AddSource is false.
	//
	// The first argument is a list of currently open groups that contain the
	// Attr. It must not be retained or modified. ReplaceAttr is never called
	// for Group attributes, only their contents. For example, the attribute
	// list
	//
	//     Int("a", 1), Group("g", Int("b", 2)), Int("c", 3)
	//
	// results in consecutive calls to ReplaceAttr with the following arguments:
	//
	//     nil, Int("a", 1)
	//     []string{"g"}, Int("b", 2)
	//     nil, Int("c", 3)
	//
	// ReplaceAttr can be used to change the default keys of the built-in
	// attributes, convert types (for example, to replace a `time.Time` with the
	// integer seconds since the Unix epoch), sanitize personal information, or
	// remove attributes from the output.
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr

	NewHandler func(w io.Writer, opts *slog.HandlerOptions) slog.Handler
}

type outputVar struct {
	io.Writer
}

func (o *outputVar) Write(p []byte) (int, error) {
	return o.Writer.Write(p)
}

func NewLogger(opts *LoggerOptions) *Logger {
	if opts.Output == nil {
		opts.Output = os.Stderr
	}
	if opts.Level == nil {
		opts.Level = slog.LevelError
	}
	if opts.NewHandler == nil {
		opts.NewHandler = func(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
			return slog.NewTextHandler(w, opts)
		}
	}
	level := &slog.LevelVar{}
	output := &outputVar{opts.Output}
	level.Set(opts.Level.Level())
	return &Logger{
		Logger: slog.New(opts.NewHandler(output, &slog.HandlerOptions{
			AddSource:   opts.AddSource,
			Level:       level,
			ReplaceAttr: opts.ReplaceAttr,
		})),
		output: output,
		level:  level,
	}
}

func (l *Logger) Output() io.Writer {
	return l.output.Writer
}

func (l *Logger) SetLevel(level slog.Level) (oldLevel slog.Level) {
	oldLevel = l.level.Level()
	l.level.Set(level)
	return
}

func (l *Logger) Level() slog.Leveler {
	return l.level
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		output: l.output,
		level:  l.level,
	}
}

func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(name),
		output: l.output,
		level:  l.level,
	}
}
