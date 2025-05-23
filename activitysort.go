package main

import (
	"fmt"
	"strings"

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

// AsDBQuery returns a mutated "query" filtering for the conditions in "q".
func (q QueryDTO) AsDBQuery(query *badgerhold.Query) *badgerhold.Query {
	if query == nil {
		query = &badgerhold.Query{}
	}
	// Special case for activity type:
	// In the app, there isn't an "activity:<fieldname>" query, but rather "<~fieldname>:true/false" queries.
	// For other API consumers, we also handle the former later.
	activityType := activityTypeGetterNameToType(q.Field)
	if activityType != ActivityUnknown {
		criterion := query.And("Type")
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
	if fieldName == "unknown" {
		panic("FIXME: Support all the weird queries of the web UI!")
	}
	criterion := query.And(fieldName)

	// FIXME: Deal with dates like we do in usercache.go
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
	subQuery := &badgerhold.Query{}
	// I don't believe you can just do Where("*"), so instead run for each field.
	for _, fieldName := range []string{"ID", "Type", "UserID", "Username", "SourceType", "Source", "SourceUsername", "InviteCode", "Value", "IP"} {
		criterion := badgerhold.Where(fieldName)
		// No case-insentive Contains method, so we use MatchFunc instead
		subQuery = subQuery.Or(criterion.MatchFunc(func(ra *badgerhold.RecordAccess) (bool, error) {
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
		}))
	}

	return subQuery
}
