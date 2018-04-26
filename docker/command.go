package docker

import (
	"errors"
	"os"

	"github.com/docker/docker/api/types"

	dockercli "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/container"
	cliflags "github.com/docker/cli/cli/flags"
)

func (d *DockerManager) RunCommand(containerName string, command []string) error {
	resp, err := d.client.ContainerExecCreate(d.context, containerName, types.ExecConfig{
		User:         "root",
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	})
	if err != nil {
		return err
	}

	execID := resp.ID
	if execID == "" {
		return errors.New("exec ID empty")
	}

	respAttach, errAttach := d.client.ContainerExecAttach(d.context, execID, types.ExecStartCheck{
		Tty: true,
	})
	if errAttach != nil {
		return errAttach
	}
	defer respAttach.Close()

	errCh := make(chan error, 1)

	dockerCli := dockercli.NewDockerCli(os.Stdin, os.Stdout, os.Stderr)
	dockerCli.Initialize(cliflags.NewClientOptions())

	go func() {
		defer close(errCh)
		errCh <- func() error {
			streamer := hijackedIOStreamer{
				streams:      dockerCli,
				inputStream:  dockerCli.In(),
				outputStream: dockerCli.Out(),
				errorStream:  dockerCli.Out(),
				resp:         respAttach,
				tty:          true,
			}

			return streamer.stream(d.context)
		}()
	}()

	if dockerCli.In().IsTerminal() {
		if err := container.MonitorTtySize(d.context, dockerCli, execID, true); err != nil {
			return err
		}
	}

	if err := <-errCh; err != nil {
		return err
	}

	respExit, err := d.client.ContainerExecInspect(d.context, execID)
	if err != nil {
		return err
	}

	status := respExit.ExitCode
	if status != 0 {
		return errors.New("command failed")
	}

	return nil
}
