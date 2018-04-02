package user

import (

	"github.com/gogits/gogs/pkg/setting"

	"github.com/gogits/gogs/pkg/context"


	"net/url"
)

func SSO(c *context.Context) {
	c.Data["Title"] = c.Tr("sign_in")

	// Check auto-login.
	isSucceed, err := AutoLogin(c)
	if err != nil {
		c.Handle(500, "AutoLogin", err)
		return
	}

	redirectTo := c.Query("redirect_to")
	if len(redirectTo) > 0 {
		c.SetCookie("redirect_to", redirectTo, 0, setting.AppSubURL)
	} else {
		redirectTo, _ = url.QueryUnescape(c.GetCookie("redirect_to"))
	}
	c.SetCookie("redirect_to", "", -1, setting.AppSubURL)

	if isSucceed {
		if isValidRedirect(redirectTo) {
			c.Redirect(redirectTo)
		} else {
			c.Redirect(setting.AppSubURL + "/token")
		}
		return
	}

	//c.HTML(200, LOGIN)


}
