package loki

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/bloom42/stdx/httputils"
)

type lokiPushRequest struct {
	Streams []lokiPushStream `json:"streams"`
}

type lokiPushStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// https://grafana.com/docs/loki/latest/reference/api/#push-log-entries-to-loki
func (handler *Handler) flushLogs(ctx context.Context) (err error) {
	fmt.Println("FLUSHING LOGS")
	return
	handler.recordsBufferMutex.Lock()

	if len(handler.recordsBuffer) == 0 {
		handler.recordsBufferMutex.Unlock()
		return
	}

	recordsBufferCopy := make([]record, len(handler.recordsBuffer))
	copy(recordsBufferCopy, handler.recordsBuffer)
	handler.recordsBuffer = make([]record, 0, defaultRecordsBufferSize)
	handler.recordsBufferMutex.Unlock()

	defer func() {
		if err != nil {
			handler.recordsBufferMutex.Lock()
			handler.recordsBuffer = append(handler.recordsBuffer, recordsBufferCopy...)
			handler.recordsBufferMutex.Unlock()
		}
	}()

	// Marshal records to gzipped JSON
	body := bytes.NewBuffer(make([]byte, len(recordsBufferCopy)*100))
	gzipWriter := gzip.NewWriter(body)
	jsonEncoder := json.NewEncoder(gzipWriter)
	jsonEncoder.Encode(convertRecords(handler.streams, recordsBufferCopy))
	err = gzipWriter.Close()
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: error closing the Gzip writer: %w", err)
		return
	}

	pushLogsEndpoint := handler.endpoint + "/loki/api/v1/push"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushLogsEndpoint, body)
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: error creating HTTP request: %w", err)
		return
	}

	req.Header.Add(httputils.HeaderContentType, httputils.MediaTypeJson)
	req.Header.Add(httputils.HeaderContentEncoding, "gzip")

	res, err := handler.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: making HTTP request: %w", err)
		return err
	}
	_, _ = io.Copy(io.Discard, res.Body)
	res.Body.Close()

	return
}

func convertRecords(streams map[string]string, records []record) lokiPushRequest {
	ret := lokiPushRequest{
		Streams: []lokiPushStream{
			{
				Stream: streams,
				Values: make([][]string, 0, len(records)),
			},
		},
	}

	for _, record := range records {
		lokiRecord := make([]string, 2)
		lokiRecord[0] = strconv.Itoa(int(record.timestamp.UnixNano()))
		lokiRecord[1] = record.message
		ret.Streams[0].Values = append(ret.Streams[0].Values, lokiRecord)
	}

	return ret
}
