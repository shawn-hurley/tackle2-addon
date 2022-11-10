package repository

import (
	"fmt"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	"github.com/konveyor/tackle2-addon/ssh"
	"github.com/konveyor/tackle2-hub/api"
	urllib "net/url"
	"os"
	pathlib "path"
	"strings"
)

// Git repository.
type Git struct {
	SCM
}

// Validate settings.
func (r *Git) Validate() (err error) {
	u := GitURL{}
	err = u.With(r.Application.Repository.URL)
	if err != nil {
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
	err = agent.Add(id, url.Host)
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

// Branch creates a branch with the given name if not exist and switch to it
func (r *Git) Branch(name string) (err error) {
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Dir = r.Path
	cmd.Options.Add("checkout", name)
	err = cmd.Run()
	if err != nil {
		cmd = command.Command{Path: "/usr/bin/git"}
		cmd.Dir = r.Path
		cmd.Options.Add("checkout", "-b", name)
	}
	r.Application.Repository.Branch = name
	return cmd.Run()
}

// addFiles adds files to staging area
func (r *Git) addFiles(files []string) (err error) {
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Dir = r.Path
	cmd.Options.Add("add", files...)
	return cmd.Run()
}

// Commit files and push to remote
func (r *Git) Commit(files []string, msg string) (err error) {
	err = r.addFiles(files)
	if err != nil {
		return err
	}
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Dir = r.Path
	cmd.Options.Add("commit")
	cmd.Options.Add("--message", msg)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return r.push()
}

// push changes to server
func (r *Git) push() (err error) {
	cmd := command.Command{Path: "/usr/bin/git"}
	cmd.Dir = r.Path
	cmd.Options.Add("push", "--set-upstream", "origin", r.Application.Repository.Branch)
	return cmd.Run()
}

// URL returns the parsed URL.
func (r *Git) URL() (u GitURL) {
	u = GitURL{}
	_ = u.With(r.Application.Repository.URL)
	return
}

// writeConfig writes config file.
func (r *Git) writeConfig() (err error) {
	path := pathlib.Join(HomeDir, ".gitconfig")
	found, err := nas.Exists(path)
	if found || err != nil {
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
	s := "[user]\n"
	s += "name = Konveyor Dev\n"
	s += "email = konveyor-dev@googlegroups.com\n"
	s += "[credential]\n"
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
	addon.Activity("[FILE] Created %s.", path)
	return
}

// writeCreds writes credentials (store) file.
func (r *Git) writeCreds(id *api.Identity) (err error) {
	if id.User == "" || id.Password == "" {
		return
	}
	path := pathlib.Join(HomeDir, ".git-credentials")
	found, err := nas.Exists(path)
	if found || err != nil {
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
	addon.Activity("[FILE] Created %s.", path)
	return
}

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
		"http://%s%s",
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

// GitURL git clone URL.
type GitURL struct {
	Raw    string
	Scheme string
	Host   string
	Path   string
}

// With populates the URL.
func (r *GitURL) With(u string) (err error) {
	r.Raw = u
	parsed, pErr := urllib.Parse(u)
	if pErr == nil {
		r.Scheme = parsed.Scheme
		r.Host = parsed.Host
		r.Path = parsed.Path
		return
	}
	notValid := liberr.New("URL not valid.")
	part := strings.Split(u, ":")
	if len(part) != 2 {
		err = notValid
		return
	}
	r.Host = part[0]
	r.Path = part[1]
	part = strings.Split(r.Host, "@")
	if len(part) != 2 {
		err = notValid
		return
	}
	r.Host = part[1]
	return
}

// String representation.
func (r *GitURL) String() string {
	return r.Raw
}
