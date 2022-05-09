package main

import (
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	hub "github.com/konveyor/tackle2-hub/addon"
	"github.com/konveyor/tackle2-hub/api"
	"os"
	pathlib "path"
	"strings"
)

var (
	addon = hub.Addon
)

const (
	MountRoot = "/mnt/"
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
	d := &Data{}
	err = addon.DataWith(d)
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	if len(d.Volumes) == 0 {
		return
	}
	for _, id := range d.Volumes {
		var v *api.Volume
		v, err = addon.Volume.Get(id)
		if err != nil {
			return
		}
		path := MountRoot + v.Name
		cmd := command.Command{Path: "/usr/bin/df"}
		cmd.Options.Add("-h")
		cmd.Options.Addf(path)
		err = cmd.Run()
		if err != nil {
			return
		}
		output := string(cmd.Output)
		output = strings.Split(output, "\n")[1]
		part := strings.Fields(output)
		v.Capacity = part[1]
		v.Used = part[2]
		err = addon.Volume.Update(v)
		if err != nil {
			return
		}
		addon.Activity("Volume (id=%d) %s updated.", v.ID, v.Name)
	}
	return
}

//
// mountClean deletes the content of the mount.
// Then triggers a mountReport to update the volume.
func mountClean() (err error) {
	d := &Data{}
	err = addon.DataWith(d)
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	if len(d.Volumes) == 0 {
		return
	}
	var entries []os.DirEntry
	for _, id := range d.Volumes {
		var v *api.Volume
		v, err = addon.Volume.Get(id)
		if err != nil {
			return
		}
		path := MountRoot + v.Name
		entries, err = os.ReadDir(path)
		if err != nil {
			err = &SoftError{
				Reason: err.Error(),
			}
			return
		}
		for _, entry := range entries {
			p := pathlib.Join(path, entry.Name())
			err = nas.RmDir(p)
			if err != nil {
				err = &SoftError{
					Reason: err.Error(),
				}
				return
			}
		}
		addon.Activity("Volume (id=%d) %s cleaned.", v.ID, v.Name)
	}
	err = mountReport()
	return
}

//
// Data input.
type Data struct {
	Volumes []uint `json:"volumes,omitempty"`
}
