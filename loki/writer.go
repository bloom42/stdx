package loki

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bloom42/stdx/httputils"
)

const defaultRecordsBufferSize = 100

type WriterOptions struct {
	// The loki enpoint
	Endpoint    string
	ChildWriter io.Writer
	Streams     map[string]string
	Ctx         context.Context
}

type Writer struct {
	lokiEndpoint string
	streams      map[string]string

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

func NewWriter(options WriterOptions) *Writer {
	streams := options.Streams
	if streams == nil {
		streams = map[string]string{}
	}

	if options.ChildWriter == nil {
		options.ChildWriter = os.Stdout
	}

	if options.Ctx == nil {
		options.Ctx = context.Background()
	}

	handler := &Writer{
		lokiEndpoint: options.Endpoint,
		streams:      streams,

		httpClient:         httputils.DefaultClient(),
		recordsBuffer:      make([]record, 0, defaultRecordsBufferSize),
		recordsBufferMutex: sync.Mutex{},
		childWriter:        options.ChildWriter,
		ctx:                options.Ctx,
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
			errFlushing := handler.flushLogs(context.Background())
			if errFlushing != nil {
				fmt.Println(errFlushing)
			}
		}
	}()

	return handler
}

// SetEndpoint sets the loki endpoint. This method IS NOT thread safe.
// It should be used just after config is loaded
func (writer *Writer) SetEndpoint(lokiEndpoint string) {
	writer.lokiEndpoint = lokiEndpoint
}

func (writer *Writer) Write(data []byte) (n int, err error) {
	// TODO: handle error?
	_, _ = writer.childWriter.Write(data)

	if writer.lokiEndpoint == "" {
		return
	}

	// if log finishes by '\n' we trim it
	data = bytes.TrimSuffix(data, []byte("\n"))

	record := record{
		timestamp: time.Now().UTC(),
		message:   string(data),
	}

	writer.recordsBufferMutex.Lock()
	writer.recordsBuffer = append(writer.recordsBuffer, record)
	writer.recordsBufferMutex.Unlock()

	return
}
