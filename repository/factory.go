package repository

import (
	"os"

	hub "github.com/konveyor/tackle2-hub/addon"
	"github.com/konveyor/tackle2-hub/api"
)

var (
	addon   = hub.Addon
	HomeDir = ""
)

func init() {
	HomeDir, _ = os.UserHomeDir()
}

// New SCM repository factory.
func New(destDir string, remote *api.Repository, identities []api.Ref) (r SCM, err error) {
	var insecure bool
	switch remote.Kind {
	case "subversion":
		insecure, err = addon.Setting.Bool("svn.insecure.enabled")
		if err != nil {
			return
		}
		r = &Subversion{
			Path: destDir,
			Remote: Remote{
				Repository: remote,
				Identities: identities,
				Insecure:   insecure,
			},
		}
	default:
		insecure, err = addon.Setting.Bool("git.insecure.enabled")
		if err != nil {
			return
		}
		r = &Git{
			Path: destDir,
			Remote: Remote{
				Repository: remote,
				Identities: identities,
				Insecure:   insecure,
			},
		}
	}
	err = r.Validate()
	return
}

// SCM interface.
type SCM interface {
	Validate() (err error)
	Fetch() (err error)
	Branch(name string) (err error)
	Commit(files []string, msg string) (err error)
}

// Remote repository.
type Remote struct {
	*api.Repository
	Identities []api.Ref
	Insecure   bool
}

// FindIdentity by kind.
func (r *Remote) findIdentity(kind string) (matched *api.Identity, found bool, err error) {
	for _, ref := range r.Identities {
		identity, nErr := addon.Identity.Get(ref.ID)
		if nErr != nil {
			err = nErr
			return
		}
		if identity.Kind == kind {
			found = true
			matched = identity
			break
		}
	}
	return
}
