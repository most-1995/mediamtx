package test

import "github.com/most-1995/mediamtx/internal/logger"

// NilLogger is a logger to /dev/null
type NilLogger struct{}

// Log implements logger.Writer.
func (NilLogger) Log(_ logger.Level, _ string, _ ...interface{}) {
}
