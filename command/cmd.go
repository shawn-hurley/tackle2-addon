/*
Package command provides support for addons to
executing (CLI) commands.
*/
package command

import (
	"context"
	"fmt"
	"os/exec"

	hub "github.com/konveyor/tackle2-hub/addon"
	"path"
)

var (
	addon = hub.Addon
)

//
// New returns a command.
func New(path string) (cmd *Command) {
	cmd = &Command{Path: path}
	return
}

//
// Command execution.
type Command struct {
	Options  Options
	Path     string
	Dir      string
	Reporter Reporter
	Writer   Writer
}

//
// Run executes the command.
// The command and output are both reported in
// task Report.Activity.
func (r *Command) Run() (err error) {
	err = r.RunWith(context.TODO())
	return
}

//
// RunWith executes the command with context.
// The command and output are both reported in
// task Report.Activity.
func (r *Command) RunWith(ctx context.Context) (err error) {
	r.Writer.reporter = &r.Reporter
	output := path.Base(r.Path) + ".output"
	r.Reporter.file, err = addon.File.Touch(output)
	if err != nil {
		return
	}
	r.Reporter.Run(r.Path, r.Options)
	addon.Attach(r.Reporter.file)
	defer func() {
		r.Writer.End()
		if err != nil {
			r.Reporter.Error(r.Path, err, r.Writer.buffer)
		} else {
			r.Reporter.Succeeded(r.Path, r.Writer.buffer)
		}
	}()
	cmd := exec.CommandContext(ctx, r.Path, r.Options...)
	cmd.Dir = r.Dir
	cmd.Stdout = &r.Writer
	cmd.Stderr = &r.Writer
	err = cmd.Start()
	if err != nil {
		return
	}
	err = cmd.Wait()
	return
}

//
// RunSilent executes the command.
// On error: The command (without arguments) and output are
// reported in task Report.Activity
func (r *Command) RunSilent() (err error) {
	r.Reporter.Verbosity = Error
	err = r.RunWith(context.TODO())
	return
}

//
// Output returns the command output.
func (r *Command) Output() (b []byte) {
	return r.Writer.buffer
}

//
// Options are CLI options.
type Options []string

//
// Add option.
func (a *Options) Add(option string, s ...string) {
	*a = append(*a, option)
	*a = append(*a, s...)
}

//
// Addf option.
func (a *Options) Addf(option string, x ...interface{}) {
	*a = append(*a, fmt.Sprintf(option, x...))
}
