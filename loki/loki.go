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
func (writer *Writer) flushLogs(ctx context.Context) (err error) {
	writer.recordsBufferMutex.Lock()

	if len(writer.recordsBuffer) == 0 {
		writer.recordsBufferMutex.Unlock()
		return
	}

	recordsBufferCopy := make([]record, len(writer.recordsBuffer))
	copy(recordsBufferCopy, writer.recordsBuffer)
	writer.recordsBuffer = make([]record, 0, defaultRecordsBufferSize)
	writer.recordsBufferMutex.Unlock()

	if writer.lokiEndpoint == "" {
		return
	}

	// TODO: as of now, if the HTTP request fail, we discard/lose the logs
	// defer func() {
	// 	if err != nil {
	// 		writer.recordsBufferMutex.Lock()
	// 		writer.recordsBuffer = append(writer.recordsBuffer, recordsBufferCopy...)
	// 		writer.recordsBufferMutex.Unlock()
	// 	}
	// }()

	// Marshal records to gzipped JSON
	payload := convertRecords(writer.streams, recordsBufferCopy)
	// body := bytes.NewBuffer(make([]byte, len(recordsBufferCopy)*100))
	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	jsonEncoder := json.NewEncoder(gzipWriter)
	err = jsonEncoder.Encode(payload)
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: error encoding JSON: %w", err)
		return
	}
	err = gzipWriter.Close()
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: error closing the Gzip writer: %w", err)
		return
	}

	pushLogsEndpoint := writer.lokiEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushLogsEndpoint, &body)
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: error creating HTTP request: %w", err)
		return
	}

	req.Header.Add(httputils.HeaderContentType, httputils.MediaTypeJson)
	req.Header.Add(httputils.HeaderContentEncoding, "gzip")

	res, err := writer.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("loki: flushing logs: making HTTP request: %w", err)
		return err
	}
	var resBuffer bytes.Buffer
	_, _ = io.Copy(&resBuffer, res.Body)
	res.Body.Close()
	fmt.Println("LOGS SENT")
	fmt.Println(resBuffer.String())

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
