package main

import (
	"github.com/gin-gonic/gin"
)

type testReq struct {
	Host     string `json:"jfHost"`
	Username string `json:"jfUser"`
	Password string `json:"jfPassword"`
}

func (app *appContext) TestJF(gc *gin.Context) {
	var req testReq
	gc.BindJSON(&req)
	tempjf := Jellyfin{}
	tempjf.init(req.Host, "jfa-go-setup", app.version, "auth", "auth")
	tempjf.noFail = true
	_, status, err := tempjf.authenticate(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.info.Printf("Auth failed with code %d (%s)", status, err)
		gc.JSON(401, map[string]bool{"success": false})
		return
	}
	gc.JSON(200, map[string]bool{"success": true})
}
