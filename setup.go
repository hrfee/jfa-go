package main

import (
	"github.com/gin-gonic/gin"
)

type testReq struct {
	Host     string `json:"jfHost"`
	Username string `json:"jfUser"`
	Password string `json:"jfPassword"`
}

func (ctx *appContext) TestJF(gc *gin.Context) {
	var req testReq
	gc.BindJSON(&req)
	tempjf := Jellyfin{}
	tempjf.init(req.Host, "jfa-go-setup", ctx.version, "auth", "auth")
	_, status, err := tempjf.authenticate(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		ctx.info.Printf("Auth failed with code %d (%s)", status, err)
		gc.JSON(401, map[string]bool{"success": false})
		return
	}
	gc.JSON(200, map[string]bool{"success": true})
}
