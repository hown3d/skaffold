package buildah

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/docker/distribution/reference"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Buildah interface {
	BuildDockerfile(ctx context.Context, dockerfilePath string, contextDir string, options define.BuildOptions) (id string, ref reference.Canonical, err error)
	Push(ctx context.Context, imageID string, imageName string, out io.Writer) (reference.Canonical, error)
	GetConfigFile(ctx context.Context, imageName string) (*v1.ConfigFile, error)
}

type Client struct {
	buildStore      storage.Store
	libImageRuntime *libimage.Runtime
	config          *config.Config
}

func NewBuildah() (Buildah, error) {
	buildStore, err := newBuildStore()
	if err != nil {
		return Client{}, err
	}
	runtime, err := libimageRuntimeFromStore(buildStore)
	if err != nil {
		return Client{}, err
	}
	config, err := defaultConfig()
	if err != nil {
		return Client{}, err
	}
	return Client{
		buildStore:      buildStore,
		libImageRuntime: runtime,
		config:          config,
	}, nil
}

func (c Client) Close() error {
	err := c.buildStoreShutdown(false)
	return err
}

func (c Client) GetConfigFile(ctx context.Context, imageName string) (*v1.ConfigFile, error) {
	imgData, err := c.inspectImage(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("inspecting image %v: %w", imageName, err)
	}
	return libImageDataToRegistryConfigFile(imgData)
}

func (c Client) BuildDockerfile(ctx context.Context, dockerfilePath string, contextDir string, options define.BuildOptions) (id string, ref reference.Canonical, err error) {
	path, err := GetDockerfilePath(contextDir, dockerfilePath)
	if err != nil {
		return "", nil, fmt.Errorf("getting full path for dockerfile %v in dir %v: %w", contextDir, dockerfilePath, err)
	}
	return imagebuildah.BuildDockerfiles(ctx, c.buildStore, options, path)
}

func (c Client) Push(ctx context.Context, imageID string, imageName string, out io.Writer) (reference.Canonical, error) {
	dest, err := alltransports.ParseImageName(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing image name: %w", err)
	}
	options := buildah.PushOptions{
		Store: c.buildStore,
	}
	ref, _, err := buildah.Push(ctx, imageID, dest, options)
	if err != nil {
		return nil, fmt.Errorf("buildah push: %w", err)
	}
	return ref, nil
}

func (c Client) buildStoreShutdown(force bool) error {
	_, err := c.buildStore.Shutdown(force)
	return err
}

// GetDockerfilePath will get the absolute path to the specified docekrfile or defaults to "Dockerfile" if path is empty.
func GetDockerfilePath(contextDir string, dockerfilePath string) (string, error) {
	if dockerfilePath != "" {
		dockerfilePath = "Dockerfile"
	}
	dockerfile, err := docker.NormalizeDockerfilePath(contextDir, dockerfilePath)
	if err != nil {
		return "", fmt.Errorf("normalizing dockerfile path: %w", err)
	}
	return dockerfile, nil
}

func newBuildStore() (storage.Store, error) {
	buildStoreOptions, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
	if err != nil {
		return nil, fmt.Errorf("buildah store options: %w", err)
	}
	return storage.GetStore(buildStoreOptions)
}

func defaultConfig() (*config.Config, error) {
	return config.Default()
}
