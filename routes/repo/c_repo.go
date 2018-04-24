package repo

import (
	//"bytes"
	//"fmt"
	//gotemplate "html/template"
	//"io/ioutil"
	//"path"
	"strings"

	//"github.com/Unknwon/paginater"
	log "gopkg.in/clog.v1"

	"github.com/gogits/git-module"

	//"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/pkg/context"
	//"github.com/gogits/gogs/pkg/markup"
	//"github.com/gogits/gogs/pkg/setting"
	//"github.com/gogits/gogs/pkg/template"
	//"github.com/gogits/gogs/pkg/template/highlight"
	//"github.com/gogits/gogs/pkg/tool"
	//"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/pkg/setting"
	"github.com/gogits/gogs/pkg/mailer"
	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/models/errors"


	//"fmt"
	"github.com/gogits/gogs/pkg/tool"
	"github.com/gogits/gogs/pkg/form"
	"strconv"
)



type fileInfo struct {

	FileName string
	Message string
	CommitID   string
	CommitDate string
	Size string
	IsDir bool

}

type latestCommit struct {
	ID string
	Author *git.Signature
	Committer *git.Signature

}

func C_Home(c *context.Context) {
	c.Data["PageIsViewFiles"] = true

	if c.Repo.Repository.IsBare {
		c.HTML(200, BARE)
		return
	}

	title := c.Repo.Repository.Owner.Name + "/" + c.Repo.Repository.Name
	if len(c.Repo.Repository.Description) > 0 {
		title += ": " + c.Repo.Repository.Description
	}
	c.Data["Title"] = title
	if c.Repo.BranchName != c.Repo.Repository.DefaultBranch {
		c.Data["Title"] = title + " @ " + c.Repo.BranchName
	}
	c.Data["RequireHighlightJS"] = true

	branchLink := c.Repo.RepoLink + "/src/" + c.Repo.BranchName
	treeLink := branchLink
	rawLink := c.Repo.RepoLink + "/raw/" + c.Repo.BranchName

	isRootDir := false
	if len(c.Repo.TreePath) > 0 {
		treeLink += "/" + c.Repo.TreePath
	} else {
		isRootDir = true

		// Only show Git stats panel when view root directory
		var err error
		c.Repo.CommitsCount, err = c.Repo.Commit.CommitsCount()
		if err != nil {
			c.Handle(500, "CommitsCount", err)
			return
		}
		c.Data["CommitsCount"] = c.Repo.CommitsCount
	}
	c.Data["PageIsRepoHome"] = isRootDir

	// Get current entry user currently looking at.
	entry, err := c.Repo.Commit.GetTreeEntryByPath(c.Repo.TreePath)



	if err != nil {
		c.NotFoundOrServerError("Repo.Commit.GetTreeEntryByPath", git.IsErrNotExist, err)
		return
	}



	if entry.IsDir() {

		renderDirectory(c, treeLink)

	} else {

		renderFile(c, entry, treeLink, rawLink)
	}
	if c.Written() {
		return
	}

	setEditorconfigIfExists(c)
	if c.Written() {
		return
	}

	var treeNames []string
	paths := make([]string, 0, 5)
	if len(c.Repo.TreePath) > 0 {
		treeNames = strings.Split(c.Repo.TreePath, "/")
		for i := range treeNames {
			paths = append(paths, strings.Join(treeNames[:i+1], "/"))
		}

		c.Data["HasParentPath"] = true
		if len(paths)-2 >= 0 {
			c.Data["ParentPath"] = "/" + paths[len(paths)-2]
		}
	}

	c.Data["Paths"] = paths
	c.Data["TreeLink"] = treeLink
	c.Data["TreeNames"] = treeNames
	c.Data["BranchLink"] = branchLink

	commitsInfo := c.Data["Files"].([][]interface{})
	var files []fileInfo
	for _, info := range commitsInfo {
		entry:= info[0].(*git.TreeEntry)
		commit:= info[1].(*git.Commit)
		var file fileInfo
		file.FileName = entry.Name()
		file.Message = commit.CommitMessage
		file.CommitID = commit.ID.String()
		file.CommitDate = commit.Committer.When.String()
		file.Size = tool.FileSize(entry.Blob().Size())
		file.IsDir = entry.IsDir()
		files = append(files, file)
	}

	res := make(map[string]interface{})

	res["Files"] = files


	lCommit := c.Data["LatestCommit"].(*git.Commit)
	var ll latestCommit
	ll.Committer = lCommit.Committer
	ll.Author = lCommit.Author
	ll.ID = lCommit.ID.String()
	//fmt.Println(lCommit)

    res["LatestCommit"] = ll
	res["LatestCommitUser"] = c.Data["LatestCommitUser"]
	res["NumWatches"] = c.Repo.Repository.NumWatches
	res["NumStars"] = c.Repo.Repository.NumStars
	res["NumForks"] = c.Repo.Repository.NumForks
	res["CommitsCount"] = c.Data["CommitsCount"]
	res["BrancheCount"] = c.Data["BrancheCount"]
	res["Branches"] = c.Data["Branches"]
	res["Releases"] = c.Repo.Repository.NumTags
	res["Tags"] = c.Data["Tags"]
	res["Repository"] = c.Repo




	c.JSON(200,res)

	//c.HTML(200, HOME)

}



func C_GetFiles(c *context.Context) {


	res := make(map[string]interface{})


	// Get current entry user currently looking at.
	entry, err := c.Repo.Commit.GetTreeEntryByPath(c.Repo.TreePath)



	if err != nil {
		c.NotFoundOrServerError("Repo.Commit.GetTreeEntryByPath", git.IsErrNotExist, err)
		return
	}

	branchLink := c.Repo.RepoLink + "/src/" + c.Repo.BranchName
	treeLink := branchLink
	rawLink := c.Repo.RepoLink + "/raw/" + c.Repo.BranchName

	if entry.IsDir() {

		renderDirectory(c, treeLink)

		var files []fileInfo
		commitsInfo := c.Data["Files"].([][]interface{})
		for _, info := range commitsInfo {
			entry:= info[0].(*git.TreeEntry)
			commit:= info[1].(*git.Commit)
			var file fileInfo
			file.FileName = entry.Name()
			file.Message = commit.CommitMessage
			file.CommitID = commit.ID.String()
			file.CommitDate = commit.Committer.When.String()
			file.Size = tool.FileSize(entry.Blob().Size())
			file.IsDir = entry.IsDir()
			files = append(files, file)
		}

		res["Files"] = files

	} else {

		renderFile(c, entry, treeLink, rawLink)
		res["File"] = c.Data["FileText"]
	}




	c.JSON(200,res)



}


func C_SettingsCollaboration(c *context.Context) {
	//fmt.Println(111,c.Req.Header)
	c.Data["Title"] = c.Tr("repo.settings")
	c.Data["PageIsSettingsCollaboration"] = true

	users, err := c.Repo.Repository.GetCollaborators()
	if err != nil {
		c.Handle(500, "GetCollaborators", err)
		return
	}
	c.Data["Collaborators"] = users


	res := make(map[string]interface{})

	res["collaborators"] = users
    c.JSON(200,res)
	//c.HTML(200, SETTINGS_COLLABORATION)
}

func C_SettingsCollaborationPost(c *context.Context) {

	name := strings.ToLower(c.Query("collaborator"))
	if len(name) == 0 || c.Repo.Owner.LowerName == name {
		c.Redirect(setting.AppSubURL + c.Req.URL.Path)
		return
	}

	u, err := models.GetUserByName(name)
	if err != nil {
		if errors.IsUserNotExist(err) {
			c.Flash.Error(c.Tr("form.user_not_exist"))
			c.Redirect(setting.AppSubURL + c.Req.URL.Path)
		} else {
			c.Handle(500, "GetUserByName", err)
		}
		return
	}

	// Organization is not allowed to be added as a collaborator
	if u.IsOrganization() {
		c.Flash.Error(c.Tr("repo.settings.org_not_allowed_to_be_collaborator"))
		c.Redirect(setting.AppSubURL + c.Req.URL.Path)
		return
	}

	if err = c.Repo.Repository.AddCollaborator(u); err != nil {
		c.Handle(500, "AddCollaborator", err)
		return
	}

	if setting.Service.EnableNotifyMail {
		mailer.SendCollaboratorMail(models.NewMailerUser(u), models.NewMailerUser(c.User), models.NewMailerRepo(c.Repo.Repository))
	}

	c.Flash.Success(c.Tr("repo.settings.add_collaborator_success"))
	c.Redirect(setting.AppSubURL + c.Req.URL.Path)
}



//
//func Token(c *context.Context) {
//
//
//	if c.IsLogged {
//		if !c.User.IsActive && setting.Service.RegisterEmailConfirm {
//			c.Data["Title"] = c.Tr("auth.active_your_account")
//			c.Success(user.ACTIVATE)
//		} else {
//			user.Dashboard(c)
//		}
//		return
//	}
//
//
//	// Check auto-login.
//	uname := c.GetCookie(setting.CookieUserName)
//	if len(uname) != 0 {
//		fmt.Print("555555"+setting.CookieUserName+"55555")
//		c.Redirect(setting.AppSubURL + "/user/login")
//		return
//	}
//	c.Redirect(setting.AppSubURL + "/user/logout" )
//	c.Data["PageIsHome"] = true
//	c.JSON(200,c.GetCookie(setting.CSRFCookieName))
//}



func C_CreatePost(c *context.Context, f form.C_CreateRepo) {
	c.Data["Title"] = c.Tr("new_repo")

	c.Data["Gitignores"] = models.Gitignores
	c.Data["Licenses"] = models.Licenses
	c.Data["Readmes"] = models.Readmes

	ctxUser := checkContextUser(c, f.UserID)
	if c.Written() {
		return
	}
	c.Data["ContextUser"] = ctxUser

	if c.HasError() {
		c.HTML(200, CREATE)
		return
	}

	repo, err := models.C_CreateRepository(c.User, ctxUser, models.C_CreateRepoOptions{
		Name:        f.RepoName,
		DirectoryId:  f.DirectoryId,
		Description: f.Description,
		Gitignores:  f.Gitignores,
		License:     f.License,
		Readme:      f.Readme,
		IsPrivate:   f.Private || setting.Repository.ForcePrivate,
		AutoInit:    f.AutoInit,
	})
	if err == nil {
		log.Trace("Repository created [%d]: %s/%s", repo.ID, ctxUser.Name, repo.Name)
		c.Redirect(setting.AppSubURL + "/" +setting.CustomURL+ "/"+ctxUser.Name + "/" +strconv.FormatInt(f.DirectoryId,10)+"/"+ repo.Name)
		return
	}

	if repo != nil {
		if errDelete := models.DeleteRepository(ctxUser.ID, repo.ID); errDelete != nil {
			log.Error(4, "DeleteRepository: %v", errDelete)
		}
	}

	handleCreateError(c, ctxUser, err, "CreatePost", CREATE, &f)
}
