package repo

import (
	"gopkg.in/macaron.v1"
	"strings"
	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/models/errors"
	"github.com/gogits/gogs/pkg/setting"
	"net/http"
	"github.com/gogits/gogs/pkg/tool"
	"time"
	"github.com/gogits/gogs/pkg/context"
	log "gopkg.in/clog.v1"
	"strconv"
	"fmt"
	"path"
	"os"
)

func C_HTTPContexter() macaron.Handler {
	return func(c *context.Context) {
		ownerName := c.Params(":username")
		directoryId,_ := strconv.ParseInt(c.Params(":directoryId") , 10, 64)
		repoName := strings.TrimSuffix(c.Params(":reponame"), ".git")
		repoName = strings.TrimSuffix(repoName, ".wiki")


		isPull := c.Query("service") == "git-upload-pack" ||
			strings.HasSuffix(c.Req.URL.Path, "git-upload-pack") ||
			c.Req.Method == "GET"
        fmt.Print("55555555555",isPull)

		owner, err := models.GetUserByName(ownerName)
		if err != nil {
			c.NotFoundOrServerError("GetUserByName", errors.IsUserNotExist, err)
			return
		}


		repo, err := models.GetRepositoryByNameAndDirectoryId(owner.ID,repoName,directoryId)
		if err != nil {
			c.NotFoundOrServerError("GetRepositoryByNameAndDirectoryId", errors.IsRepoNotExist, err)
			return
		}


        fmt.Print("333333333")
		// Authentication is not required for pulling from public repositories.
		if isPull && !repo.IsPrivate && !setting.Service.RequireSignInView {
			c.Map(&HTTPContext{
				Context: c,
			})
			return
		}

		////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		//////////////////////////////////////////////克隆公有库到此为止///////////////////////////////////////////////
		////////////////////////////////////////////////////////////////////////////////////////////////////////////////
       fmt.Print("444444444")
		// In case user requested a wrong URL and not intended to access Git objects.
		action := c.Params("*")
		if !strings.Contains(action, "git-") &&
			!strings.Contains(action, "info/") &&
			!strings.Contains(action, "HEAD") &&
			!strings.Contains(action, "objects/") {
			c.NotFound()
			return
		}
        fmt.Print("22222222222")
		// Handle HTTP Basic Authentication
		authHead := c.Req.Header.Get("Authorization")
		if len(authHead) == 0 {
			askCredentials(c, http.StatusUnauthorized, "")
			return
		}

		auths := strings.Fields(authHead)
		if len(auths) != 2 || auths[0] != "Basic" {
			askCredentials(c, http.StatusUnauthorized, "")
			return
		}
		authUsername, authPassword, err := tool.BasicAuthDecode(auths[1])
		if err != nil {
			askCredentials(c, http.StatusUnauthorized, "")
			return
		}

		authUser, err := models.UserSignIn(authUsername, authPassword)
		if err != nil && !errors.IsUserNotExist(err) {
			c.Handle(http.StatusInternalServerError, "UserSignIn", err)
			return
		}

		// If username and password combination failed, try again using username as a token.
		if authUser == nil {
			token, err := models.GetAccessTokenBySHA(authUsername)
			if err != nil {
				if models.IsErrAccessTokenEmpty(err) || models.IsErrAccessTokenNotExist(err) {
					askCredentials(c, http.StatusUnauthorized, "")
				} else {
					c.Handle(http.StatusInternalServerError, "GetAccessTokenBySHA", err)
				}
				return
			}
			token.Updated = time.Now()

			authUser, err = models.GetUserByID(token.UID)
			if err != nil {
				// Once we found token, we're supposed to find its related user,
				// thus any error is unexpected.
				c.Handle(http.StatusInternalServerError, "GetUserByID", err)
				return
			}
		} else if authUser.IsEnabledTwoFactor() {
			askCredentials(c, http.StatusUnauthorized, `User with two-factor authentication enabled cannot perform HTTP/HTTPS operations via plain username and password
Please create and use personal access token on user settings page`)
			return
		}

		log.Trace("HTTPGit - Authenticated user: %s", authUser.Name)

		mode := models.ACCESS_MODE_WRITE
		if isPull {
			mode = models.ACCESS_MODE_READ
		}
		has, err := models.HasAccess(authUser.ID, repo, mode)
		if err != nil {
			c.Handle(http.StatusInternalServerError, "HasAccess", err)
			return
		} else if !has {
			askCredentials(c, http.StatusForbidden, "User permission denied")
			return
		}

		if !isPull && repo.IsMirror {
			c.HandleText(http.StatusForbidden, "Mirror repository is read-only")
			return
		}

		c.Map(&HTTPContext{
			Context:   c,
			OwnerName: ownerName,
			OwnerSalt: owner.Salt,
			RepoID:    repo.ID,
			RepoName:  repoName,
			AuthUser:  authUser,
		})
	}



}


func C_HTTP(c *HTTPContext) {
	for _, route := range routes {
		reqPath := strings.ToLower(c.Req.URL.Path)
		m := route.reg.FindStringSubmatch(reqPath)

		if m == nil {
			continue
		}

		// We perform check here because routes matched in cmd/web.go is wider than needed,
		// but we only want to output this message only if user is really trying to access
		// Git HTTP endpoints.
		if setting.Repository.DisableHTTPGit {
			c.HandleText(http.StatusForbidden, "Interacting with repositories by HTTP protocol is not disabled")
			return
		}

		if route.method != c.Req.Method {
			c.NotFound()
			return
		}

		file := strings.TrimPrefix(reqPath, m[1]+"/")
		dir, err := c_getGitRepoPath(m[1])
		if err != nil {
			log.Warn("HTTP.getGitRepoPath: %v", err)
			c.NotFound()
			return
		}
       fmt.Print("4444")
		route.handler(serviceHandler{
			w:    c.Resp,
			r:    c.Req.Request,
			dir:  dir,
			file: file,

			authUser:  c.AuthUser,
			ownerName: c.OwnerName,
			ownerSalt: c.OwnerSalt,
			repoID:    c.RepoID,
			repoName:  c.RepoName,
		})
		return
	}

	c.NotFound()
}


func c_getGitRepoPath(dir string) (string, error) {
	if !strings.HasSuffix(dir, ".git") {
		dir += ".git"
	}
	fmt.Print(dir)
	fmt.Print(setting.RepoRootPath)

	filename := path.Join(setting.RepoRootPath, dir)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", err
	}

	return filename, nil
}

