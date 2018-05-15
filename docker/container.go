package docker

import (
	"context"
	"io"
	"os"
	"time"

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
	ctx := dc.getContext()
	err := dc.manager.client.ContainerStart(ctx, dc.id, types.ContainerStartOptions{})
	if err != nil {
		dc.GetLogger().WithError(err).Error("Error starting container")
	}
	return err
}

func (dc *DockerContainer) Stop(timeout time.Duration) error {
	ctx := dc.getContext()
	err := dc.manager.client.ContainerStop(ctx, dc.id, &timeout)
	if err != nil {
		dc.GetLogger().WithError(err).Error("Error stopping container")
	}
	return err
}

func (dc *DockerContainer) Remove() error {
	ctx := dc.getContext()
	err := dc.manager.client.ContainerRemove(ctx, dc.id, types.ContainerRemoveOptions{})
	if err != nil {
		dc.GetLogger().WithError(err).Errorf("Error removing container")
	}
	return err
}

func (dc *DockerContainer) IsRunning() (bool, error) {
	ctx := dc.getContext()
	descr, err := dc.manager.client.ContainerInspect(ctx, dc.id)
	if err != nil {
		dc.GetLogger().WithError(err).Errorf("Error checking container running status")
	}
	run := descr.ContainerJSONBase != nil && descr.State != nil && descr.State.Running
	return run, err
}

func (dc *DockerContainer) Wait(dumpLog bool) (int, error) {
	ctx := dc.getContext()
	var exitCode int

	if dumpLog {
		out, err := dc.manager.client.ContainerLogs(ctx, dc.id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
		if err != nil {
			dc.GetLogger().WithError(err).Error("Error getting container logs")
			return 0, err
		}
		defer func() { _ = out.Close() }()
		go func() { _, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, out) }()
	}

	statusCh, errCh := dc.manager.client.ContainerWait(ctx, dc.id, container.WaitConditionNotRunning)
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
	ctx := dc.getContext()
	l := dc.GetLogger().WithFields(log.Fields{
		"src":  src,
		"dest": dest,
	})
	reader, stat, err := dc.manager.client.CopyFromContainer(ctx, dc.id, src)
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
	ctx := dc.getContext()
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

	err := dc.manager.client.CopyToContainer(ctx, dc.id, dest, reader, types.CopyToContainerOptions{})
	if err != nil {
		l.WithError(err).Error("Error copying to container")
		return err
	}

	return nil
}

func (dc *DockerContainer) GetLogger() *log.Entry {
	return log.WithField("containerID", dc.id)
}

func (dc *DockerContainer) getContext() context.Context {
	select {
	case <-dc.manager.context.Done():
		return context.TODO()
	default:
		return dc.manager.context
	}
}
