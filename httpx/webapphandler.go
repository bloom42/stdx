package httpx

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var ErrDir = errors.New("path is a folder")
var ErrInvalidPath = errors.New("path is not valid")
var ErrInternalError = errors.New("Internal Server Error")
var errFileIsMissing = func(file string) error { return fmt.Errorf("webappHandler: %s is missing", file) }

type fileMetadata struct {
	contentType string
	etag        string
	// we store the contentLength as a string to avoid the conversion to string for each request
	contentLength string
	cacheControl  string
}

type WebappHandlerConfig struct {
	// default: index.html
	FileNotFound string
	// default: 200
	StatusNotFound int
	// default: ".js", ".css", ".woff", ".woff2"
	ImmutableFilesExtensions []string
}

// WebappHandler is an http.Handler that is designed to efficiently serve Single Page Applications.
// if a file is not found, it will return notFoundFile (default: index.html) with the stauscode statusNotFound
// WebappHandler sets the correct ETag header and cache the hash of files so that repeated requests
// to files return only StatusNotModified responses
// WebappHandler returns StatusMethodNotAllowed if the method is different than GET or HEAD
func WebappHandler(folder fs.FS, config *WebappHandlerConfig) (func(w http.ResponseWriter, r *http.Request), error) {
	defaultConfig := defaultWebappHandlerConfig()
	if config == nil {
		config = defaultConfig
	} else {
		if config.FileNotFound == "" {
			config.FileNotFound = defaultConfig.FileNotFound
		}
		if config.StatusNotFound == 0 {
			config.StatusNotFound = defaultConfig.StatusNotFound
		}
		if config.ImmutableFilesExtensions == nil {
			config.ImmutableFilesExtensions = defaultConfig.ImmutableFilesExtensions
		}
	}

	filesMetadata, err := loadFilesMetdata(folder, config)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed.\n"))
			return
		}

		statusCode := http.StatusOK
		path := strings.TrimPrefix(req.URL.Path, "/")
		fileMetadata, fileExists := filesMetadata[path]
		cacheControl := fileMetadata.cacheControl
		if !fileExists {
			path = config.FileNotFound
			fileMetadata = filesMetadata[path]
			statusCode = config.StatusNotFound
			cacheControl = CacheControlNoCache
		} else {
			w.Header().Set(HeaderETag, fileMetadata.etag)
		}

		w.Header().Set(HeaderContentLength, fileMetadata.contentLength)
		w.Header().Set(HeaderContentType, fileMetadata.contentType)
		w.Header().Set(HeaderCacheControl, cacheControl)

		requestEtag := cleanRequestEtag(req.Header.Get(HeaderIfNoneMatch))
		if fileExists && requestEtag == fileMetadata.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.WriteHeader(statusCode)
		err = sendFile(folder, path, w)
		if err != nil {
			w.Header().Set(HeaderCacheControl, CacheControlNoCache)
			handleError(http.StatusInternalServerError, ErrInternalError.Error(), w)
			return
		}
	}, nil
}

func defaultWebappHandlerConfig() *WebappHandlerConfig {
	return &WebappHandlerConfig{
		FileNotFound:             "index.html",
		StatusNotFound:           200,
		ImmutableFilesExtensions: []string{".js", ".css", ".woff", ".woff2"},
	}
}

func sendFile(folder fs.FS, path string, w http.ResponseWriter) (err error) {
	file, err := folder.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	_, err = io.Copy(w, file)
	return
}

func handleError(code int, message string, w http.ResponseWriter) {
	http.Error(w, message, code)
}

// sometimes, a CDN may add the weak Etag prefix: W/
func cleanRequestEtag(requestEtag string) string {
	return strings.TrimPrefix(strings.TrimSpace(requestEtag), "W/")
}

func loadFilesMetdata(folder fs.FS, config *WebappHandlerConfig) (ret map[string]fileMetadata, err error) {
	ret = make(map[string]fileMetadata, 10)

	err = fs.WalkDir(folder, ".", func(path string, fileEntry fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			return fmt.Errorf("webappHandler: error processing file %s: %w", path, errWalk)
		}

		if fileEntry.IsDir() || !fileEntry.Type().IsRegular() {
			return nil
		}

		fileInfo, errWalk := fileEntry.Info()
		if errWalk != nil {
			return fmt.Errorf("webappHandler: error getting info for file %s: %w", path, errWalk)
		}

		file, errWalk := folder.Open(path)
		if err != nil {
			return fmt.Errorf("webappHandler: error opening file %s: %w", path, errWalk)
		}
		defer file.Close()

		// we hash the file to generate its Etag
		hasher := sha256.New()
		_, errWalk = io.Copy(hasher, file)
		if errWalk != nil {
			return fmt.Errorf("webappHandler: error hashing file %s: %w", path, errWalk)
		}
		fileHash := hasher.Sum(nil)

		etag := encodeEtag(fileHash)

		extension := filepath.Ext(path)
		contentType := mime.TypeByExtension(extension)

		// the cacheControl value depends on the type of the file
		cacheControl := CacheControlDynamic
		// some webapp's assets files can be cached for very long time because they are versionned by
		// the webapp's bundler
		if slices.Contains(config.ImmutableFilesExtensions, extension) {
			cacheControl = CacheControlImmutable
		}

		ret[path] = fileMetadata{
			contentType:   contentType,
			etag:          etag,
			contentLength: strconv.FormatInt(fileInfo.Size(), 10),
			cacheControl:  cacheControl,
		}

		return nil
	})

	if _, indexHtmlExists := ret[config.FileNotFound]; !indexHtmlExists {
		err = errFileIsMissing(config.FileNotFound)
		return
	}

	return
}

func encodeEtag(hash []byte) string {
	return `"` + base64.RawURLEncoding.EncodeToString(hash) + `"`
}
