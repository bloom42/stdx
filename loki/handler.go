package loki

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bloom42/stdx/httputils"
)

const defaultRecordsBufferSize = 100

type Options struct {
	// The loki enpoint
	Endpoint string
	// Don't print the logs to stdout
	Silent  bool
	Json    bool
	Streams map[string]string
	Level   slog.Leveler
	Ctx     context.Context
}

type Handler struct {
	endpoint string
	streams  map[string]string
	level    slog.Leveler

	httpClient         *http.Client
	recordsBuffer      []record
	recordsBufferMutex sync.Mutex
	childHandler       slog.Handler
	writer             io.Writer
	stopped            atomic.Bool
	ctx                context.Context
}

type record struct {
	timestamp time.Time
	message   string
}

func NewHandler(options Options) *Handler {
	streams := options.Streams
	if streams == nil {
		streams = map[string]string{}
	}

	var writer io.Writer = os.Stdout
	if options.Silent {
		writer = io.Discard
	}

	if options.Level == nil {
		options.Level = slog.LevelInfo
	}

	if options.Ctx == nil {
		options.Ctx = context.Background()
	}

	handler := &Handler{
		endpoint: options.Endpoint,
		writer:   writer,
		streams:  streams,
		level:    options.Level,

		httpClient:         httputils.DefaultClient(),
		recordsBuffer:      make([]record, 0, defaultRecordsBufferSize),
		recordsBufferMutex: sync.Mutex{},
		childHandler:       nil,
		stopped:            atomic.Bool{},
		ctx:                options.Ctx,
	}
	handler.stopped.Store(false)

	if options.Json {
		handler.childHandler = slog.NewJSONHandler(handler, nil)
	} else {
		handler.childHandler = slog.NewTextHandler(handler, nil)
	}

	go func() {
		sleepFor := 100 * time.Millisecond
		done := false
		for {
			if !done {
				select {
				case <-handler.ctx.Done():
					done = true
					// we sleep less to avoid losing logs
					sleepFor = 10 * time.Millisecond
				default:
				}
			}
			<-time.After(sleepFor)
			handler.flushLogs(context.Background())
		}
	}()

	return handler
}

func (handler *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= handler.level.Level()
}

func (handler *Handler) Handle(ctx context.Context, slogRecord slog.Record) error {
	return handler.childHandler.Handle(ctx, slogRecord)
}

func (handler *Handler) Write(data []byte) (n int, err error) {
	// TODO: handle error?
	_, _ = handler.writer.Write(data)
	record := record{
		timestamp: time.Now().UTC(),
		message:   string(data),
	}

	handler.recordsBufferMutex.Lock()
	handler.recordsBuffer = append(handler.recordsBuffer, record)
	handler.recordsBufferMutex.Unlock()

	return
}

// TODO: make copy?
func (handler *Handler) WithGroup(name string) slog.Handler {
	handler.childHandler = handler.childHandler.WithGroup(name)
	return handler
}

// TODO: make copy?
func (handler *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handler.childHandler = handler.childHandler.WithAttrs(attrs)
	return handler
}
