package contman

import (
	"errors"
	"os"
)

type Receipt struct {
	Image            string
	Cmd              string
	Env              map[string]string
	InputCopy        map[string]string
	OutputCopy       map[string]string
	UseControlSocket bool
	UseLocalImage    bool
	OnlyCreate       bool
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

	cntr, err := cm.ContainerCreate(Config{
		Image:  receipt.Image,
		Cmd:    receipt.Cmd,
		Env:    receipt.Env,
		Mounts: mounts,
	})

	if err != nil {
		return err
	}

	defer func() {
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
