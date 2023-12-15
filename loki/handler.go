package loki

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bloom42/stdx/httputils"
)

const defaultRecordsBufferSize = 100

type Options struct {
	// The loki enpoint
	Endpoint string
	// also prints the logs to stdout
	Silent  bool
	Streams map[string]string
	Level   slog.Level
}

type Handler struct {
	endpoint string
	stdout   io.Writer
	streams  map[string]string
	level    slog.Level

	httpClient         *http.Client
	recordsBuffer      []record
	recordsBufferMutex sync.Mutex
	textHandler        *slog.TextHandler
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

	var stdout io.Writer = os.Stdout
	if options.Silent {
		stdout = io.Discard
	}

	handler := &Handler{
		endpoint: options.Endpoint,
		stdout:   stdout,
		streams:  streams,
		level:    options.Level,

		httpClient:         httputils.DefaultClient(),
		recordsBuffer:      make([]record, 0, defaultRecordsBufferSize),
		recordsBufferMutex: sync.Mutex{},
		textHandler:        nil,
	}

	// TOO: circular reference: how it is for garbage collection?
	handler.textHandler = slog.NewTextHandler(handler, &slog.HandlerOptions{Level: handler.level})

	return handler
}

func (handler *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= handler.level
}

func (handler *Handler) Handle(ctx context.Context, slogRecord slog.Record) error {
	return handler.textHandler.Handle(ctx, slogRecord)
}

func (handler *Handler) Write(data []byte) (n int, err error) {
	// TODO: handle error?
	_, _ = handler.stdout.Write(data)
	record := record{
		timestamp: time.Now().UTC(),
		message:   string(data),
	}

	handler.recordsBufferMutex.Lock()
	handler.recordsBuffer = append(handler.recordsBuffer, record)
	handler.recordsBufferMutex.Unlock()

	return
}
