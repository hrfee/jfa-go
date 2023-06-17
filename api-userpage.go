package main

import "github.com/gin-gonic/gin"

// @Summary Returns the logged-in user's Jellyfin ID & Username.
// @Produce json
// @Success 200 {object} MyDetailsDTO
// @Router /my/details [get]
// @tags User Page
func (app *appContext) MyDetails(gc *gin.Context) {
	resp := MyDetailsDTO{
		Id: gc.GetString("jfId"),
	}

	user, status, err := app.jf.UserByID(resp.Id, false)
	if status != 200 || err != nil {
		app.err.Printf("Failed to get Jellyfin user (%d): %+v\n", status, err)
		respond(500, "Failed to get user", gc)
		return
	}
	resp.Username = user.Name

	gc.JSON(200, resp)
}
