package model

import (
	"time"
)

func init() {
	time.Local = time.UTC
}

var (
	NEW_LONG_TIMESTAMP  = func() string { return time.Now().Format("2006-01-02 15:04:05") }
	NEW_SHORT_TIMESTAMP = func() string { return time.Now().Format("2006-01-02") }
)
