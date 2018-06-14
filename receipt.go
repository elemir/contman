package contman

import (
	"errors"
	"os"
	"time"
)

type Receipt struct {
	Image              string
	Cmd                string
	Env                map[string]string
	InputCopy          map[string]string
	OutputCopy         map[string]string
	Timeout            time.Duration
	UseControlSocket   bool
	UseLocalImage      bool
	OnlyCreate         bool
	UseImageWorkingDir bool
}

func RunReceipt(cm Manager, receipt Receipt) error {
	if !receipt.UseLocalImage {
		err := cm.PullImage(receipt.Image)
		if err != nil {
			return err
		}
	}

	mounts := []Mount{}

	if receipt.UseControlSocket {
		mounts = cm.GetSystemMounts()
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	config := Config{
		Image:  receipt.Image,
		Cmd:    receipt.Cmd,
		Env:    receipt.Env,
		Mounts: mounts,
	}

	if !receipt.UseImageWorkingDir {
		config.WorkingDir = wd
	}

	cntr, err := cm.ContainerCreate(config)

	if err != nil {
		return err
	}

	defer func() {
		isRunning, _ := cntr.IsRunning()
		if isRunning {
			cntr.Stop(receipt.Timeout)
		}
		cntr.Remove()
	}()

	if !receipt.OnlyCreate {
		if err := startReceiptContainer(cntr, receipt); err != nil {
			return err
		}
	}

	for src, dest := range receipt.OutputCopy {
		cntr.CopyFrom(src, dest)
	}

	return nil
}

func startReceiptContainer(cntr Container, receipt Receipt) error {
	for src, dest := range receipt.InputCopy {
		if _, err := os.Stat(src); err != nil {
			continue
		}
		cntr.CopyTo(src, dest)
	}

	if err := cntr.Start(); err != nil {
		return err
	}

	exitCode, err := cntr.Wait(true)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		cntr.GetLogger().Errorf("Container exited with non-zero code: %d", exitCode)
		return errors.New("failed to run receipt")
	}

	return nil
}
