package contman

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

type Config struct {
	Image      string
	Cmd        string
	Env        map[string]string
	Mounts     []Mount
	WorkingDir string
}

type Container interface {
	Start() error
	Stop(timeout time.Duration) error
	Remove() error

	IsRunning() (bool, error)
	Wait(dumpLog bool) (int, error)

	CopyFrom(src, dest string) error
	CopyTo(src, dest string) error

	GetLogger() *log.Entry
}

type Manager interface {
	PullImage(string) error
	HasImage(string) bool

	ContainerCreate(Config) (Container, error)
	GetSystemMounts() []Mount
}
