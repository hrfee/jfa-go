// Routes for triggering background tasks manually.
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary List existing task routes, with friendly names and descriptions.
// @Produce json
// @Success 200 {object} TasksDTO
// @Router /tasks [get]
// @Security Bearer
// @tags Tasks
func (app *appContext) TaskList(gc *gin.Context) {
	resp := TasksDTO{Tasks: []TaskDTO{
		TaskDTO{
			URL:         "/tasks/housekeeping",
			Name:        "Housekeeping",
			Description: "General housekeeping tasks: Clearing expired invites & activities, unused contact details & captchas, etc.",
		},
		TaskDTO{
			URL:         "/tasks/users",
			Name:        "Users",
			Description: "Checks for (pending) account expiries and performs the appropriate actions.",
		},
	}}
	if app.config.Section("jellyseerr").Key("enabled").MustBool(false) {
		resp.Tasks = append(resp.Tasks, TaskDTO{
			URL:         "/tasks/jellyseerr",
			Name:        "Jellyseerr user import",
			Description: "Imports existing users into jellyfin and synchronizes contact details. Should only need to be run once when the feature is enabled, which is done automatically.",
		})
	}
	gc.JSON(200, resp)
}

// @Summary Triggers general housekeeping tasks: Clearing expired invites, activities, unused contact details, captchas, etc.
// @Success 204
// @Router /tasks/housekeeping [post]
// @Security Bearer
// @tags Tasks
func (app *appContext) TaskHousekeeping(gc *gin.Context) {
	app.housekeepingDaemon.Trigger()
	gc.Status(http.StatusNoContent)
}

// @Summary Triggers check for account expiry.
// @Success 204
// @Router /tasks/users [post]
// @Security Bearer
// @tags Tasks
func (app *appContext) TaskUserCleanup(gc *gin.Context) {
	app.userDaemon.Trigger()
	gc.Status(http.StatusNoContent)
}

// @Summary Triggers sync of user details with Jellyseerr. Not usually needed after one run, details are synced on change anyway.
// @Success 204
// @Router /tasks/jellyseerr [post]
// @Security Bearer
// @tags Tasks
func (app *appContext) TaskJellyseerrImport(gc *gin.Context) {
	if app.jellyseerrDaemon != nil {
		app.jellyseerrDaemon.Trigger()
	} else {
		app.SynchronizeJellyseerrUsers()
	}
	gc.Status(http.StatusNoContent)
}
