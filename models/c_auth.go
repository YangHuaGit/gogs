package models

import (
	"github.com/go-xorm/xorm"
	"fmt"
	"time"
)

type Auth struct {
	auth_id     int    `xorm:"<-"`
	parent_id int      `xorm:"<-"`
	auth_code string    `xorm:"<-"`
	auth_name  string  `xorm:"<-"`
	carete_date     time.Time `xorm:"<-"`
	description      string `xorm:"<-"`
	url   string  `xorm:"<-"`
}


func GetUserAuth (uid int64)[]map[string]string{




	engine, _ := xorm.NewEngine("mysql", "root:root@tcp/syslink?charset=utf8")
	//engine.Sync2(new(Auth))

	results, err := engine.QueryString ("select d.* from auth as d INNER JOIN user_auth as l " +
		"on d.auth_id = l.auth_id where l.uid = ? " +
		"UNION  SELECT    c.* FROM auth as c " +
		"INNER JOIN role_auth as r " +
		"ON c.auth_id = r.auth_id " +
		"WHERE r.role_id in (SELECT ur.role_id FROM user_role as ur WHERE ur.uid  = ?)",uid,uid)
	fmt.Print(err)
	//jjj := new(Auth)
	//has, err := engine.Where("auth_id=?", "1").Get(jjj)
	//fmt.Print(has,err)

	return results
}
