// Routes for triggering background tasks manually.
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
