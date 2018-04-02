package user

import (
	"github.com/gogits/gogs/pkg/setting"
	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/pkg/context"
)

func C_Dashboard(c *context.Context) {
	ctxUser := getDashboardContextUser(c)
	if c.Written() {
		return
	}

	retrieveFeeds(c, ctxUser, c.User.ID, false)
	if c.Written() {
		return
	}

	if c.Req.Header.Get("X-AJAX") == "true" {
		c.HTML(200, NEWS_FEED)
		return
	}

	c.Data["Title"] = ctxUser.DisplayName() + " - " + c.Tr("dashboard")
	c.Data["PageIsDashboard"] = true
	c.Data["PageIsNews"] = true

	// Only user can have collaborative repositories.
	if !ctxUser.IsOrganization() {
		collaborateRepos, err := c.User.GetAccessibleRepositories(setting.UI.User.RepoPagingNum)
		if err != nil {
			c.Handle(500, "GetAccessibleRepositories", err)
			return
		} else if err = models.RepositoryList(collaborateRepos).LoadAttributes(); err != nil {
			c.Handle(500, "RepositoryList.LoadAttributes", err)
			return
		}
		c.Data["CollaborativeRepos"] = collaborateRepos
	}

	var err error
	var repos, mirrors []*models.Repository
	var repoCount int64
	if ctxUser.IsOrganization() {
		repos, repoCount, err = ctxUser.GetUserRepositories(c.User.ID, 1, setting.UI.User.RepoPagingNum)
		if err != nil {
			c.Handle(500, "GetUserRepositories", err)
			return
		}

		mirrors, err = ctxUser.GetUserMirrorRepositories(c.User.ID)
		if err != nil {
			c.Handle(500, "GetUserMirrorRepositories", err)
			return
		}
	} else {
		if err = ctxUser.GetRepositories(1, setting.UI.User.RepoPagingNum); err != nil {
			c.Handle(500, "GetRepositories", err)
			return
		}
		repos = ctxUser.Repos
		repoCount = int64(ctxUser.NumRepos)

		mirrors, err = ctxUser.GetMirrorRepositories()
		if err != nil {
			c.Handle(500, "GetMirrorRepositories", err)
			return
		}
	}
	c.Data["Repos"] = repos
	c.Data["RepoCount"] = repoCount
	c.Data["MaxShowRepoNum"] = setting.UI.User.RepoPagingNum

	if err := models.MirrorRepositoryList(mirrors).LoadAttributes(); err != nil {
		c.Handle(500, "MirrorRepositoryList.LoadAttributes", err)
		return
	}
	c.Data["MirrorCount"] = len(mirrors)
	c.Data["Mirrors"] = mirrors

	//c.HTML(200, DASHBOARD)
}
