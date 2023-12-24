package main

import (
	"github.com/gin-gonic/gin"
	"github.com/timshannon/badgerhold/v4"
)

func stringToActivityType(v string) ActivityType {
	switch v {
	case "creation":
		return ActivityCreation
	case "deletion":
		return ActivityDeletion
	case "disabled":
		return ActivityDisabled
	case "enabled":
		return ActivityEnabled
	case "contactLinked":
		return ActivityContactLinked
	case "contactUnlinked":
		return ActivityContactUnlinked
	case "changePassword":
		return ActivityChangePassword
	case "resetPassword":
		return ActivityResetPassword
	case "createInvite":
		return ActivityCreateInvite
	case "deleteInvite":
		return ActivityDeleteInvite
	}
	return ActivityUnknown
}

func activityTypeToString(v ActivityType) string {
	switch v {
	case ActivityCreation:
		return "creation"
	case ActivityDeletion:
		return "deletion"
	case ActivityDisabled:
		return "disabled"
	case ActivityEnabled:
		return "enabled"
	case ActivityContactLinked:
		return "contactLinked"
	case ActivityContactUnlinked:
		return "contactUnlinked"
	case ActivityChangePassword:
		return "changePassword"
	case ActivityResetPassword:
		return "resetPassword"
	case ActivityCreateInvite:
		return "createInvite"
	case ActivityDeleteInvite:
		return "deleteInvite"
	}
	return "unknown"
}

func stringToActivitySource(v string) ActivitySource {
	switch v {
	case "user":
		return ActivityUser
	case "admin":
		return ActivityAdmin
	case "anon":
		return ActivityAnon
	case "daemon":
		return ActivityDaemon
	}
	return ActivityAnon
}

func activitySourceToString(v ActivitySource) string {
	switch v {
	case ActivityUser:
		return "user"
	case ActivityAdmin:
		return "admin"
	case ActivityAnon:
		return "anon"
	case ActivityDaemon:
		return "daemon"
	}
	return "anon"
}

// @Summary Get the requested set of activities, Paginated, filtered and sorted.
// @Produce json
// @Param GetActivitiesDTO body GetActivitiesDTO true "search parameters"
// @Success 200 {object} GetActivitiesRespDTO
// @Router /activity [post]
// @Security Bearer
// @tags Activity
func (app *appContext) GetActivities(gc *gin.Context) {
	req := GetActivitiesDTO{}
	gc.BindJSON(&req)
	query := &badgerhold.Query{}
	activityTypes := make([]interface{}, len(req.Type))
	for i, v := range req.Type {
		activityTypes[i] = stringToActivityType(v)
	}
	if len(activityTypes) != 0 {
		query = badgerhold.Where("Type").In(activityTypes...)
	}

	if !req.Ascending {
		query = query.Reverse()
	}

	query = query.SortBy("Time")

	if req.Limit == 0 {
		req.Limit = 10
	}

	query = query.Skip(req.Page * req.Limit).Limit(req.Limit)

	var results []Activity
	err := app.storage.db.Find(&results, query)

	if err != nil {
		app.err.Printf("Failed to read activities from DB: %v\n", err)
	}

	resp := GetActivitiesRespDTO{
		Activities: make([]ActivityDTO, len(results)),
		LastPage:   len(results) != req.Limit,
	}

	for i, act := range results {
		resp.Activities[i] = ActivityDTO{
			ID:         act.ID,
			Type:       activityTypeToString(act.Type),
			UserID:     act.UserID,
			SourceType: activitySourceToString(act.SourceType),
			Source:     act.Source,
			InviteCode: act.InviteCode,
			Value:      act.Value,
			Time:       act.Time.Unix(),
			IP:         act.IP,
		}
		if act.Type == ActivityDeletion || act.Type == ActivityCreation {
			resp.Activities[i].Username = act.Value
			resp.Activities[i].Value = ""
		} else if user, status, err := app.jf.UserByID(act.UserID, false); status == 200 && err == nil {
			resp.Activities[i].Username = user.Name
		}

		if (act.SourceType == ActivityUser || act.SourceType == ActivityAdmin) && act.Source != "" {
			user, status, err := app.jf.UserByID(act.Source, false)
			if status == 200 && err == nil {
				resp.Activities[i].SourceUsername = user.Name
			}
		}
	}

	gc.JSON(200, resp)
}

// @Summary Delete the activity with the given ID. No-op if non-existent, always succeeds.
// @Produce json
// @Param id path string true "ID of activity to delete"
// @Success 200 {object} boolResponse
// @Router /activity/{id} [delete]
// @Security Bearer
// @tags Activity
func (app *appContext) DeleteActivity(gc *gin.Context) {
	app.storage.DeleteActivityKey(gc.Param("id"))
	respondBool(200, true, gc)
}

// @Summary Returns the total number of activities stored in the database.
// @Produce json
// @Success 200 {object} GetActivityCountDTO
// @Router /activity/count [get]
// @Security Bearer
// @tags Activity
func (app *appContext) GetActivityCount(gc *gin.Context) {
	resp := GetActivityCountDTO{}
	var err error
	resp.Count, err = app.storage.db.Count(&Activity{}, &badgerhold.Query{})
	if err != nil {
		resp.Count = 0
	}
	gc.JSON(200, resp)
}
