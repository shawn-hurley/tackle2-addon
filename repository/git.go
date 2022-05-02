package repository

import (
	"errors"
	"fmt"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	"github.com/konveyor/tackle2-addon/ssh"
	"github.com/konveyor/tackle2-hub/api"
	urllib "net/url"
	"os"
	pathlib "path"
)

//
// Git repository.
type Git struct {
	SCM
}

//
// Validate settings.
func (r *Git) Validate() (err error) {
	u, err := urllib.Parse(r.Application.Repository.URL)
	if err != nil {
		err = &SoftError{Reason: err.Error()}
		return
	}
	insecure, err := addon.Setting.Bool("git.insecure.enabled")
	if err != nil {
		return
	}
	switch u.Scheme {
	case "http":
		if !insecure {
			err = &SoftError{
				Reason: "http URL used with git.insecure.enabled = FALSE",
			}
			return
		}
	}
	return
}

//
// Fetch clones the repository.
func (r *Git) Fetch() (err error) {
	url := r.URL()
	addon.Activity("[GIT] Cloning: %s", url.String())
	_ = nas.RmDir(r.Path)
	id, found, err := addon.Application.FindIdentity(r.Application.ID, "source")
	if err != nil {
		return
	}
	if found {
		addon.Activity(
			"[GIT] Using credentials (id=%d) %s.",
			id.ID,
			id.Name)
	} else {
		id = &api.Identity{}
	}
	err = r.writeConfig()
	if err != nil {
		return
	}
	err = r.writeCreds(id)
	if err != nil {
		return
	}
	agent := ssh.Agent{}
	err = agent.Add(id)
	if err != nil {
		return
	}
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Options.Add("clone", url.String(), r.Path)
	err = cmd.Run()
	if err != nil {
		return
	}
	err = r.checkout()
	return
}

//
// URL returns the parsed URL.
func (r *Git) URL() (u *urllib.URL) {
	u, _ = urllib.Parse(r.Application.Repository.URL)
	return
}

//
// writeConfig writes config file.
func (r *Git) writeConfig() (err error) {
	path := pathlib.Join(HomeDir, ".gitconfig")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = liberr.Wrap(os.ErrExist)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	insecure, err := addon.Setting.Bool("git.insecure.enabled")
	if err != nil {
		return
	}
	proxy, err := r.proxy()
	if err != nil {
		return
	}
	s := "[credential]\n"
	s += "helper = store\n"
	s += "[http]\n"
	s += fmt.Sprintf("sslVerify = %t\n", !insecure)
	if proxy != "" {
		s += fmt.Sprintf("proxy = %s\n", proxy)
	}
	_, err = f.Write([]byte(s))
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	_ = f.Close()
	return
}

//
// writeCreds writes credentials (store) file.
func (r *Git) writeCreds(id *api.Identity) (err error) {
	if id.User == "" || id.Password == "" {
		return
	}
	path := pathlib.Join(HomeDir, ".git-credentials")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = liberr.Wrap(os.ErrExist)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
		return
	}
	url := r.URL()
	for _, scheme := range []string{
		url.Scheme,
		"https",
		"http",
	} {
		entry := scheme
		entry += "://"
		if id.User != "" {
			entry += id.User
			entry += ":"
		}
		if id.Password != "" {
			entry += id.Password
			entry += "@"
		}
		entry += url.Host
		_, err = f.Write([]byte(entry + "\n"))
		if err != nil {
			err = liberr.Wrap(
				err,
				"path",
				path)
			break
		}
	}
	_ = f.Close()
	return
}

//
// proxy builds the proxy.
func (r *Git) proxy() (proxy string, err error) {
	kind := ""
	url := r.URL()
	switch url.Scheme {
	case "http":
		kind = "http"
	case "https",
		"git@github.com":
		kind = "https"
	default:
		return
	}
	p, err := addon.Proxy.Find(kind)
	if err != nil || p == nil || !p.Enabled {
		return
	}
	for _, h := range p.Excluded {
		if h == url.Host {
			return
		}
	}
	addon.Activity(
		"[GIT] Using proxy (%d) %s.",
		p.ID,
		p.Kind)
	auth := ""
	if p.Identity != nil {
		var id *api.Identity
		id, err = addon.Identity.Get(p.Identity.ID)
		if err != nil {
			return
		}
		auth = fmt.Sprintf(
			"%s:%s@",
			id.User,
			id.Password)
	}
	proxy = fmt.Sprintf(
		"%s://%s%s",
		p.Kind,
		auth,
		p.Host)
	if p.Port > 0 {
		proxy = fmt.Sprintf(
			"%s:%d",
			proxy,
			p.Port)
	}
	return
}

//
// checkout ref.
func (r *Git) checkout() (err error) {
	branch := r.Application.Repository.Branch
	if branch == "" {
		return
	}
	dir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(dir)
	}()
	_ = os.Chdir(r.Path)
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Options.Add("checkout", branch)
	err = cmd.Run()
	return
}
