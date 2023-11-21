package selfupdate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bloom42/stdx/filesutil"
	"github.com/bloom42/stdx/zign"
)

type CreateReleaseInput struct {
	// Name of the project. e.g. myapp
	Name string
	// Version of the release of the project. e.g. 1.1.52
	Version string
	Channel string
	Files   []string
	// ZignPrivateKey is the base64 encoded zign privateKey, encrypted with password
	ZignPrivateKey string
	ZignPassword   string
}

type Release struct {
	Name            string
	ChannelManifest ChannelManifest
	ZignManifest    zign.Manifest
}

func CreateRelease(ctx context.Context, info CreateReleaseInput) (release Release, err error) {
	release = Release{
		Name: info.Name,
		ChannelManifest: ChannelManifest{
			Name:    info.Name,
			Channel: info.Channel,
			Version: info.Version,
		},
	}

	signInput := make([]zign.SignInput, len(info.Files))
	for index, file := range info.Files {
		fileExists := false
		var fileHandle *os.File
		filename := filepath.Base(file)

		fileExists, err = filesutil.Exists(file)
		if err != nil {
			err = fmt.Errorf("selfupdate: checking if file exists (%s): %w", file, err)
			return
		}

		if !fileExists {
			err = fmt.Errorf("selfupdate: file does not exist: %s", file)
			return
		}

		fileHandle, err = os.Open(file)
		if err != nil {
			err = fmt.Errorf("opening file (%s): %w", file, err)
			return
		}
		defer func(fileToClose *os.File) {
			fileToClose.Close()
		}(fileHandle)
		fileSignInput := zign.SignInput{
			Filename: filename,
			Reader:   fileHandle,
		}
		signInput[index] = fileSignInput
	}

	signatures, err := zign.SignMany(info.ZignPrivateKey, info.ZignPassword, signInput)
	if err != nil {
		return
	}

	release.ZignManifest = zign.GenerateManifest(signatures)

	return
}
