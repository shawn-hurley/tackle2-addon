package ssh

import (
	"errors"
	"fmt"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	hub "github.com/konveyor/tackle2-hub/addon"
	"github.com/konveyor/tackle2-hub/api"
	"os"
	pathlib "path"
)

var (
	addon   = hub.Addon
	HomeDir = ""
	SSHDir  = ""
)

func init() {
	HomeDir, _ = os.UserHomeDir()
	SSHDir = pathlib.Join(
		HomeDir,
		".ssh")

}

//
// Agent agent.
type Agent struct {
}

//
// Start the ssh-agent.
func (r *Agent) Start() (err error) {
	pid := os.Getpid()
	socket := fmt.Sprintf("/tmp/agent.%d", pid)
	cmd := command.Command{Path: "/usr/bin/ssh-agent"}
	cmd.Options.Add("-a", socket)
	err = cmd.Run()
	if err != nil {
		_ = os.Setenv("SSH_AUTH_SOCK", socket)
	}
	err = nas.MkDir(SSHDir, 0700)
	if err != nil {
		return
	}

	addon.Activity("[SSH] Agent started.")

	return
}

//
// Add ssh key.
func (r *Agent) Add(id *api.Identity) (err error) {
	if id.Key == "" {
		return
	}
	addon.Activity("[SSH] Adding key: %s", id.Name)
	keyPath := pathlib.Join(
		SSHDir,
		"id_"+id.Name)
	_, err = os.Stat(keyPath)
	if !errors.Is(err, os.ErrNotExist) {
		err = liberr.Wrap(os.ErrExist)
		return
	}
	f, err := os.Create(keyPath)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			keyPath)
		return
	}
	_, err = f.Write([]byte(id.Key))
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			keyPath)
	}
	_ = f.Close()
	if id.Password == "" {
		return
	}
	err = r.writeAskScript(id)
	if err != nil {
		return
	}
	cmd := command.Command{Path: "/usr/bin/ssh-add"}
	cmd.Options.Add(keyPath)
	err = cmd.Run()
	return
}

//
// writeAskScript writes script that returns the key password.
func (r *Agent) writeAskScript(id *api.Identity) (err error) {
	path := "/tmp/ask.sh"
	f, err := os.Create(path)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	script := fmt.Sprintf(
		"#!/bin/sh\necho %s",
		id.Password)
	_, err = f.Write([]byte(script))
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	_ = os.Setenv("SSH_ASKPASS", path)
	_ = f.Close()
	return
}
