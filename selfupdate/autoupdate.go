package selfupdate

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/bloom42/stdx/slogutil"
)

const (
	updatedExecutableOpenFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
)

func (updater *Updater) RunAutoupdateInBackground(ctx context.Context) {
	logger := slogutil.FromCtx(ctx)
	var err error
	var manifest ChannelManifest

	for {
		// sleep for autoupdateInterval + 10 seconds jitter
		waitFor := rand.Int63n(10) + updater.autoupdateInterval

		select {
		case <-ctx.Done():
			if updater.verbose {
				logger.Info("selfupdate: stopping")
			}

			return
		case <-time.After(time.Duration(waitFor) * time.Second):
			if updater.verbose {
				logger.Info("selfupdate: checking for update")
			}

			manifest, err = updater.CheckUpdate(ctx)
			if err != nil {
				logger.Warn("selfupdate: error while checking for update", slogutil.Err(err))
				continue
			}

			if updater.UpdateAvailable() {
				logger = logger.With(slog.String("selfupdate_new_version", manifest.Version))
				if updater.verbose {
					logger.Info("selfupdate: a new update is available")
				}

				err = updater.Update(ctx, manifest)
				if err != nil {
					logger.Warn("selfupdate: installing new version", slogutil.Err(err))
					continue
				}

				if updater.verbose {
					logger.Info("selfupdate: new version successfully installed")
				}

				updater.Updated <- struct{}{}
			}
		}
	}
}
