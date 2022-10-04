package ssh

import (
	"context"
	"fmt"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	hub "github.com/konveyor/tackle2-hub/addon"
	"github.com/konveyor/tackle2-hub/api"
	"os"
	pathlib "path"
	"strings"
	"time"
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
		return
	}
	_ = os.Setenv("SSH_AUTH_SOCK", socket)
	err = nas.MkDir(SSHDir, 0700)
	if err != nil {
		return
	}

	addon.Activity("[SSH] Agent started.")

	return
}

//
// Add ssh key.
func (r *Agent) Add(id *api.Identity, host string) (err error) {
	if id.Key == "" {
		return
	}
	addon.Activity("[SSH] Adding key: %s", id.Name)
	suffix := fmt.Sprintf("id_%d", id.ID)
	path := pathlib.Join(
		SSHDir,
		suffix)
	found, err := nas.Exists(path)
	if found || err != nil {
		return
	}
	f, err := os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE,
		0600)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	_, err = f.Write([]byte(r.format(id.Key)))
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	_ = f.Close()
	err = r.writeAsk(id)
	if err != nil {
		return
	}
	ctx, fn := context.WithTimeout(
		context.TODO(),
		time.Second)
	defer fn()
	cmd := command.Command{Path: "/usr/bin/ssh-add"}
	cmd.Options.Add(path)
	err = cmd.RunWith(ctx)
	if err != nil {
		return
	}
	cmd = command.Command{Path: "/usr/bin/ssh-keyscan"}
	cmd.Options.Add(host)
	err = cmd.Run()
	if err != nil {
		return
	}
	known := "/etc/ssh/ssh_known_hosts"
	f, err = os.OpenFile(
		known, os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	_, err = f.Write(cmd.Output)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	_ = f.Close()
	addon.Activity("[FILE] Created %s.", path)
	return
}

//
// Ensure key formatting.
func (r *Agent) format(in string) (out string) {
	if in != "" {
		out = strings.TrimSpace(in) + "\n"
	}
	return
}

//
// writeAsk writes script that returns the key password.
func (r *Agent) writeAsk(id *api.Identity) (err error) {
	path := "/tmp/ask.sh"
	f, err := os.OpenFile(
		path,
		os.O_RDWR|os.O_CREATE,
		0700)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	script := fmt.Sprintf(
		"#!/bin/sh\necho '%s'",
		id.Password)
	_, err = f.Write([]byte(script))
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	_ = os.Setenv("SSH_ASKPASS", path)
	_ = os.Setenv("DISPLAY", "1")
	_ = f.Close()
	return
}
