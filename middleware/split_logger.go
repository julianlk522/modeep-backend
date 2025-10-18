package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	FileLogFormatter *splitLogFormatter
)

func init() {
	var err error
	FileLogFormatter, err = newSplitLogFormatter(
		log.New(
			os.Stdout,
			"",
			log.LstdFlags,
		),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func SplitRequestLogger(f *splitLogFormatter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := f.newLogEntry(r)
			t1 := time.Now()

			crw := &responseWriterWithStatusText{ResponseWriter: w}
			ww := middleware.NewWrapResponseWriter(crw, r.ProtoMajor)

			defer func() {
				entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(t1), crw)
			}()

			next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
		}
		return http.HandlerFunc(fn)
	}
}

type splitLogFormatter struct {
	middleware.DefaultLogFormatter
	FileLogger *log.Logger
}

func (slf *splitLogFormatter) newLogEntry(r *http.Request) middleware.LogEntry {
	return &splitLogEntry{
		LogEntry:  slf.DefaultLogFormatter.NewLogEntry(r),
		Formatter: slf,
		Request:   r,
	}
}

func newSplitLogFormatter(logger middleware.LoggerInterface) (*splitLogFormatter, error) {
	err_log_file_path := os.Getenv("MODEEP_ERR_LOG_FILE")
	if err_log_file_path == "" {
		return nil, fmt.Errorf("MODEEP_ERR_LOG_FILE not set")
	}
	log_file, err := os.OpenFile(err_log_file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Could not open err log file: %s", err)
		return nil, err
	}
	log.Printf("Logging errs to %s", err_log_file_path)

	return &splitLogFormatter{
		DefaultLogFormatter: middleware.DefaultLogFormatter{
			Logger:  logger,
			NoColor: false,
		},
		FileLogger: log.New(log_file, "", log.LstdFlags),
	}, nil
}

type splitLogEntry struct {
	middleware.LogEntry
	Formatter *splitLogFormatter
	Request   *http.Request
}

func (sle *splitLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra any) {
	sle.LogEntry.Write(status, bytes, header, elapsed, extra)

	// save GitHub webhook responses
	if sle.Request.URL.Path == "/ghwh" {
		status_text := "Unknown"
		if crw, ok := extra.(*responseWriterWithStatusText); ok {
			status_text = crw.StatusText
		}

		sle.Formatter.FileLogger.Printf(
			"GitHub Webhook: %d %s %s %s %s %dB %v\nStatus Text: %s",
			status,
			sle.Request.Method,
			sle.Request.URL.Path,
			sle.Request.RemoteAddr,
			sle.Request.Header.Get("X-GitHub-Event"),
			bytes,
			elapsed,
			status_text,
		)

		// save errors
	} else if status > 299 {
		status_text := "Unknown Error"
		if crw, ok := extra.(*responseWriterWithStatusText); ok {
			status_text = crw.StatusText
		}
		sle.Formatter.FileLogger.Printf(
			"Err: %d %s %s %s %dB %v\nStatus Text: %s",
			status,
			sle.Request.Method,
			sle.Request.URL.Path,
			sle.Request.RemoteAddr,
			bytes,
			elapsed,
			status_text,
		)
	}
}

type responseWriterWithStatusText struct {
	http.ResponseWriter
	StatusText string
}

func (crw *responseWriterWithStatusText) Write(b []byte) (int, error) {
	if crw.StatusText == "" {
		crw.StatusText = string(b)
	}
	return crw.ResponseWriter.Write(b)
}
