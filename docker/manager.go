package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"github.com/elemir/contman"
)

type DockerManager struct {
	client  *client.Client
	context context.Context
}

func NewDockerManagerWithContext(ctx context.Context) (*DockerManager, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	cli.NegotiateAPIVersion(ctx)

	dm := &DockerManager{
		client:  cli,
		context: ctx,
	}

	return dm, nil
}

func NewDockerManager() (*DockerManager, error) {
	ctx := context.Background()
	dm, err := NewDockerManagerWithContext(ctx)

	return dm, err
}

func (dm *DockerManager) PullImage(image string) error {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		log.WithError(err).Error("Cannot parse image name")
		return err
	}
	authConfig := getAuthConfig(reference.Domain(named))
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.WithError(err).Error("Error encoding auth config")
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	out, err := dm.client.ImagePull(dm.context, image, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		log.WithError(err).Error("Error pulling image")
		return err
	}
	defer func() { _ = out.Close() }()
	_, _ = io.Copy(os.Stdout, out)
	return nil
}

func (dm *DockerManager) HasImage(image string) bool {
	if image == "" {
		return false
	}
	if !strings.Contains(image, ":") {
		image += ":latest"
	}

	images, err := dm.client.ImageList(dm.context, types.ImageListOptions{})
	if err != nil {
		log.WithError(err).Error("Unable to list images")
		return false
	}

	hasImage := false

OuterLoop:
	for _, imageInfo := range images {
		for _, tag := range imageInfo.RepoTags {
			if tag == image {
				hasImage = true
				break OuterLoop
			}
		}
	}

	return hasImage
}

func (dm *DockerManager) ContainerCreate(config contman.Config) (contman.Container, error) {
	mounts := make([]mount.Mount, len(config.Mounts))
	for i, m := range config.Mounts {
		mounts[i] = mount.Mount{
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
			Type:     mount.TypeBind,
		}
	}

	env := make([]string, len(config.Env))
	i := 0
	for key, value := range config.Env {
		env[i] = fmt.Sprintf("%s=%s", key, value)
		i++
	}

	containerConfig := &container.Config{
		Image:      config.Image,
		Entrypoint: []string{"sh"},
		Cmd: []string{
			"-c",
			config.Cmd,
		},
		WorkingDir: config.WorkingDir,
		Env:        env,
	}

	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		NetworkMode: "host",
	}

	resp, err := dm.client.ContainerCreate(dm.context, containerConfig, hostConfig, nil, "")

	if err != nil {
		log.WithError(err).Error("Error creating container")
		return nil, err
	}

	return &DockerContainer{
		manager: dm,
		id:      resp.ID,
	}, nil
}

func (dm *DockerManager) GetSystemMounts() []contman.Mount {
	return []contman.Mount{
		{
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		},
		{
			Source:   "/root/.docker",
			Target:   "/root/.docker",
			ReadOnly: true,
		},
	}
}

func getAuthConfig(registry string) *types.AuthConfig {
	authConfigurations, err := docker.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		return &types.AuthConfig{}
	}

	authConfiguration, ok := authConfigurations.Configs[registry]
	if !ok {
		return &types.AuthConfig{}
	}

	return &types.AuthConfig{
		Username: authConfiguration.Username,
		Password: authConfiguration.Password,
	}
}
