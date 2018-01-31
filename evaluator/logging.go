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

	LogEntry interface {
		Level() LogLevel
		Message() string
	}

	ArrayLogger struct {
		entries []LogEntry
	}

	ReportedEntry struct {
		issue *issue.ReportedIssue
	}

	TextEntry struct {
		level   LogLevel
		message string
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
	IGNORE  = LogLevel(``)
)

var LOG_LEVELS = []LogLevel{ALERT, CRIT, DEBUG, EMERG, ERR, INFO, NOTICE, WARNING}

func (l LogLevel) Severity() issue.Severity {
	switch l {
	case CRIT, EMERG, ERR:
		return issue.SEVERITY_ERROR
	case ALERT, WARNING:
		return issue.SEVERITY_WARNING
	default:
		return issue.SEVERITY_IGNORE
	}
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
	return &ArrayLogger{make([]LogEntry, 0, 16)}
}

func (l *ArrayLogger) Entries(level LogLevel) []LogEntry {
	result := make([]LogEntry, 0, 8)
	for _, entry := range l.entries {
		if entry.Level() == level {
			result = append(result, entry)
		}
	}
	return result
}

func (l *ArrayLogger) Log(level LogLevel, args ...PValue) {
	w := bytes.NewBufferString(``)
	for _, arg := range args {
		ToString3(arg, w)
	}
	l.entries = append(l.entries, &TextEntry{level, w.String()})
}

func (l *ArrayLogger) Logf(level LogLevel, format string, args ...interface{}) {
	l.entries = append(l.entries, &TextEntry{level, fmt.Sprintf(format, args...)})
}

func (l *ArrayLogger) LogIssue(i *issue.ReportedIssue) {
	if i.Severity() != issue.SEVERITY_IGNORE {
		l.entries = append(l.entries, &ReportedEntry{i})
	}
}

func (te *TextEntry) Level() LogLevel {
	return te.level
}

func (te *TextEntry) Message() string {
	return te.message
}

func (re *ReportedEntry) Level() LogLevel {
	return LogLevelFromSeverity(re.issue.Severity())
}

func (re *ReportedEntry) Message() string {
	return re.issue.String()
}

func (re *ReportedEntry) Issue() *issue.ReportedIssue {
	return re.issue
}

func LogLevelFromSeverity(severity issue.Severity) LogLevel {
	switch severity {
	case issue.SEVERITY_ERROR:
		return ERR
	case issue.SEVERITY_WARNING, issue.SEVERITY_DEPRECATION:
		return WARNING
	default:
		return IGNORE
	}
}
