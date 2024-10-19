package middleware

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var (
	FileLogFormatter *SplitLogFormatter
)

func init() {
	var err error
	FileLogFormatter, err = NewSplitLogFormatter(
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

// creates a new SplitLogFormatter
func NewSplitLogFormatter(logger middleware.LoggerInterface) (*SplitLogFormatter, error) {
	log_file, err := os.OpenFile(os.Getenv("FITM_ERR_LOG_FILE"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("unable to locate err log file: %s", err)
		return nil, err
	}
	log.Printf("logging errs to %s", os.Getenv("FITM_ERR_LOG_FILE"))

	return &SplitLogFormatter{
		DefaultLogFormatter: middleware.DefaultLogFormatter{
			Logger:  logger,
			NoColor: false,
		},
		FileLogger: log.New(log_file, "", log.LstdFlags),
	}, nil
}

// middleware that logs requests using SplitLogFormatter
func SplitRequestLogger(f *SplitLogFormatter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := f.NewLogEntry(r)

			// custom response writer to capture status text
			crw := &CustomResponseWriter{ResponseWriter: w}
			ww := middleware.NewWrapResponseWriter(crw, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				entry.Write(ww.Status(), ww.BytesWritten(), ww.Header(), time.Since(t1), crw)
			}()

			next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
		}
		return http.HandlerFunc(fn)
	}
}

// CustomResponseWriter wraps the http.ResponseWriter to capture any
// custom status text
type CustomResponseWriter struct {
	http.ResponseWriter
	StatusText string
}

func (crw *CustomResponseWriter) Write(b []byte) (int, error) {
	if crw.StatusText == "" {
		crw.StatusText = string(b)
	}
	return crw.ResponseWriter.Write(b)
}

// wraps DefaultLogFormatter to allow "teeing" err logs to file
type SplitLogFormatter struct {
	middleware.DefaultLogFormatter
	FileLogger *log.Logger
}

func (l *SplitLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &SplitLogEntry{
		LogEntry:  l.DefaultLogFormatter.NewLogEntry(r),
		Formatter: l,
		Request:   r,
	}
}

// wraps default log entry
type SplitLogEntry struct {
	middleware.LogEntry
	Formatter *SplitLogFormatter
	Request   *http.Request
}

func (l *SplitLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.LogEntry.Write(status, bytes, header, elapsed, extra)

	if status > 299 {
		status_text := "Unknown Error"
		if crw, ok := extra.(*CustomResponseWriter); ok {
			status_text = crw.StatusText
		}
		l.Formatter.FileLogger.Printf(
			"Err: %d %s %s %s %dB %v\nStatus Text: %s",
			status,
			l.Request.Method,
			l.Request.URL.Path,
			l.Request.RemoteAddr,
			bytes,
			elapsed,
			status_text,
		)
	}
}
