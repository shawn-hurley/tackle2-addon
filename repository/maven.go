package repository

import (
	"os"
	pathlib "path"

	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-hub/nas"

	"github.com/clbanning/mxj"
	liberr "github.com/jortel/go-utils/error"
)

const emptySettings = ` 
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0" 
                 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
                 xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0
                 http://maven.apache.org/xsd/settings-1.2.0.xsd">
</settings>
`

// Maven repository.
type Maven struct {
	Remote
	BinDir string
	M2Dir  string
}

// CreateSettingsFile creates the maven settings.xml file
// Will exit with path and nil error if file exists already
func (r *Maven) CreateSettingsFile() (path string, err error) {
	dir, _ := os.Getwd()
	path = pathlib.Join(dir, "settings.xml")
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
	defer f.Close()
	id, found, err := r.findIdentity("maven")
	if err != nil {
		return
	}
	var settingsXML mxj.Map
	if found {
		addon.Activity(
			"[MVN] Using credentials (id=%d) %s.",
			id.ID,
			id.Name)
		settingsXML, err = mxj.NewMapXml([]byte(id.Settings))
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	} else {
		settingsXML, err = mxj.NewMapXml([]byte(emptySettings))
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}
	err = r.injectProxy(settingsXML)
	if err != nil {
		return
	}
	err = r.injectCacheDir(settingsXML)
	if err != nil {
		return
	}
	settings, err := settingsXML.XmlIndent("", "  ")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	_, err = f.Write(settings)
	if err != nil {
		err = liberr.Wrap(
			err,
			"path",
			path)
	}
	addon.Activity("[FILE] Created %s.", path)
	return
}

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

// FetchArtifact fetches an application artifact.
func (r *Maven) FetchArtifact(artifact string) (err error) {
	addon.Activity("[MVN] Fetch artifact %s.", artifact)
	options := command.Options{
		"dependency:copy",
	}
	options.Addf("-Dartifact=%s", artifact)
	options.Add("-Dmdep.useBaseVersion=true")
	err = r.run(options)
	return
}

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

// run executes maven.
func (r *Maven) run(options command.Options) (err error) {
	settings, err := r.CreateSettingsFile()
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
	cmd := command.New("/usr/bin/svn")
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

func (r *Maven) injectCacheDir(settings mxj.Map) (err error) {
	err = settings.SetValueForPath(r.M2Dir, "settings.localRepository")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	return
}

// injectProxy injects proxy settings.
func (r *Maven) injectProxy(settingsXML mxj.Map) (err error) {
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
	v, err := settingsXML.ValuesForPath("settings.proxies.proxy")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	err = settingsXML.SetValueForPath(
		mxj.Map{"proxy": append(v, pList...)},
		"settings.proxies")
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	return
}
