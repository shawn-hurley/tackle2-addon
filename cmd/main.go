package main

import (
	hub "github.com/konveyor/tackle2-hub/addon"
)

var (
	addon = hub.Addon
)

type SoftError = hub.SoftError

func main() {
	addon.Run(func() (err error) {
		return
	})
}
