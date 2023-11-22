/*
Package command provides support for addons to
executing (CLI) commands.
*/
package command

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	hub "github.com/konveyor/tackle2-hub/addon"
)

var (
	addon = hub.Addon
)

//
// Command execution.
type Command struct {
	Options Options
	Path    string
	Dir     string
	Output  []byte
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
	addon.Activity(
		"[CMD] Running: %s %s",
		r.Path,
		strings.Join(r.Options, " "))
	cmd := exec.CommandContext(ctx, r.Path, r.Options...)
	cmd.Dir = r.Dir
	r.Output, err = cmd.CombinedOutput()
	if err != nil {
		addon.Activity(
			"[CMD] %s failed: %s.\n%s",
			r.Path,
			err.Error(),
			string(r.Output))
	} else {
		addon.Activity("[CMD] succeeded.")
	}
	return
}

//
// RunSilent executes the command.
// On error: The command (without arguments) and output are
// reported in task Report.Activity
func (r *Command) RunSilent() (err error) {
	err = r.RunSilentWith(context.TODO())
	return
}

//
// RunSilentWith executes the command with context.
// On error: The command (without arguments) and output are
// reported in task Report.Activity
func (r *Command) RunSilentWith(ctx context.Context) (err error) {
	cmd := exec.CommandContext(ctx, r.Path, r.Options...)
	cmd.Dir = r.Dir
	r.Output, err = cmd.CombinedOutput()
	if err != nil {
		addon.Activity(
			"[CMD] %s failed: %s.\n%s",
			r.Path,
			err.Error(),
			string(r.Output))
	}
	return
}

//
// Options are CLI options.
type Options []string

//
// add
func (a *Options) Add(option string, s ...string) {
	*a = append(*a, option)
	*a = append(*a, s...)
}

//
// add
func (a *Options) Addf(option string, x ...interface{}) {
	*a = append(*a, fmt.Sprintf(option, x...))
}
