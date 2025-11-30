package main

import (
	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/timshannon/badgerhold/v4"
)

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

// generateActivitiesQuery generates a badgerhold query from QueryDTOs and search terms, which can then be searched, counted, or whatever you want.
func (app *appContext) generateActivitiesQuery(req ServerFilterReqDTO) *badgerhold.Query {

	var query *badgerhold.Query
	if len(req.SearchTerms) != 0 {
		query = ActivityMatchesSearchAsDBBaseQuery(req.SearchTerms)
	} else {
		query = nil
	}

	for _, q := range req.Queries {
		nq := q.AsDBQuery(query)
		if nq == nil {
			nq = ActivityDBQueryFromSpecialField(app.jf, query, q)
		}
		query = nq
	}

	if query == nil {
		query = &badgerhold.Query{}
	}
	return query
}

// @Summary Get the requested set of activities, Paginated, filtered and sorted. Is a POST because of some issues I was having, ideally should be a GET.
// @Produce json
// @Param ServerSearchReqDTO body ServerSearchReqDTO true "search parameters"
// @Success 200 {object} GetActivitiesRespDTO
// @Router /activity [post]
// @Security Bearer
// @tags Activity,Statistics
func (app *appContext) GetActivities(gc *gin.Context) {
	req := ServerSearchReqDTO{}
	gc.BindJSON(&req)
	if req.SortByField == "" {
		req.SortByField = USER_DEFAULT_SORT_FIELD
	} else {
		req.SortByField = activityDTONameToField(req.SortByField)
	}

	query := app.generateActivitiesQuery(req.ServerFilterReqDTO)

	query = query.SortBy(req.SortByField)
	if !req.Ascending {
		query = query.Reverse()
	}

	query = query.Skip(req.Page * req.Limit).Limit(req.Limit)

	var results []Activity
	err := app.storage.db.Find(&results, query)
	if err != nil {
		app.err.Printf(lm.FailedDBReadActivities, err)
	}

	resp := GetActivitiesRespDTO{
		Activities: make([]ActivityDTO, len(results)),
	}
	resp.LastPage = len(results) != req.Limit
	for i, act := range results {
		resp.Activities[i] = ActivityDTO{
			ID:             act.ID,
			Type:           activityTypeToString(act.Type),
			UserID:         act.UserID,
			SourceType:     activitySourceToString(act.SourceType),
			Source:         act.Source,
			InviteCode:     act.InviteCode,
			Value:          act.Value,
			Time:           act.Time.Unix(),
			IP:             act.IP,
			Username:       act.MustGetUsername(app.jf),
			SourceUsername: act.MustGetSourceUsername(app.jf),
		}
		if act.Type == ActivityDeletion || act.Type == ActivityCreation {
			// Username would've been in here, clear it to avoid confusion to the consumer
			resp.Activities[i].Value = ""
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
// @Success 200 {object} PageCountDTO
// @Router /activity/count [get]
// @Security Bearer
// @tags Activity,Statistics
func (app *appContext) GetActivityCount(gc *gin.Context) {
	resp := PageCountDTO{}
	var err error
	resp.Count, err = app.storage.db.Count(&Activity{}, &badgerhold.Query{})
	if err != nil {
		resp.Count = 0
	}
	gc.JSON(200, resp)
}

// @Summary Returns the total number of activities matching the given filtering. Fails silently.
// @Produce json
// @Param ServerFilterReqDTO body ServerFilterReqDTO true "search parameters"
// @Success 200 {object} PageCountDTO
// @Router /activity/count [post]
// @Security Bearer
// @tags Activity,Statistics
func (app *appContext) GetFilteredActivityCount(gc *gin.Context) {
	resp := PageCountDTO{}
	req := ServerFilterReqDTO{}
	gc.BindJSON(&req)

	query := app.generateActivitiesQuery(req)

	var err error
	resp.Count, err = app.storage.db.Count(&Activity{}, query)
	if err != nil {
		// app.err.Printf(lm.FailedDBReadActivities, err)
		resp.Count = 0
	}
	gc.JSON(200, resp)
}
