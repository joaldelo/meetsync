package logs

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents the severity level of a log message
type Level int

const (
	// DEBUG level for detailed troubleshooting information
	DEBUG Level = iota
	// INFO level for general operational information
	INFO
	// WARN level for non-critical issues
	WARN
	// ERROR level for errors that should be investigated
	ERROR
	// FATAL level for critical errors that require the process to exit
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var levelValues = map[string]Level{
	"DEBUG": DEBUG,
	"INFO":  INFO,
	"WARN":  WARN,
	"ERROR": ERROR,
	"FATAL": FATAL,
}

// Logger is a simple wrapper around Go's log package
type Logger struct {
	level  Level
	logger *log.Logger
}

// New creates a new Logger with the specified minimum level and output
func New(level string, output io.Writer) *Logger {
	l, exists := levelValues[strings.ToUpper(level)]
	if !exists {
		l = INFO
	}

	if output == nil {
		output = os.Stdout
	}

	return &Logger{
		level:  l,
		logger: log.New(output, "", 0),
	}
}

// log logs a message at the specified level if it's greater than or equal to the logger's level
func (l *Logger) log(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.logger.Printf("[%s] %s: %s", timestamp, levelNames[level], msg)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a message at DEBUG level
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Info logs a message at INFO level
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Warn logs a message at WARN level
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

// Error logs a message at ERROR level
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Fatal logs a message at FATAL level and exits the program
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
}

// Default logger instance
var defaultLogger = New("INFO", os.Stdout)

// SetDefaultLogger sets the default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// Debug logs a message at DEBUG level using the default logger
func Debug(format string, v ...interface{}) {
	defaultLogger.Debug(format, v...)
}

// Info logs a message at INFO level using the default logger
func Info(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

// Warn logs a message at WARN level using the default logger
func Warn(format string, v ...interface{}) {
	defaultLogger.Warn(format, v...)
}

// Error logs a message at ERROR level using the default logger
func Error(format string, v ...interface{}) {
	defaultLogger.Error(format, v...)
}

// Fatal logs a message at FATAL level using the default logger and exits the program
func Fatal(format string, v ...interface{}) {
	defaultLogger.Fatal(format, v...)
}
