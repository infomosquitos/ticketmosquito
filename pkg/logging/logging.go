package logging

import "go.uber.org/zap"

// Sugar wraps a named *zap.SugaredLogger.
type Sugar struct {
	*zap.SugaredLogger
}

// New creates a named sugared logger for a package.
func New(name string, logger *zap.Logger) Sugar {
	return Sugar{
		SugaredLogger: logger.Named(name).Sugar(),
	}
}

// Enabled reports whether the underlying logger is non-nil.
func (s Sugar) Enabled() bool {
	return s.SugaredLogger != nil
}
