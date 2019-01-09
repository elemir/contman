# contman ![CircleCI](https://img.shields.io/circleci/project/github/elemir/contman.svg) ![license](https://img.shields.io/github/license/elemir/contman.svg)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Felemir%2Fcontman.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Felemir%2Fcontman?ref=badge_shield)

Library for high-level control of container system, running commands and prepared receipts. It provides three main abstractions: Manager, Container and Receipt. Manager allow container creation and using images, Container has all basic actions of using specific container. Main and most interesting this is Receipt

## Receipt
Receipt is a declarative description of running specific container and copying data from/to it. It useful to make some actions in isolated or remote environment. 
```.go
type Receipt struct {
	Image            string
	Cmd              string
	Env              map[string]string
	InputCopy        map[string]string
	OutputCopy       map[string]string
	UseControlSocket bool
	OnlyCreate       bool
}

```

Basic usage of receipts looks like this:
```.go
package main

import (
	"log"

	"github.com/elemir/contman"
	"github.com/elemir/contman/docker"
)

var receipt = contman.Receipt{
	Image:      "alpine:latest",
	Cmd:        "sed \"s/README.md/$MD/g\" -i /README.md",
	InputCopy:  map[string]string{"README.md": "/"},
	OutputCopy: map[string]string{"/README.md": "."},
	Env:        map[string]string{"MD": "WRITEYOU.md"},
}

func main() {
	dm, err := docker.NewDockerManager()
	if err != nil {
		log.Println("Cannot create docker manager: ", err)
	}
	err = contman.RunReceipt(dm, receipt)
	if err != nil {
		log.Println("Cannot run receipt: ", err)
	}
}
```

This code will change all 'README.md' entrances in README.md file to 'WRITEYOU.md'



## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Felemir%2Fcontman.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Felemir%2Fcontman?ref=badge_large)