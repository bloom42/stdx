package loki

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bloom42/stdx/httpx"
)

type WriterOptions struct {
	// The loki enpoint
	LokiEndpoint string
	ChildWriter  io.Writer
	// DefaultRecordsBufferSize is your number of `(logs per second) / 5`
	DefaultRecordsBufferSize   uint32
	EmptyEndpointMaxBufferSize uint32
}

type Writer struct {
	lokiEndpoint               string
	streams                    map[string]string
	defaultRecordsBufferSize   uint32
	emptyEndpointMaxBufferSize uint32

	httpClient         *http.Client
	recordsBuffer      []record
	recordsBufferMutex sync.Mutex
	childWriter        io.Writer
	ctx                context.Context
}

type record struct {
	timestamp time.Time
	message   string
}

func NewWriter(ctx context.Context, lokiEndpoint string, streams map[string]string, options *WriterOptions) *Writer {
	if streams == nil {
		streams = map[string]string{}
	}

	defaultOptions := defaultOptions()
	if options == nil {
		options = defaultOptions
	} else {
		if options.ChildWriter == nil {
			options.ChildWriter = defaultOptions.ChildWriter
		}
		if options.DefaultRecordsBufferSize == 0 {
			options.DefaultRecordsBufferSize = defaultOptions.DefaultRecordsBufferSize
		}
		if options.EmptyEndpointMaxBufferSize == 0 {
			options.EmptyEndpointMaxBufferSize = defaultOptions.EmptyEndpointMaxBufferSize
		}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	handler := &Writer{
		lokiEndpoint:               lokiEndpoint,
		streams:                    streams,
		defaultRecordsBufferSize:   options.DefaultRecordsBufferSize,
		emptyEndpointMaxBufferSize: options.EmptyEndpointMaxBufferSize,

		httpClient:         httpx.DefaultClient(),
		recordsBuffer:      make([]record, 0, options.DefaultRecordsBufferSize),
		recordsBufferMutex: sync.Mutex{},
		childWriter:        options.ChildWriter,
		ctx:                ctx,
	}

	go func() {
		done := false
		for {
			if !done {
				select {
				case <-handler.ctx.Done():
					done = true
				case <-time.After(200 * time.Millisecond):
				}
			} else {
				// we sleep less to avoid losing logs
				time.Sleep(20 * time.Millisecond)
			}

			go func() {
				// TODO: as of now, if the HTTP request fail after X retries, we discard/lose the logs
				err := handler.flushLogs(context.Background())
				if err != nil {
					log.Println(err.Error())
					return
				}
				// if err != nil {
				// 		writer.recordsBufferMutex.Lock()
				// 		writer.recordsBuffer = append(writer.recordsBuffer, recordsBufferCopy...)
				// 		writer.recordsBufferMutex.Unlock()
				// 	}
			}()
		}
	}()

	return handler
}

func defaultOptions() *WriterOptions {
	return &WriterOptions{
		LokiEndpoint:               "",
		ChildWriter:                os.Stdout,
		DefaultRecordsBufferSize:   100,
		EmptyEndpointMaxBufferSize: 200,
	}
}

// SetEndpoint sets the loki endpoint. This method IS NOT thread safe.
// It should be used just after config is loaded
func (writer *Writer) SetEndpoint(lokiEndpoint string) {
	writer.lokiEndpoint = lokiEndpoint
}

func (writer *Writer) Write(data []byte) (n int, err error) {
	// TODO: handle error?
	_, _ = writer.childWriter.Write(data)

	// if log finishes by '\n' we trim it
	data = bytes.TrimSuffix(data, []byte("\n"))

	record := record{
		timestamp: time.Now().UTC(),
		message:   string(data),
	}

	writer.recordsBufferMutex.Lock()
	if writer.lokiEndpoint != "" || len(writer.recordsBuffer) < int(writer.emptyEndpointMaxBufferSize) {
		writer.recordsBuffer = append(writer.recordsBuffer, record)
	}
	writer.recordsBufferMutex.Unlock()

	return
}
