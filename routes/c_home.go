package routes

import (

	"github.com/gogits/gogs/pkg/context"
	"github.com/gogits/gogs/pkg/setting"
	"github.com/gogits/gogs/routes/user"
	"fmt"
)

func Token(c *context.Context) {


	if c.IsLogged {
		fmt.Println(11111)
		if !c.User.IsActive && setting.Service.RegisterEmailConfirm {
			c.Data["Title"] = c.Tr("auth.active_your_account")
			c.Success(user.ACTIVATE)
		} else {
			user.C_Dashboard(c)
			csrf := make(map[string]interface{})
			csrf["csrf"] = c.GetCookie(setting.CSRFCookieName)
			//csrf["Feeds"] = c.Data["Feeds"]
			c.JSON(200,csrf)

		}

		return
	}


	// Check auto-login.
	uname := c.GetCookie(setting.CookieUserName)
	fmt.Println("123",uname)
	fmt.Println("123456",len(uname))
	if len(uname) != 0 {
		fmt.Print("555555"+setting.CookieUserName+"55555")
		c.Redirect(setting.AppSubURL + "/sso")
		return
	}
	c.Redirect(setting.AppSubURL + "/user/logout" )
	c.Data["PageIsHome"] = true


}

func BBB(c *context.Context){







	//c.Params(":path")
	//fmt.Print(c.Params(":path"))
	//b, err := redis.Dial("tcp", "127.0.0.1:6379")
	//if err != nil {
		//fmt.Println("Connect to redis error", err)
		//return
	//}
	//
	//defer b.Close()
	//c.Session.Set("asdas","dsadsa")
    //fmt.Sprint(c.Session)
    //c.JSON(200,c.Session)










}


