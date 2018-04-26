package docker

import (
	"testing"

	"github.com/elemir/contman"
)

var alpineReceipt = contman.Receipt{
	Image: "alpine:latest",
	Cmd:   "echo Hello World!",
}

func TestRun(t *testing.T) {
	dm, err := NewDockerManager()
	if err != nil {
		t.Error("Cannot create docker manager: ", err)
	}
	err = contman.RunReceipt(dm, alpineReceipt)
	if err != nil {
		t.Error("Cannot run receipt: ", err)
	}
}
