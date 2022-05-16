package buildah

import (
	"fmt"

	"github.com/containers/common/libimage"
	"github.com/containers/storage"
	"github.com/jinzhu/copier"

	registryV1 "github.com/google/go-containerregistry/pkg/v1"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Builder is an artifact builder that uses buildah
type Builder struct {
	pushImages  bool
	concurrency *int
	buildStore  storage.Store
}

const (
	gzipCompression  = "gzip"
	bzip2Compression = "bzip2"
	zstdCompression  = "zstd"
	xzCompression    = "xz"
	uncompressed     = "uncompressed"
)

func NewBuilder(pushImages bool, concurrency *int) (*Builder, error) {
	buildStore, err := newBuildStore()
	if err != nil {
		return nil, fmt.Errorf("creating buildah store: %w", err)
	}
	return &Builder{
		pushImages:  pushImages,
		concurrency: concurrency,
		buildStore:  buildStore,
	}, nil
}

func libImageDataToRegistryConfigFile(d *libimage.ImageData) (*registryV1.ConfigFile, error) {

	history, err := ociHistoryToRegistryHistory(d.History)
	if err != nil {
		return nil, fmt.Errorf("parsing oci history struct to registry history struct: %w", err)
	}
	rootFS, err := libimageRootFSToRegistryRootFS(d.RootFS)
	if err != nil {
		return nil, fmt.Errorf("parsing libimage rootfs struct to registry rootfs struct: %w", err)
	}
	config, err := ociConfigToRegistryConfig(d.Config)
	if err != nil {
		return nil, fmt.Errorf("parsing oci config struct to registry config struct: %w", err)
	}

	return &registryV1.ConfigFile{
		Architecture: d.Architecture,
		Author:       d.Author,
		Created: registryV1.Time{
			Time: *d.Created,
		},
		OS:      d.Os,
		Config:  *config,
		History: history,
		RootFS:  rootFS,
	}, nil
}

func ociHistoryToRegistryHistory(ociHistory []ociv1.History) ([]registryV1.History, error) {
	regHistory := make([]registryV1.History, len(ociHistory))
	err := copier.Copy(regHistory, ociHistory)
	return regHistory, err
	// regHistory := []registryV1.History{}
	// for _, hist := range ociHistory {
	// 	regHistory = append(regHistory, registryV1.History{
	// 		Author:    hist.Author,
	// 		Created:   registryV1.Time{*hist.Created},
	// 		CreatedBy: hist.CreatedBy,
	// 		Comment:   hist.Comment,
	// 	})
	// }
	// return regHistory
}

type libImageRootFS libimage.RootFS

func (f libImageRootFS) DiffIDs() []registryV1.Hash {
	hashes := []registryV1.Hash{}
	for _, layer := range f.Layers {
		hashes = append(hashes, registryV1.Hash{
			Algorithm: string(layer.Algorithm()),
			Hex:       layer.Encoded(),
		})
	}
	return hashes
}

func libimageRootFSToRegistryRootFS(rootfs *libimage.RootFS) (registryV1.RootFS, error) {
	libRootfs := libImageRootFS(*rootfs)
	regRootfs := new(registryV1.RootFS)
	err := copier.Copy(&regRootfs, &libRootfs)
	return *regRootfs, err
}

func ociConfigToRegistryConfig(ociConfig *ociv1.ImageConfig) (*registryV1.Config, error) {
	regConfig := new(registryV1.Config)
	err := copier.Copy(ociConfig, regConfig)
	return regConfig, err
	// return registryV1.Config{
	// 	Cmd:          ociConfig.Cmd,
	// 	StopSignal:   ociConfig.StopSignal,
	// 	User:         ociConfig.User,
	// 	WorkingDir:   ociConfig.WorkingDir,
	// 	Env:          ociConfig.Env,
	// 	Entrypoint:   ociConfig.Entrypoint,
	// 	ExposedPorts: ociConfig.ExposedPorts,
	// 	Volumes:      ociConfig.Volumes,
	// 	Labels:       ociConfig.Labels,
	// }
}
