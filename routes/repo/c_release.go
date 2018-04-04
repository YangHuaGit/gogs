package repo

import (
	"github.com/gogits/gogs/pkg/form"
	"github.com/gogits/gogs/pkg/setting"
	"github.com/gogits/gogs/models"
	log "gopkg.in/clog.v1"
	"github.com/gogits/gogs/pkg/context"
)

func C_NewReleasePost(c *context.Context, f form.NewRelease) {
	c.Data["Title"] = c.Tr("repo.release.new_release")
	c.Data["PageIsReleaseList"] = true
	renderReleaseAttachmentSettings(c)


	if c.HasError() {
		c.HTML(200, RELEASE_NEW)
		return
	}

	if !c.Repo.GitRepo.IsBranchExist(f.Target) {
		c.RenderWithErr(c.Tr("form.target_branch_not_exist"), RELEASE_NEW, &f)
		return
	}

	// Use current time if tag not yet exist, otherwise get time from Git
	var tagCreatedUnix int64
	tag, err := c.Repo.GitRepo.GetTag(f.TagName)
	if err == nil {
		commit, err := tag.Commit()
		if err == nil {
			tagCreatedUnix = commit.Author.When.Unix()
		}
	}

	commit, err := c.Repo.GitRepo.GetBranchCommit(f.Target)
	if err != nil {
		c.Handle(500, "GetBranchCommit", err)
		return
	}

	commitsCount, err := commit.CommitsCount()
	if err != nil {
		c.Handle(500, "CommitsCount", err)
		return
	}

	var attachments []string
	if setting.Release.Attachment.Enabled {
		attachments = f.Files
	}

	rel := &models.Release{
		RepoID:       c.Repo.Repository.ID,
		PublisherID:  c.User.ID,
		Title:        f.Title,
		TagName:      f.TagName,
		Target:       f.Target,
		Sha1:         commit.ID.String(),
		NumCommits:   commitsCount,
		Note:         f.Content,
		IsDraft:      len(f.Draft) > 0,
		IsPrerelease: f.Prerelease,
		CreatedUnix:  tagCreatedUnix,
	}
	if err = models.NewRelease(c.Repo.GitRepo, rel, attachments); err != nil {
		c.Data["Err_TagName"] = true
		switch {
		case models.IsErrReleaseAlreadyExist(err):
			c.RenderWithErr(c.Tr("repo.release.tag_name_already_exist"), RELEASE_NEW, &f)
		case models.IsErrInvalidTagName(err):
			c.RenderWithErr(c.Tr("repo.release.tag_name_invalid"), RELEASE_NEW, &f)
		default:
			c.Handle(500, "NewRelease", err)
		}
		return
	}
	log.Trace("Release created: %s/%s:%s", c.User.LowerName, c.Repo.Repository.Name, f.TagName)

	//c.Redirect(c.Repo.RepoLink + "/releases")

	c.JSON(200, map[string]interface{}{
		"flag": true,
		"msg": "添加成功",
	})

}




func C_EditReleasePost(c *context.Context, f form.EditRelease) {
	c.Data["Title"] = c.Tr("repo.release.edit_release")
	c.Data["PageIsReleaseList"] = true
	c.Data["PageIsEditRelease"] = true
	renderReleaseAttachmentSettings(c)

	tagName := c.Params("*")
	rel, err := models.GetRelease(c.Repo.Repository.ID, tagName)
	if err != nil {
		if models.IsErrReleaseNotExist(err) {
			c.Handle(404, "GetRelease", err)
		} else {
			c.Handle(500, "GetRelease", err)
		}
		return
	}
	c.Data["tag_name"] = rel.TagName
	c.Data["tag_target"] = rel.Target
	c.Data["title"] = rel.Title
	c.Data["content"] = rel.Note
	c.Data["attachments"] = rel.Attachments
	c.Data["prerelease"] = rel.IsPrerelease
	c.Data["IsDraft"] = rel.IsDraft

	if c.HasError() {
		c.HTML(200, RELEASE_NEW)
		return
	}

	var attachments []string
	if setting.Release.Attachment.Enabled {
		attachments = f.Files
	}

	isPublish := rel.IsDraft && len(f.Draft) == 0
	rel.Title = f.Title
	rel.Note = f.Content
	rel.IsDraft = len(f.Draft) > 0
	rel.IsPrerelease = f.Prerelease
	if err = models.UpdateRelease(c.User, c.Repo.GitRepo, rel, isPublish, attachments); err != nil {
		c.Handle(500, "UpdateRelease", err)
		return
	}

	c.JSON(200, map[string]interface{}{
		"flag": true,
		"msg": "编辑成功",
	})
	//c.Redirect(c.Repo.RepoLink + "/releases")
}


func C_DeleteRelease(c *context.Context) {

	if err := models.DeleteReleaseOfRepoByID(c.Repo.Repository.ID, c.QueryInt64("id")); err != nil {
		c.Flash.Error("DeleteReleaseByID: " + err.Error())
	} else {
		c.Flash.Success(c.Tr("repo.release.deletion_success"))
	}

	//c.JSON(200, map[string]interface{}{
	//	"redirect": c.Repo.RepoLink + "/releases",
	//})
	c.JSON(200, map[string]interface{}{
		"flag": true,
		"msg": "删除成功",
	})
}
