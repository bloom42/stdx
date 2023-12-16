package selfupdate

import (
	"errors"
	"net/http"
	"sync"

	"github.com/bloom42/stdx/httpx"
	"github.com/bloom42/stdx/semver"
)

type Config struct {
	ZingPublicKey string
	// BaseURL is the URL of the folder containing the manifest
	// e.g. https://downloads.example.com/myapp
	BaseURL        string
	CurrentVersion string
	ReleaseChannel string

	// Default: 300 seconds
	AutoupdateInterval int64
	// Verbose logs actions with the INFO level
	Verbose    bool
	UserAgent  *string
	HttpClient *http.Client
}

type Updater struct {
	httpClient             *http.Client
	baseUrl                string
	zingPublicKey          string
	currentVersion         string
	userAgent              string
	releaseChannel         string
	updateInProgress       sync.Mutex
	latestVersionAvailable string
	latestVersionInstalled string
	autoupdateInterval     int64
	verbose                bool

	Updated chan struct{}
}

func NewUpdater(config Config) (updater *Updater, err error) {
	if config.HttpClient == nil {
		config.HttpClient = httpx.DefaultClient()
	}

	if config.BaseURL == "" {
		err = errors.New("selfupdate: BaseURL is empty")
		return
	}

	if config.ZingPublicKey == "" {
		err = errors.New("selfupdate: ZingPublicKey is empty")
		return
	}

	if config.CurrentVersion == "" {
		err = errors.New("selfupdate: CurrentVersion is empty")
		return
	}

	if config.AutoupdateInterval == 0 {
		config.AutoupdateInterval = 300
	}

	userAgent := DefaultUserAgent
	if config.UserAgent != nil {
		userAgent = *config.UserAgent
	}

	updater = &Updater{
		httpClient:             config.HttpClient,
		baseUrl:                config.BaseURL,
		zingPublicKey:          config.ZingPublicKey,
		currentVersion:         config.CurrentVersion,
		userAgent:              userAgent,
		releaseChannel:         config.ReleaseChannel,
		updateInProgress:       sync.Mutex{},
		latestVersionAvailable: config.CurrentVersion,
		latestVersionInstalled: config.CurrentVersion,
		autoupdateInterval:     config.AutoupdateInterval,
		verbose:                config.Verbose,
		Updated:                make(chan struct{}),
	}
	return
}

func (updater *Updater) RestartRequired() bool {
	return updater.latestVersionInstalled != updater.currentVersion
}

// UpdateAvailable returns true if the latest avaiable version is > to the latest install version
func (updater *Updater) UpdateAvailable() bool {
	return semver.Compare(updater.latestVersionAvailable, updater.latestVersionInstalled) > 0
}
