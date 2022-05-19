package buildah

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
)

func inspectImage(ctx context.Context, runtime *libimage.Runtime, image string) (*libimage.ImageData, error) {
	img, _, err := runtime.LookupImage(image, nil)
	if err != nil {
		if unwrapedError := errors.Unwrap(err); unwrapedError != nil {
			err = unwrapedError
		}
		if errors.Is(err, storage.ErrImageUnknown) {
			// try pulling if the image wasn't found
			images, err := runtime.Pull(ctx, image, config.PullPolicyMissing, nil)
			if err != nil {
				return nil, fmt.Errorf("image %v was not found locally and pulling was not successfull too: %w", image, err)
			}
			log.New().Debug(images)
		} else {
			return nil, fmt.Errorf("lookup image %v in store: %w", image, err)
		}
	}
	data, err := img.Inspect(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("inspecting image %v: %w", image, err)
	}
	return data, nil
}

func runtimeFromStore(store storage.Store) (*libimage.Runtime, error) {
	runtime, err := libimage.RuntimeFromStore(store, nil)
	if err != nil {
		return nil, fmt.Errorf("creating libimage runtime: %w", err)
	}
	return runtime, nil
}
