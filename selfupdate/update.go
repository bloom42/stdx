package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bloom42/stdx/httputils"
	"github.com/bloom42/stdx/semver"
	"github.com/bloom42/stdx/slogutil"
	"github.com/bloom42/stdx/zign"
)

func (updater *Updater) CheckUpdate(ctx context.Context) (manifest ChannelManifest, err error) {
	logger := slogutil.FromCtx(ctx)

	manifestUrl := fmt.Sprintf("%s/%s.json", updater.baseUrl, updater.releaseChannel)

	logger.Debug("selfupdate.CheckUpdate: fetching channel manifest", slog.String("url", manifestUrl))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestUrl, nil)
	if err != nil {
		err = fmt.Errorf("selfupdate.CheckUpdate: creating channel manifest HTTP request: %w", err)
		return
	}

	req.Header.Add(httputils.HeaderAccept, httputils.MediaTypeJson)
	req.Header.Add(httputils.HeaderUserAgent, updater.userAgent)

	res, err := updater.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("selfupdate.CheckUpdate: fetching channel manifest: %w", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("selfupdate.CheckUpdate: Status code is not 200 when fetching channel manifest: %d", res.StatusCode)
		return

	}

	mainfestData, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("selfupdate.CheckUpdate: Reading manifest response: %d", res.StatusCode)
		return
	}

	err = json.Unmarshal(mainfestData, &manifest)
	if err != nil {
		err = fmt.Errorf("selfupdate.CheckUpdate: parsing manifest: %w", err)
		return
	}

	if !semver.IsValid(manifest.Version) {
		err = fmt.Errorf("selfupdate.CheckUpdate: version (%s) is not a valid semantic version string", manifest.Version)
		return
	}

	updater.latestVersionAvailable = manifest.Version

	return
}

func (updater *Updater) Update(ctx context.Context, channelManifest ChannelManifest) (err error) {
	updater.updateInProgress.Lock()
	defer updater.updateInProgress.Unlock()

	zignManifest, err := updater.fetchZignManifest(ctx, channelManifest)
	if err != nil {
		return
	}

	tmpDir, err := os.MkdirTemp("", channelManifest.Name+"_autoupdate_"+channelManifest.Version)
	if err != nil {
		err = fmt.Errorf("selfupdate: creating temporary directory: %w", err)
		return
	}
	destPath := filepath.Join(tmpDir, channelManifest.Name)

	platform := runtime.GOOS + "_" + runtime.GOARCH

	artifactExists := false
	var artifactToDownload zign.SignOutput
	for _, artifact := range zignManifest.Files {
		if strings.Contains(artifact.Filename, platform) {
			artifactExists = true
			artifactToDownload = artifact
		}
	}
	if !artifactExists {
		err = fmt.Errorf("selfupdate: No file found for platform: %s", platform)
		return
	}

	artifactUrl := updater.baseUrl + "/" + channelManifest.Version + "/" + artifactToDownload.Filename

	res, err := updater.httpClient.Get(artifactUrl)
	if err != nil {
		err = fmt.Errorf("selfupdate: fetching artifact: %w", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("selfupdate: Status code is not 200 when fetching artifact: %d", res.StatusCode)
		return
	}

	artifactFile, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("selfupdate: reading artifact's response: %d", res.StatusCode)
		return
	}

	artifactFileReader := bytes.NewReader(artifactFile)

	sha256Hash, err := hex.DecodeString(artifactToDownload.Sha256)
	if err != nil {
		err = fmt.Errorf("selfupdate: decoding hash for file %s: %w", artifactToDownload.Filename, err)
		return
	}

	zignVeryfiInput := zign.VerifyInput{
		Reader:    artifactFileReader,
		Sha256:    sha256Hash,
		Signature: artifactToDownload.Signature,
	}
	err = zign.Verify(updater.zingPublicKey, zignVeryfiInput)
	if err != nil {
		err = fmt.Errorf("selfupdate: verifying signature: %w", err)
		return
	}

	artifactFileReader.Seek(0, io.SeekStart)

	// handle both .tar.gz and .zip artifacts
	if strings.HasSuffix(artifactToDownload.Filename, ".tar.gz") {
		err = updater.extractTarGzArchive(artifactFileReader, destPath)
	} else if strings.HasSuffix(artifactToDownload.Filename, ".zip") {
		err = updater.extractZipArchive(artifactFileReader, int64(artifactFileReader.Len()), destPath)
	} else {
		err = fmt.Errorf("selfupdate: unsupported archive format: %s", filepath.Ext(artifactToDownload.Filename))
	}
	if err != nil {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		err = fmt.Errorf("selfupdate: getting current executable path: %w", err)
		return
	}

	err = os.Rename(destPath, execPath)
	if err != nil {
		err = fmt.Errorf("selfupdate: moving update to executable path: %w", err)
		return
	}

	updater.latestVersionInstalled = channelManifest.Version

	_ = os.RemoveAll(tmpDir)

	return
}

func (updater *Updater) extractTarGzArchive(dataReader io.Reader, destPath string) (err error) {
	gzipReader, err := gzip.NewReader(dataReader)
	if err != nil {
		err = fmt.Errorf("selfupdate: creating gzip reader: %w", err)
		return
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	fileToExtractHeader, err := tarReader.Next()
	if fileToExtractHeader == nil || err == io.EOF {
		err = errors.New("selfupdate: no file inside .tar.gz archive")
		return
	} else if err != nil {
		err = fmt.Errorf("selfupdate: reading .tar.gz archive: %w", err)
		return
	}

	if fileToExtractHeader.Typeflag != tar.TypeReg {
		err = fmt.Errorf("selfupdate: reading .tar.gz archive: %s is not a regular file", fileToExtractHeader.Name)
		return
	}

	updatedExecutable, err := os.OpenFile(destPath, updatedExecutableOpenFlags, fileToExtractHeader.FileInfo().Mode())
	if err != nil {
		err = fmt.Errorf("selfupdate: creating dest file (%s): %w", destPath, err)
		return
	}
	defer updatedExecutable.Close()

	_, err = io.Copy(updatedExecutable, tarReader)
	if err != nil {
		err = fmt.Errorf("selfupdate: extracting .tar.gzipped file (%s): %w", fileToExtractHeader.Name, err)
		return
	}

	return
}

func (updater *Updater) extractZipArchive(dataReader io.ReaderAt, dataLen int64, destPath string) (err error) {
	zipReader, err := zip.NewReader(dataReader, dataLen)
	if err != nil {
		err = fmt.Errorf("selfupdate: creating zip reader: %w", err)
		return
	}

	zippedFiles := zipReader.File
	if len(zippedFiles) != 1 {
		err = fmt.Errorf("selfupdate: zip archive contains more than 1 file (%d)", len(zippedFiles))
		return
	}

	zippedFileToExtract := zippedFiles[0]

	srcFile, err := zippedFileToExtract.Open()
	if err != nil {
		err = fmt.Errorf("selfupdate: Opening zipped file (%s): %w", zippedFileToExtract.Name, err)
		return
	}
	defer srcFile.Close()

	updatedExecutable, err := os.OpenFile(destPath, updatedExecutableOpenFlags, zippedFileToExtract.Mode())
	if err != nil {
		err = fmt.Errorf("selfupdate: creating dest file (%s): %w", destPath, err)
		return
	}
	defer updatedExecutable.Close()

	_, err = io.Copy(updatedExecutable, srcFile)
	if err != nil {
		err = fmt.Errorf("selfupdate: extracting zipped file (%s): %w", zippedFileToExtract.Name, err)
		return
	}

	return
}

func (updater *Updater) fetchZignManifest(ctx context.Context, channelManifest ChannelManifest) (zignManifest zign.Manifest, err error) {
	logger := slogutil.FromCtx(ctx)

	zignManifestUrl := fmt.Sprintf("%s/%s/zign.json", updater.baseUrl, channelManifest.Version)

	logger.Debug("selfupdate.fetchZignManifest: fetching zign manifest", slog.String("url", zignManifestUrl))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zignManifestUrl, nil)
	if err != nil {
		err = fmt.Errorf("selfupdate.fetchZignManifest: creating zign manifest HTTP request: %w", err)
		return
	}

	req.Header.Add(httputils.HeaderAccept, httputils.MediaTypeJson)
	req.Header.Add(httputils.HeaderUserAgent, updater.userAgent)

	res, err := updater.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("selfupdate.fetchZignManifest: fetching zign manifest: %w", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("selfupdate.fetchZignManifest: Status code is not 200 when fetching zign manifest: %d", res.StatusCode)
		return

	}

	mainfestData, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("selfupdate.fetchZignManifest: Reading manifest response: %d", res.StatusCode)
		return
	}

	err = json.Unmarshal(mainfestData, &zignManifest)
	if err != nil {
		err = fmt.Errorf("selfupdate.fetchZignManifest: parsing zign manifest: %w", err)
		return
	}
	return
}
