package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// Logger provides structured logging capabilities
type Logger struct {
	logger *log.Logger
	level  LogLevel
}

// New creates a new Logger instance
func New() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
		level:  LevelInfo,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

// formatMessage creates a structured log message
func (l *Logger) formatMessage(level LogLevel, message string, fields map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf("[%s] %s: %s", timestamp, level, message)
	
	if len(fields) > 0 {
		msg += " |"
		for key, value := range fields {
			msg += fmt.Sprintf(" %s=%v", key, value)
		}
	}
	
	return msg
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	if l.shouldLog(LevelDebug) {
		l.logger.Println(l.formatMessage(LevelDebug, message, fields))
	}
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	if l.shouldLog(LevelInfo) {
		l.logger.Println(l.formatMessage(LevelInfo, message, fields))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	if l.shouldLog(LevelWarn) {
		l.logger.Println(l.formatMessage(LevelWarn, message, fields))
	}
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	if l.shouldLog(LevelError) {
		l.logger.Println(l.formatMessage(LevelError, message, fields))
	}
}

// Global logger instance
var globalLogger = New()

// SetGlobalLevel sets the log level for the global logger
func SetGlobalLevel(level LogLevel) {
	globalLogger.SetLevel(level)
}

// Debug logs a debug message using the global logger
func Debug(message string, fields map[string]interface{}) {
	globalLogger.Debug(message, fields)
}

// Info logs an info message using the global logger
func Info(message string, fields map[string]interface{}) {
	globalLogger.Info(message, fields)
}

// Warn logs a warning message using the global logger
func Warn(message string, fields map[string]interface{}) {
	globalLogger.Warn(message, fields)
}

// Error logs an error message using the global logger
func Error(message string, fields map[string]interface{}) {
	globalLogger.Error(message, fields)
}