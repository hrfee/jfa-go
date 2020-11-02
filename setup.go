package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/common"
	"github.com/hrfee/jfa-go/jfapi"
)

type testReq struct {
	Host     string `json:"jfHost"`
	Username string `json:"jfUser"`
	Password string `json:"jfPassword"`
}

func (app *appContext) TestJF(gc *gin.Context) {
	var req testReq
	gc.BindJSON(&req)
	tempjf, _ := jfapi.NewJellyfin(req.Host, "jfa-go-setup", app.version, "auth", "auth", common.NewTimeoutHandler("authJF", req.Host, true), 30)
	_, status, err := tempjf.Authenticate(req.Username, req.Password)
	if !(status == 200 || status == 204) || err != nil {
		app.info.Printf("Auth failed with code %d (%s)", status, err)
		gc.JSON(401, map[string]bool{"success": false})
		return
	}
	gc.JSON(200, map[string]bool{"success": true})
}
