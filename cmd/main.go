package main

import (
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	hub "github.com/konveyor/tackle2-hub/addon"
	"os"
	pathlib "path"
	"strings"
)

var (
	addon = hub.Addon
)

type SoftError = hub.SoftError

func main() {
	addon.Run(func() (err error) {
		variant := addon.Variant()
		addon.Activity("Variant: %s", variant)
		switch variant {
		case "mount:report":
			err = mountReport()
		case "mount:clean":
			err = mountClean()
		default:
			err = &SoftError{Reason: "Variant not supported."}
		}
		return
	})
}

//
// mountReport reports mount statistics.
func mountReport() (err error) {
	d := &MountInput{}
	err = addon.DataWith(d)
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	if len(d.Mount) == 0 {
		err = &SoftError{Reason: "Path required."}
		return
	}
	cmd := command.Command{Path: "/usr/bin/df"}
	cmd.Options.Add("-h")
	cmd.Options.Addf(d.path())
	err = cmd.Run()
	if err != nil {
		return
	}
	result := MountReport{}
	output := string(cmd.Output)
	output = strings.Split(output, "\n")[1]
	part := strings.Fields(output)
	result.Capacity = part[1]
	result.Used = part[2]
	addon.Result(result)
	return
}

//
// mountClean deletes the content of the mount.
func mountClean() (err error) {
	d := &MountInput{}
	err = addon.DataWith(d)
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	if len(d.Mount) == 0 {
		err = &SoftError{Reason: "Mount name required."}
		return
	}
	content, err := os.ReadDir(d.path())
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	for _, entry := range content {
		p := pathlib.Join(d.path(), entry.Name())
		err = nas.RmDir(p)
		if err != nil {
			err = &SoftError{Reason: err.Error()}
			return
		}
	}
	err = mountReport()
	return
}

//
// MountInput data.
type MountInput struct {
	Mount string `json:"path"`
}

//
// Mount path.
func (r *MountInput) path() string {
	return "/mnt/" + r.Mount
}

//
// MountReport The df variant result.
type MountReport struct {
	Capacity string `json:"capacity,omitempty"`
	Used     string `json:"used"`
}
