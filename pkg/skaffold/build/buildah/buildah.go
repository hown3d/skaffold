package buildah

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/unshare"
)

func (b *Builder) Build(ctx context.Context, out io.Writer, a *latestV1.Artifact, tag string, platforms platform.Matcher) (string, error) {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "buildah",
		"Context":     instrumentation.PII(a.Workspace),
		"Destination": instrumentation.PII(tag),
	})

	dockerfile, err := getDockerfilePath(a.Workspace, a.BuildahArtifact.DockerfilePath)
	if err != nil {
		return "", containerfileNotFound(err, a.ImageName)
	}

	format, err := getFormat(a.BuildahArtifact.Format)
	if err != nil {
		return "", fmt.Errorf("buildah format: %w", err)
	}

	compression, err := getCompression(a.BuildahArtifact.Compression)
	if err != nil {
		return "", fmt.Errorf("buildah compression: %w", err)
	}

	absContextDir, err := filepath.Abs(a.Workspace)
	if err != nil {
		return "", fmt.Errorf("getting absolute path of context for image %v: %w", a.ImageName, err)
	}

	buildahPlatforms := []struct {
		OS      string
		Arch    string
		Variant string
	}{}
	for _, platform := range platforms.Platforms {
		buildahPlatforms = append(buildahPlatforms, struct {
			OS      string
			Arch    string
			Variant string
		}{
			OS:      platform.OS,
			Arch:    platform.Architecture,
			Variant: platform.Variant,
		})
	}

	buildOptions := define.BuildOptions{
		ContextDirectory: absContextDir,
		NoCache:          a.BuildahArtifact.NoCache,
		Args:             a.BuildahArtifact.BuildArgs,
		Target:           a.BuildahArtifact.Target,
		AdditionalTags:   []string{tag},
		Err:              out,
		Out:              out,
		ReportWriter:     out,
		Squash:           a.BuildahArtifact.Squash,
		Output:           a.ImageName,
		Compression:      compression,
		OutputFormat:     format,
		// TODO: maybe switch to IsolationChRoot, not sure
		// running in rootless mode, so isolate chroot
		// otherwise buildah will try to create devices,
		// which I am not allowed to do in a rootless environment.
		Isolation: buildah.IsolationDefault,
		CommonBuildOpts: &define.CommonBuildOptions{
			Secrets: a.BuildahArtifact.Secrets,
			AddHost: a.BuildahArtifact.AddHost,
		},
		AllPlatforms: platforms.All,
		Platforms:    buildahPlatforms,
		Jobs:         b.concurrency,
	}

	id, ref, err := imagebuildah.BuildDockerfiles(ctx, b.buildStore, buildOptions, dockerfile)
	if err != nil {
		return "", fmt.Errorf("building image %v: %w", a.ImageName, err)
	}

	log.Entry(ctx).Trace(fmt.Sprintf("built image %v with id %v", a.ImageName, id))

	if b.pushImages {
		ref, err = pushImage(ctx, id, a.ImageName, out)
		if err != nil {
			return "", fmt.Errorf("pushing image %v: %w", a.ImageName, err)
		}
	}

	log.Entry(ctx).Trace(fmt.Sprintf("id for image %v: %v", a.ImageName, id))
	return ref.Name(), nil
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }

func newBuildStore() (storage.Store, error) {
	buildStoreOptions, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
	if err != nil {
		return nil, fmt.Errorf("buildah store options: %w", err)
	}
	return storage.GetStore(buildStoreOptions)
}

func pushImage(ctx context.Context, containerID string, imageName string, out io.Writer) (reference.Canonical, error) {
	dest, err := alltransports.ParseImageName(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing image name: %w", err)
	}
	pushOpts := buildah.PushOptions{
		ReportWriter: out,
	}
	ref, _, err := buildah.Push(ctx, containerID, dest, pushOpts)
	if err != nil {
		return nil, fmt.Errorf("buildah push: %w", err)
	}
	return ref, nil
}

// getDockerfilePath will get the absolute path to the specified containerfile or defaults to "Dockerfile" if path is empty.
func getDockerfilePath(contextDir string, containerfilePath string) (string, error) {
	if containerfilePath != "" {
		// Fail fast if the Containerfile can't be found.
		containerfile, err := docker.NormalizeDockerfilePath(contextDir, containerfilePath)
		if err != nil {
			return "", fmt.Errorf("normalizing dockerfile path for file %v: %w", containerfilePath, err)
		}
		return containerfile, nil
	}
	containerfile, err := docker.NormalizeDockerfilePath(contextDir, "Dockerfile")
	if err != nil {
		return "", fmt.Errorf("normalizing dockerfile path: %w", err)
	}
	return containerfile, nil
}

func getCompression(compression string) (archive.Compression, error) {
	switch compression {
	case xzCompression:
		return archive.Xz, nil
	case zstdCompression:
		return archive.Zstd, nil
	case gzipCompression:
		return archive.Gzip, nil
	case bzip2Compression:
		return archive.Bzip2, nil
	case uncompressed:
		return archive.Uncompressed, nil
	case "":
		return archive.Gzip, nil
	default:
		return -1, fmt.Errorf("unknown compression algorithm: %q", compression)
	}
}

func getFormat(format string) (string, error) {
	switch format {
	case define.OCI:
		return define.OCIv1ImageManifest, nil
	case define.DOCKER:
		return define.Dockerv2ImageManifest, nil
	case "":
		return define.OCIv1ImageManifest, nil
	default:
		return "", fmt.Errorf("unrecognized image type %q", format)
	}
}
