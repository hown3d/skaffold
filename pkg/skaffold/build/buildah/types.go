package buildah

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/buildah"
)

// Builder is an artifact builder that uses buildah
type Builder struct {
	client     buildah.Buildah
	concurreny int
	pushImages bool
}

func NewBuilder(concurrency int, pushImages bool) (*Builder, error) {
	client, err := buildah.NewBuildah()
	if err != nil {
		return nil, err
	}
	return &Builder{
		client:     client,
		concurreny: concurrency,
		pushImages: pushImages,
	}, nil
}

const (
	gzipCompression  = "gzip"
	bzip2Compression = "bzip2"
	zstdCompression  = "zstd"
	xzCompression    = "xz"
	uncompressed     = "uncompressed"
)
