package main

import (
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/repository"
	"github.com/konveyor/tackle2-addon/ssh"
)

func main() {
	_ = command.New("")
	_ = ssh.Agent{}
	_, _ = repository.New("", nil, nil)
}
