package models

import (
	"strings"
	"github.com/gogits/gogs/models/errors"
	"github.com/gogits/gogs/pkg/setting"
	"strconv"
	"path/filepath"
	"fmt"
	"github.com/gogits/gogs/pkg/process"
	"github.com/Unknwon/com"
	"github.com/gogits/git-module"
	"os"
	"time"
	"io/ioutil"
	"bytes"
	"github.com/go-xorm/xorm"
)



type C_CreateRepoOptions struct {
	Name        string
	DirectoryId int64
	Description string
	Gitignores  string
	License     string
	Readme      string
	IsPrivate   bool
	IsMirror    bool
	AutoInit    bool
}


// GetRepositoryByName returns the repository by given name under user if exists.
func GetRepositoryByNameAndDirectoryId(ownerID int64, name string,dirctoryId int64) (*Repository, error) {
	repo := &Repository{
		OwnerID:   ownerID,
		LowerName: strings.ToLower(name),
		DirectoryId :  dirctoryId,
	}
	has, err := x.Get(repo)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, errors.RepoNotExist{0, ownerID, name}
	}
	return repo, repo.LoadAttributes()
}



func (repo *Repository) C_Link(userName string) string {
	diretoryId := strconv.FormatInt(repo.DirectoryId,10)
	return setting.AppSubURL + "/syslink/" +userName+"/"+ diretoryId+"/"+ repo.Name
}



// RepoPath returns repository path by given user and repository name.
func C_RepoPath(userName, direftoryId,repoName string) string {
	return filepath.Join(setting.RepoRootPath,setting.CustomURL,strings.ToLower(userName), direftoryId,strings.ToLower(repoName)+".git")
}



// CreateRepository creates a repository for given user or organization.
func C_CreateRepository(doer, owner *User, opts C_CreateRepoOptions) (_ *Repository, err error) {
	if !owner.CanCreateRepo() {
		return nil, errors.ReachLimitOfRepo{owner.RepoCreationNum()}
	}

	repo := &Repository{
		OwnerID:      owner.ID,
		DirectoryId:  opts.DirectoryId,
		Owner:        owner,
		Name:         opts.Name,
		LowerName:    strings.ToLower(opts.Name),
		Description:  opts.Description,
		IsPrivate:    opts.IsPrivate,
		EnableWiki:   true,
		EnableIssues: true,
		EnablePulls:  true,
	}

	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return nil, err
	}
    /////////////////////////////////////////////////////////////////////////////////////////////////////
	if err = createRepository(sess, doer, owner, repo); err != nil {
		return nil, err
	}
	/////////////////////////////////////////////////////////////////////////////////////////////////////



	// No need for init mirror.
	if !opts.IsMirror {

		diretoryId := strconv.FormatInt(opts.DirectoryId,10)

		repoPath := C_RepoPath(owner.Name, diretoryId,repo.Name)


		if err = c_initRepository(sess, repoPath, doer, repo, opts); err != nil {
			RemoveAllWithNotice("Delete repository for initialization failure", repoPath)
			return nil, fmt.Errorf("initRepository: %v", err)
		}

		_, stderr, err := process.ExecDir(-1,
			repoPath, fmt.Sprintf("CreateRepository 'git update-server-info': %s", repoPath),
			"git", "update-server-info")
		if err != nil {
			return nil, fmt.Errorf("CreateRepository 'git update-server-info': %s", stderr)
		}
	}

	return repo, sess.Commit()
}



func c_createRepository(e *xorm.Session, doer, owner *User, repo *Repository) (err error) {
	if err = IsUsableRepoName(repo.Name); err != nil {
		return err
	}

	has, err := isRepositoryExist(e, owner, repo.Name)
	//has, err := isRepositoryExist1(e, repo.Name)
	if err != nil {
		return fmt.Errorf("IsRepositoryExist: %v", err)
	} else if has {
		return ErrRepoAlreadyExist{owner.Name, repo.Name}
	}

	if _, err = e.Insert(repo); err != nil {
		return err
	}

	owner.NumRepos++
	// Remember visibility preference.
	owner.LastRepoVisibility = repo.IsPrivate
	if err = updateUser(e, owner); err != nil {
		return fmt.Errorf("updateUser: %v", err)
	}

	// Give access to all members in owner team.
	if owner.IsOrganization() {
		t, err := owner.getOwnerTeam(e)
		if err != nil {
			return fmt.Errorf("getOwnerTeam: %v", err)
		} else if err = t.addRepository(e, repo); err != nil {
			return fmt.Errorf("addRepository: %v", err)
		}
	} else {
		// Organization automatically called this in addRepository method.
		if err = repo.recalculateAccesses(e); err != nil {
			return fmt.Errorf("recalculateAccesses: %v", err)
		}
	}

	if err = watchRepo(e, owner.ID, repo.ID, true); err != nil {
		return fmt.Errorf("watchRepo: %v", err)
	} else if err = newRepoAction(e, doer, owner, repo); err != nil {
		return fmt.Errorf("newRepoAction: %v", err)
	}

	return repo.loadAttributes(e)
}



// initRepository performs initial commit with chosen setup files on behave of doer.
func c_initRepository(e Engine, repoPath string, doer *User, repo *Repository, opts C_CreateRepoOptions) (err error) {
	// Somehow the directory could exist.
	if com.IsExist(repoPath) {
		return fmt.Errorf("initRepository: path already exists: %s", repoPath)
	}

	// Init bare new repository.
	if err = git.InitRepository(repoPath, true); err != nil {
		return fmt.Errorf("InitRepository: %v", err)
	} else if err = createDelegateHooks(repoPath); err != nil {
		return fmt.Errorf("createDelegateHooks: %v", err)
	}

	tmpDir := filepath.Join(os.TempDir(), "gogs-"+repo.Name+"-"+com.ToStr(time.Now().Nanosecond()))

	// Initialize repository according to user's choice.
	if opts.AutoInit {
		os.MkdirAll(tmpDir, os.ModePerm)
		defer RemoveAllWithNotice("Delete repository for auto-initialization", tmpDir)

		if err = c_prepareRepoCommit(repo, tmpDir, repoPath, opts); err != nil {
			return fmt.Errorf("prepareRepoCommit: %v", err)
		}

		// Apply changes and commit.
		if err = initRepoCommit(tmpDir, doer.NewGitSig()); err != nil {
			return fmt.Errorf("initRepoCommit: %v", err)
		}
	}

	// Re-fetch the repository from database before updating it (else it would
	// override changes that were done earlier with sql)
	if repo, err = getRepositoryByID(e, repo.ID); err != nil {
		return fmt.Errorf("getRepositoryByID: %v", err)
	}

	if !opts.AutoInit {
		repo.IsBare = true
	}

	repo.DefaultBranch = "master"
	if err = updateRepository(e, repo, false); err != nil {
		return fmt.Errorf("updateRepository: %v", err)
	}

	return nil
}



func c_prepareRepoCommit(repo *Repository, tmpDir, repoPath string, opts C_CreateRepoOptions) error {
	// Clone to temprory path and do the init commit.
	_, stderr, err := process.Exec(
		fmt.Sprintf("initRepository(git clone): %s", repoPath), "git", "clone", repoPath, tmpDir)
	if err != nil {
		return fmt.Errorf("git clone: %v - %s", err, stderr)
	}

	// README
	data, err := getRepoInitFile("readme", opts.Readme)
	if err != nil {
		return fmt.Errorf("getRepoInitFile[%s]: %v", opts.Readme, err)
	}

	//cloneLink := repo.CloneLink()
	// match := map[string]string{
	// 	"Name":           repo.Name,
	// 	"Description":    repo.Description,
	// 	"CloneURL.SSH":   cloneLink.SSH,
	// 	"CloneURL.HTTPS": cloneLink.HTTPS,
	// }

	//if err = ioutil.WriteFile(filepath.Join(tmpDir, "README.md"),
	//	[]byte(com.Expand(string(data), match)), 0644); err != nil {
	//	return fmt.Errorf("write README.md: %v", err)
	//}


	//替换package.mo模板为项目名称
	pkgContent := string(data)
	newpkgContent := strings.Replace(pkgContent, "syslinkPkg", repo.Name, -1)
	if err = ioutil.WriteFile(filepath.Join(tmpDir, "package.mo"),
		[]byte(newpkgContent), 0644); err != nil {
		return fmt.Errorf("write package.mo: %v", err)
	}

	// .gitignore
	if len(opts.Gitignores) > 0 {
		var buf bytes.Buffer
		names := strings.Split(opts.Gitignores, ",")
		for _, name := range names {
			data, err = getRepoInitFile("gitignore", name)
			if err != nil {
				return fmt.Errorf("getRepoInitFile[%s]: %v", name, err)
			}
			buf.WriteString("# ---> " + name + "\n")
			buf.Write(data)
			buf.WriteString("\n")
		}

		if buf.Len() > 0 {
			if err = ioutil.WriteFile(filepath.Join(tmpDir, ".gitignore"), buf.Bytes(), 0644); err != nil {
				return fmt.Errorf("write .gitignore: %v", err)
			}
		}
	}

	// LICENSE
	if len(opts.License) > 0 {
		data, err = getRepoInitFile("license", opts.License)
		if err != nil {
			return fmt.Errorf("getRepoInitFile[%s]: %v", opts.License, err)
		}

		if err = ioutil.WriteFile(filepath.Join(tmpDir, "LICENSE"), data, 0644); err != nil {
			return fmt.Errorf("write LICENSE: %v", err)
		}
	}

	return nil
}


