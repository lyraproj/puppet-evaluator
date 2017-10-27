package evaluator

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/puppetlabs/go-parser/issue"
)

type (
	LogLevel string

	Logger interface {
		Log(level LogLevel, args ...PValue)

		Logf(level LogLevel, format string, args ...interface{})

		LogIssue(issue *issue.ReportedIssue)
	}

	stdlog struct {
		out io.Writer
		err io.Writer
	}

	LogEntry struct {
		level   LogLevel
		message string
	}

	ArrayLogger struct {
		entries []*LogEntry
	}
)

const (
	ALERT   = LogLevel(`alert`)
	CRIT    = LogLevel(`crit`)
	DEBUG   = LogLevel(`debug`)
	EMERG   = LogLevel(`emerg`)
	ERR     = LogLevel(`err`)
	INFO    = LogLevel(`info`)
	NOTICE  = LogLevel(`notice`)
	WARNING = LogLevel(`warning`)
)

var LOG_LEVELS = []LogLevel{ALERT, CRIT, DEBUG, EMERG, ERR, INFO, NOTICE, WARNING}

func Alert(logger Logger, format string, args ...interface{}) {
	logger.Logf(ALERT, format, args)
}

func Crit(logger Logger, format string, args ...interface{}) {
	logger.Logf(CRIT, format, args)
}

func Debug(logger Logger, format string, args ...interface{}) {
	logger.Logf(DEBUG, format, args)
}

func Emerg(logger Logger, format string, args ...interface{}) {
	logger.Logf(EMERG, format, args)
}

func Err(logger Logger, format string, args ...interface{}) {
	logger.Logf(ERR, format, args)
}

func Info(logger Logger, format string, args ...interface{}) {
	logger.Logf(INFO, format, args)
}

func Notice(logger Logger, format string, args ...interface{}) {
	logger.Logf(NOTICE, format, args)
}

func Warning(logger Logger, format string, args ...interface{}) {
	logger.Logf(WARNING, format, args)
}

func NewStdLogger() Logger {
	return &stdlog{os.Stdout, os.Stderr}
}

func (l *stdlog) Log(level LogLevel, args ...PValue) {
	w := l.writerFor(level)
	fmt.Fprintf(w, `%s: `, level)
	for _, arg := range args {
		ToString3(arg, w)
	}
}

func (l *stdlog) Logf(level LogLevel, format string, args ...interface{}) {
	w := l.writerFor(level)
	fmt.Fprintf(w, `%s: `, level)
	fmt.Fprintf(w, format, args...)
}

func (l *stdlog) writerFor(level LogLevel) io.Writer {
	switch level {
	case DEBUG, INFO, NOTICE:
		return l.out
	default:
		return l.err
	}
}

func (l *stdlog) LogIssue(issue *issue.ReportedIssue) {
	fmt.Fprintln(l.err, issue.String())
}

func NewArrayLogger() *ArrayLogger {
	return &ArrayLogger{make([]*LogEntry, 0, 16)}
}

func (l *ArrayLogger) Entries(level LogLevel) (result []string) {
	result = make([]string, 0, 8)
	for _, entry := range l.entries {
		if entry.level == level {
			result = append(result, entry.message)
		}
	}
	return
}

func (l *ArrayLogger) Log(level LogLevel, args ...PValue) {
	w := bytes.NewBufferString(``)
	for _, arg := range args {
		ToString3(arg, w)
	}
	l.entries = append(l.entries, &LogEntry{level, w.String()})
}

func (l *ArrayLogger) Logf(level LogLevel, format string, args ...interface{}) {
	l.entries = append(l.entries, &LogEntry{level, fmt.Sprintf(format, args...)})
}

func (l *ArrayLogger) LogIssue(i *issue.ReportedIssue) {
	var level LogLevel
	switch i.Severity() {
	case issue.SEVERITY_ERROR:
		level = ERR
	case issue.SEVERITY_WARNING, issue.SEVERITY_DEPRECATION:
		level = WARNING
	default:
		return
	}
	l.entries = append(l.entries, &LogEntry{level, i.String()})
}
