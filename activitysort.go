package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/hrfee/mediabrowser"
	"github.com/timshannon/badgerhold/v4"
)

const (
	ACTIVITY_DEFAULT_SORT_FIELD = "Time"
	// This will be default anyway, as the default value of a bool field is false.
	// ACTIVITY_DEFAULT_SORT_ASCENDING = false
)

func activityDTONameToField(field string) string {
	// Only "ID" and "Time" of these are actually searched by the UI.
	// We support the rest though for other consumers of the API.
	switch field {
	case "id":
		return "ID"
	case "type":
		return "Type"
	case "user_id":
		return "UserID"
	case "username":
		return "Username"
	case "source_type":
		return "SourceType"
	case "source":
		return "Source"
	case "source_username":
		return "SourceUsername"
	case "invite_code":
		return "InviteCode"
	case "value":
		return "Value"
	case "time":
		return "Time"
	case "ip":
		return "IP"
	}
	return "unknown"
	// Only these query types actually search the ActivityDTO data.
}

func activityTypeGetterNameToType(getter string) ActivityType {
	switch getter {
	case "accountCreation":
		return ActivityCreation
	case "accountDeletion":
		return ActivityDeletion
	case "accountDisabled":
		return ActivityDisabled
	case "accountEnabled":
		return ActivityEnabled
	case "contactLinked":
		return ActivityContactLinked
	case "contactUnlinked":
		return ActivityContactUnlinked
	case "passwordChange":
		return ActivityChangePassword
	case "passwordReset":
		return ActivityResetPassword
	case "inviteCreated":
		return ActivityCreateInvite
	case "inviteDeleted":
		return ActivityDeleteInvite
	}
	return ActivityUnknown
}

// andField appends to the existing query if not nil, and otherwise creates a new one.
func andField(q *badgerhold.Query, field string) *badgerhold.Criterion {
	if q == nil {
		return badgerhold.Where(field)
	}
	return q.And(field)
}

// AsDBQuery returns a mutated "query" filtering for the conditions in "q".
func (q QueryDTO) AsDBQuery(query *badgerhold.Query) *badgerhold.Query {
	// Special case for activity type:
	// In the app, there isn't an "activity:<fieldname>" query, but rather "<~fieldname>:true/false" queries.
	// For other API consumers, we also handle the former later.
	activityType := activityTypeGetterNameToType(q.Field)
	if activityType != ActivityUnknown {
		criterion := andField(query, "Type")
		if q.Operator != EqualOperator {
			panic(fmt.Errorf("impossible operator for activity type: %v", q.Operator))
		}
		if q.Value.(bool) == true {
			query = criterion.Eq(activityType)
		} else {
			query = criterion.Ne(activityType)
		}
		return query
	}

	fieldName := activityDTONameToField(q.Field)
	// Fail if unrecognized, or recognized as time (we handle this with DateAttempt.Compare separately).
	if fieldName == "unknown" || fieldName == "Time" {
		// Caller is expected to fall back to ActivityDBQueryFromSpecialField after this.
		return nil
	}
	criterion := andField(query, fieldName)

	switch q.Operator {
	case LesserOperator:
		query = criterion.Lt(q.Value)
	case EqualOperator:
		query = criterion.Eq(q.Value)
	case GreaterOperator:
		query = criterion.Gt(q.Value)
	}
	return query
}

// ActivityMatchesSearchAsDBBaseQuery returns a base query (which you should then apply other mutations to) matching the search "term" to Activities by searching all fields. Does not search the generated title like the web app.
func ActivityMatchesSearchAsDBBaseQuery(terms []string) *badgerhold.Query {
	var baseQuery *badgerhold.Query = nil
	// I don't believe you can just do Where("*"), so instead run for each field.
	// FIXME: Match username and source_username and source_type and type
	for _, fieldName := range []string{"ID", "UserID", "Source", "InviteCode", "Value", "IP"} {
		criterion := badgerhold.Where(fieldName)
		// No case-insentive Contains method, so we use MatchFunc instead
		f := criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
			field := ra.Field()
			// _, ok := field.(string)
			// if !ok {
			// 	return false, fmt.Errorf("field not string: %s", fieldName)
			// }
			lower := strings.ToLower(field.(string))
			for _, term := range terms {
				if strings.Contains(lower, term) {
					return true, nil
				}
			}
			return false, nil
		})
		if baseQuery == nil {
			baseQuery = f
		} else {
			baseQuery = baseQuery.Or(f)
		}
	}

	return baseQuery
}

func (act Activity) SourceIsUser() bool {
	return (act.SourceType == ActivityUser || act.SourceType == ActivityAdmin) && act.Source != ""
}

func (act Activity) MustGetUsername(jf *mediabrowser.MediaBrowser) string {
	if act.Type == ActivityDeletion || act.Type == ActivityCreation {
		return act.Value
	}
	if act.UserID == "" {
		return ""
	}
	// Don't care abt errors, user.Name will be blank in that case anyway
	user, _ := jf.UserByID(act.UserID, false)
	return user.Name
}

func (act Activity) MustGetSourceUsername(jf *mediabrowser.MediaBrowser) string {
	if !act.SourceIsUser() {
		return ""
	}
	// Don't care abt errors, user.Name will be blank in that case anyway
	user, _ := jf.UserByID(act.Source, false)
	return user.Name
}

func ActivityDBQueryFromSpecialField(jf *mediabrowser.MediaBrowser, query *badgerhold.Query, q QueryDTO) *badgerhold.Query {
	switch q.Field {
	case "mentionedUsers":
		query = matchMentionedUsersAsQuery(jf, query, q)
	case "actor":
		query = matchActorAsQuery(jf, query, q)
	case "referrer":
		query = matchReferrerAsQuery(jf, query, q)
	case "time":
		query = matchTimeAsQuery(query, q)
	default:
		panic(fmt.Errorf("unknown activity query field %s", q.Field))
	}
	return query
}

// matchMentionedUsersAsQuery is a custom match function for the "mentionedUsers" getter/query type.
func matchMentionedUsersAsQuery(jf *mediabrowser.MediaBrowser, query *badgerhold.Query, q QueryDTO) *badgerhold.Query {
	criterion := andField(query, "UserID")
	query = criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
		act := ra.Record().(*Activity)
		usernames := act.MustGetUsername(jf) + " " + act.MustGetSourceUsername(jf)
		return strings.Contains(strings.ToLower(usernames), strings.ToLower(q.Value.(string))), nil
	})
	return query
}

// matchActorAsQuery is a custom match function for the "actor" getter/query type.
func matchActorAsQuery(jf *mediabrowser.MediaBrowser, query *badgerhold.Query, q QueryDTO) *badgerhold.Query {
	criterion := andField(query, "SourceType")
	query = criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
		act := ra.Record().(*Activity)
		matchString := activitySourceToString(act.SourceType)
		if act.SourceType == ActivityAdmin || act.SourceType == ActivityUser && act.SourceIsUser() {
			matchString += " " + act.MustGetSourceUsername(jf)
		}
		return strings.Contains(strings.ToLower(matchString), strings.ToLower(q.Value.(string))), nil
	})
	return query
}

// matchReferrerAsQuery is a custom match function for the "referrer" getter/query type.
func matchReferrerAsQuery(jf *mediabrowser.MediaBrowser, query *badgerhold.Query, q QueryDTO) *badgerhold.Query {
	criterion := andField(query, "Type")
	query = criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
		act := ra.Record().(*Activity)
		if act.Type == ActivityCreation || act.SourceType == ActivityUser || !act.SourceIsUser() {
			return false, nil
		}
		return strings.Contains(strings.ToLower(act.MustGetSourceUsername(jf)), strings.ToLower(q.Value.(string))), nil
	})
	return query
}

// mathcTimeAsQuery is a custom match function for the "time" getter/query type. Roughly matches the same way as the web app, and in usercache.go.
func matchTimeAsQuery(query *badgerhold.Query, q QueryDTO) *badgerhold.Query {
	operator := Equal
	switch q.Operator {
	case LesserOperator:
		operator = Lesser
	case EqualOperator:
		operator = Equal
	case GreaterOperator:
		operator = Greater
	}
	criterion := andField(query, "Time")
	query = criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
		return q.Value.(DateAttempt).Compare(ra.Field().(time.Time)) == int(operator), nil
	})
	return query
}
