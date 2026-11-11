package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// prettyTimeLayout is the local time shown in the console format.
const prettyTimeLayout = "15:04:05.000"

// ANSI sequences used when the output is a terminal.
const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiBold   = "\x1b[1m"
	ansiGray   = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

// prettyHandler writes one human-readable line per record:
//
//	15:04:05.000 INFO  message key=value group.key=value
//
// It is intended for local development. Values that contain spaces, quotes or
// control characters are quoted and escaped, so a record can never be split
// into several lines (log injection, ADR-003).
type prettyHandler struct {
	writer io.Writer
	mu     *sync.Mutex
	level  slog.Leveler
	color  bool
	groups string
	attrs  string
}

// newPrettyHandler builds the console handler. Colors are enabled only when
// the writer is a terminal and NO_COLOR is not set.
func newPrettyHandler(w io.Writer, level slog.Leveler) *prettyHandler {
	return &prettyHandler{
		writer: w,
		mu:     &sync.Mutex{},
		level:  level,
		color:  isTerminal(w) && os.Getenv("NO_COLOR") == "",
	}
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

// Enabled implements slog.Handler.
func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements slog.Handler.
func (h *prettyHandler) Handle(_ context.Context, record slog.Record) error {
	var line strings.Builder

	h.paint(&line, ansiDim, record.Time.Format(prettyTimeLayout))
	line.WriteByte(' ')
	h.writeLevel(&line, record.Level)
	line.WriteByte(' ')
	h.paint(&line, ansiBold, escapeText(record.Message))
	line.WriteString(h.attrs)

	record.Attrs(func(attr slog.Attr) bool {
		writeAttr(&line, h.groups, attr)

		return true
	})

	line.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := io.WriteString(h.writer, line.String())

	return err
}

// WithAttrs implements slog.Handler.
func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h

	var preformatted strings.Builder

	preformatted.WriteString(h.attrs)

	for _, attr := range attrs {
		writeAttr(&preformatted, h.groups, attr)
	}

	clone.attrs = preformatted.String()

	return &clone
}

// WithGroup implements slog.Handler.
func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := *h
	clone.groups = h.groups + name + "."

	return &clone
}

func (h *prettyHandler) writeLevel(line *strings.Builder, level slog.Level) {
	switch {
	case level >= slog.LevelError:
		h.paint(line, ansiRed+ansiBold, "ERROR")
	case level >= slog.LevelWarn:
		h.paint(line, ansiYellow+ansiBold, "WARN ")
	case level >= slog.LevelInfo:
		h.paint(line, ansiGreen, "INFO ")
	default:
		h.paint(line, ansiGray, "DEBUG")
	}
}

func (h *prettyHandler) paint(line *strings.Builder, sequence, text string) {
	if !h.color {
		line.WriteString(text)

		return
	}

	line.WriteString(sequence)
	line.WriteString(text)
	line.WriteString(ansiReset)
}

func writeAttr(line *strings.Builder, groups string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		prefix := groups
		if attr.Key != "" {
			prefix += attr.Key + "."
		}

		for _, nested := range attr.Value.Group() {
			writeAttr(line, prefix, nested)
		}

		return
	}

	line.WriteByte(' ')
	line.WriteString(groups)
	line.WriteString(attr.Key)
	line.WriteByte('=')
	line.WriteString(quoteIfNeeded(attr.Value.String()))
}

// escapeText keeps human text readable: spaces and quotes are allowed, but any
// control character forces strconv.Quote so the record stays on one line.
func escapeText(value string) string {
	for _, r := range value {
		if !unicode.IsPrint(r) && r != ' ' {
			return strconv.Quote(value)
		}
	}

	return value
}

// quoteIfNeeded returns value unchanged when it is a plain token and a quoted,
// escaped form otherwise. strconv.Quote escapes newlines and control bytes.
func quoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}

	for _, r := range value {
		if r == ' ' || r == '"' || r == '=' || r == '\\' || !unicode.IsPrint(r) {
			return strconv.Quote(value)
		}
	}

	return value
}
