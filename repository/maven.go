package repository

import (
	"github.com/clbanning/mxj"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/nas"
	"github.com/konveyor/tackle2-hub/api"
	"os"
	pathlib "path"
)

//
// Maven repository.
type Maven struct {
	Application *api.Application
	BinDir      string
	M2Dir       string
}

//
// Fetch fetches dependencies listed in the POM.
func (r *Maven) Fetch(sourceDir string) (err error) {
	addon.Activity("[MVN] Fetch dependencies.")
	pom := pathlib.Join(sourceDir, "pom.xml")
	options := command.Options{
		"dependency:copy-dependencies",
		"-f",
		pom,
	}
	err = r.run(options)
	return
}

//
// FetchArtifact fetches an application artifact.
func (r *Maven) FetchArtifact() (err error) {
	artifact := r.Application.Binary
	addon.Activity("[MVN] Fetch artifact %s.", artifact)
	options := command.Options{
		"dependency:copy",
	}
	options.Addf("-Dartifact=%s", artifact)
	options.Add("-Dmdep.useBaseVersion=true")
	err = r.run(options)
	return
}

//
// InstallArtifacts installs application artifacts.
func (r *Maven) InstallArtifacts(sourceDir string) (err error) {
	addon.Activity("[MVN] Install application.")
	pom := pathlib.Join(sourceDir, "pom.xml")
	options := command.Options{
		"install",
		"-DskipTests",
		"-f",
		pom,
	}
	err = r.run(options)
	return
}

//
// DeleteArtifacts deletes application artifacts.
func (r *Maven) DeleteArtifacts(sourceDir string) (err error) {
	addon.Activity("[MVN] Delete application artifacts.")
	pom := pathlib.Join(sourceDir, "pom.xml")
	options := command.Options{
		"org.codehaus.mojo:build-helper-maven-plugin:3.3.0:remove-project-artifact",
		"-f",
		pom,
	}
	err = r.run(options)
	return
}

//
// HasModules determines if the POM specifies modules.
func (r *Maven) HasModules(sourceDir string) (found bool, err error) {
	pom := pathlib.Join(sourceDir, "pom.xml")
	f, err := os.Open(pom)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	m, err := mxj.NewMapXmlReader(f)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	v, err := m.ValuesForPath("project.modules.module")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	found = len(v) > 0
	return
}

//
// run executes maven.
func (r *Maven) run(options command.Options) (err error) {
	settings, err := r.writeSettings()
	if err != nil {
		return
	}
	insecure, err := addon.Setting.Bool("mvn.insecure.enabled")
	if err != nil {
		return
	}
	err = nas.MkDir(r.BinDir, 0755)
	if err != nil {
		return
	}
	cmd := command.Command{Path: "/usr/bin/mvn"}
	cmd.Options = options
	cmd.Options.Addf("-DoutputDirectory=%s", r.BinDir)
	cmd.Options.Addf("-Dmaven.repo.local=%s", r.M2Dir)
	if insecure {
		cmd.Options.Add("-Dmaven.wagon.http.ssl.insecure=true")
	}
	if settings != "" {
		cmd.Options.Add("-s", settings)
	}
	err = cmd.Run()
	return
}

//
// writeSettings writes settings file.
func (r *Maven) writeSettings() (path string, err error) {
	id, found, err := addon.Application.FindIdentity(r.Application.ID, "maven")
	if err != nil {
		return
	}
	if found {
		addon.Activity(
			"[MVN] Using credentials (id=%d) %s.",
			id.ID,
			id.Name)
	} else {
		return
	}
	dir, _ := os.Getwd()
	path = pathlib.Join(dir, "settings.xml")
	found, err = nas.Exists(path)
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
	settings := id.Settings
	settings, err = r.injectProxy(id)
	if err != nil {
		return
	}
	_, err = f.Write([]byte(settings))
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
// injectProxy injects proxy settings.
func (r *Maven) injectProxy(id *api.Identity) (s string, err error) {
	s = id.Settings
	m, err := mxj.NewMapXml([]byte(id.Settings))
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	proxies, err := addon.Proxy.List()
	if err != nil {
		return
	}
	pList := []interface{}{}
	for _, p := range proxies {
		if !p.Enabled {
			continue
		}
		addon.Activity(
			"[MVN] Using proxy (%d) %s.",
			p.ID,
			p.Kind)
		mp := mxj.Map{
			"id":       p.Kind,
			"active":   p.Enabled,
			"protocol": p.Kind,
			"host":     p.Host,
			"port":     p.Port,
		}
		if p.Identity != nil {
			pid, idErr := addon.Identity.Get(p.Identity.ID)
			if idErr != nil {
				err = idErr
				return
			}
			mp["username"] = pid.User
			mp["password"] = pid.Password
		}
		pList = append(pList, mp)
	}
	if len(pList) == 0 {
		return
	}
	v, err := m.ValuesForPath("settings.proxies.proxy")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	err = m.SetValueForPath(
		mxj.Map{"proxy": append(v, pList...)},
		"settings.proxies")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	b, err := m.XmlIndent("", "  ")
	s = string(b)
	return
}
