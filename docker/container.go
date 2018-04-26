package docker

import (
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	log "github.com/sirupsen/logrus"
)

type DockerContainer struct {
	id      string
	manager *DockerManager
}

func (dc *DockerContainer) Start() error {
	dm := dc.manager
	err := dm.client.ContainerStart(dm.context, dc.id, types.ContainerStartOptions{})
	if err != nil {
		dc.GetLogger().WithError(err).Error("Error starting container")
	}
	return err
}

func (dc *DockerContainer) Remove() error {
	dm := dc.manager
	err := dm.client.ContainerRemove(dm.context, dc.id, types.ContainerRemoveOptions{})
	if err != nil {
		dc.GetLogger().WithError(err).Errorf("Error removing container")
	}
	return err
}

func (dc *DockerContainer) Wait(dumpLog bool) (int, error) {
	var exitCode int

	dm := dc.manager
	if dumpLog {
		out, err := dm.client.ContainerLogs(dm.context, dc.id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			dc.GetLogger().WithError(err).Error("Error getting container logs")
			return 0, err
		}
		defer func() { _ = out.Close() }()
		go func() { _, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, out) }()
	}

	statusCh, errCh := dm.client.ContainerWait(dm.context, dc.id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			dc.GetLogger().WithError(err).Error("Error waiting container")
			return 0, err
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	}

	return exitCode, nil
}

func (dc *DockerContainer) CopyFrom(src, dest string) error {
	dm := dc.manager
	l := dc.GetLogger().WithFields(log.Fields{
		"src":  src,
		"dest": dest,
	})
	reader, stat, err := dm.client.CopyFromContainer(dm.context, dc.id, src)
	if err != nil {
		l.WithError(err).Error("Error copying from container")
		return err
	}
	defer reader.Close()

	l.Infof("Found in container: %v", stat)

	err = extractTarFromReader(reader, dest)
	if err != nil {
		l.WithError(err).Error("Error extracting from container")
		return err
	}

	return nil
}

func (dc *DockerContainer) CopyTo(src, dest string) error {
	dm := dc.manager
	l := log.WithFields(log.Fields{
		"src":  src,
		"dest": dest,
	})
	reader, writer := io.Pipe()
	defer reader.Close()

	go func() {
		defer writer.Close()
		err := createTarToWriter(src, writer)
		if err != nil {
			l.WithError(err).Error("Failed to create tar archive")
			return
		}

	}()

	err := dm.client.CopyToContainer(dm.context, dc.id, dest, reader, types.CopyToContainerOptions{})
	if err != nil {
		l.WithError(err).Error("Error copying to container")
		return err
	}

	return nil
}

func (dc *DockerContainer) GetLogger() *log.Entry {
	return log.WithField("containerID", dc.id)
}
