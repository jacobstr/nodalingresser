package logger

import (
	"go.uber.org/zap"
)

// DefaultLogger is the default logger used internally within the package
// when no logger is injected. Loggers are generally injected via corresponding
// WithLogger option functions.
var DefaultLogger *zap.Logger

func init() {
	DefaultLogger, _ = zap.NewDevelopment()
}
