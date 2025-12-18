package slogcute

import (
	"context"
	"encoding/json"
	"io"
	stdLog "log"
	"log/slog"

	"github.com/fatih/color"
)

type CuteHandlerOptions struct {
	SlogOptions *slog.HandlerOptions
}

type CuteHandler struct {
	logger *stdLog.Logger
	attrs  []slog.Attr
}

// NewCuteHandler creates a new CuteHandler with the given options.
func (opts CuteHandlerOptions) NewCuteHandler(out io.Writer) *CuteHandler {
	handler := &CuteHandler{
		logger: stdLog.New(out, "", 0),
	}
	return handler
}

// Enabled always returns true, indicating that all log levels are enabled.
func (handler *CuteHandler) Enabled(_ context.Context, _ slog.Level) bool {
	// Always enabled for all log levels
	return true
}

// Handle formats and outputs the log record in a cute way.
func (handler *CuteHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	fields := make(map[string]interface{}, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()

		return true
	})

	for _, a := range handler.attrs {
		fields[a.Key] = a.Value.Any()
	}

	var b []byte
	var err error

	if len(fields) > 0 {
		b, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	}

	timeStr := r.Time.Format("[15:05:05.000]")
	msg := color.CyanString(r.Message)

	handler.logger.Println(
		timeStr,
		level,
		msg,
		color.WhiteString(string(b)),
	)

	return nil
}

// WithAttrs returns a new CuteHandler with the given attributes added.
func (handler *CuteHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CuteHandler{
		logger: handler.logger,
		attrs:  append(handler.attrs, attrs...),
	}
}

// WithGroup returns a new CuteHandler with the given group added.
func (handler *CuteHandler) WithGroup(_ string) slog.Handler {

	return &CuteHandler{
		logger: handler.logger,
		attrs:  handler.attrs,
	}
}
