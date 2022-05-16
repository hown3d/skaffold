package buildah

import (
	"context"
	"fmt"

	"github.com/containers/common/libimage"
	"github.com/containers/storage"
)

func inspectImage(ctx context.Context, runtime *libimage.Runtime, image string) (*libimage.ImageData, error) {
	img, _, err := runtime.LookupImage(image, nil)
	if err != nil {
		return nil, fmt.Errorf("lookup image %v in store: %w", image, err)
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
