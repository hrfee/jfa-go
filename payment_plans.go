package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/ini.v1"
)

const stripePaymentPlansSetting = "payment_plans"

var paymentPlanIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type PaymentPlan struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	Enabled             bool   `json:"enabled"`
	Price               int64  `json:"price"`
	Currency            string `json:"currency"`
	Recurring           bool   `json:"recurring"`
	StripeInterval      string `json:"stripe_interval,omitempty"`
	StripeIntervalCount int64  `json:"stripe_interval_count,omitempty"`
	AccessMonths        int    `json:"access_months,omitempty"`
	AccessDays          int    `json:"access_days,omitempty"`
	Profile             string `json:"profile,omitempty"`
}

type paymentPlansDTO struct {
	Plans []PaymentPlan `json:"plans"`
}

type storePlanView struct {
	ID          string
	Name        string
	Description string
	Price       string
	Currency    string
	Billing     string
	Access      string
}

func defaultPaymentPlans(monthlyPrice int64, currency string) []PaymentPlan {
	currency = normalizePaymentCurrency(currency)
	if monthlyPrice <= 0 {
		monthlyPrice = 200
	}
	return []PaymentPlan{
		{
			ID:                  "monthly",
			Name:                paymentPlanMonthly,
			Description:         "Recurring monthly access.",
			Enabled:             true,
			Price:               monthlyPrice,
			Currency:            currency,
			Recurring:           true,
			StripeInterval:      "month",
			StripeIntervalCount: 1,
			AccessMonths:        1,
			Profile:             paymentDefaultProfile,
		},
		{
			ID:                  "quarterly",
			Name:                "Quarterly",
			Description:         "Recurring access billed every three months.",
			Enabled:             true,
			Price:               monthlyPrice * 3,
			Currency:            currency,
			Recurring:           true,
			StripeInterval:      "month",
			StripeIntervalCount: 3,
			AccessMonths:        3,
			Profile:             paymentDefaultProfile,
		},
		{
			ID:                  "yearly",
			Name:                "Yearly",
			Description:         "Recurring annual access.",
			Enabled:             true,
			Price:               monthlyPrice * 12,
			Currency:            currency,
			Recurring:           true,
			StripeInterval:      "year",
			StripeIntervalCount: 1,
			AccessMonths:        12,
			Profile:             paymentDefaultProfile,
		},
		{
			ID:           "one_month",
			Name:         "One Month Pass",
			Description:  "One-time access for one month.",
			Enabled:      false,
			Price:        monthlyPrice,
			Currency:     currency,
			Recurring:    false,
			AccessMonths: 1,
			Profile:      paymentDefaultProfile,
		},
	}
}

func defaultPaymentPlansJSON(monthlyPrice int64, currency string) string {
	b, _ := json.Marshal(defaultPaymentPlans(monthlyPrice, currency))
	return string(b)
}

func normalizePaymentCurrency(currency string) string {
	currency = strings.ToLower(strings.TrimSpace(currency))
	if len(currency) != 3 {
		return "usd"
	}
	return currency
}

func normalizePaymentPlanID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, " ", "_")
	out := strings.Builder{}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '_' || r == '-':
			out.WriteRune(r)
		}
	}
	return strings.Trim(out.String(), "_-")
}

func normalizeStripeInterval(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "day", "week", "month", "year":
		return strings.ToLower(strings.TrimSpace(interval))
	default:
		return "month"
	}
}

func normalizePaymentPlans(plans []PaymentPlan, defaultCurrency string) ([]PaymentPlan, error) {
	defaultCurrency = normalizePaymentCurrency(defaultCurrency)
	seen := map[string]bool{}
	out := make([]PaymentPlan, 0, len(plans))
	for i, plan := range plans {
		plan.ID = normalizePaymentPlanID(plan.ID)
		if plan.ID == "" {
			plan.ID = normalizePaymentPlanID(plan.Name)
		}
		if plan.ID == "" || !paymentPlanIDPattern.MatchString(plan.ID) {
			return nil, fmt.Errorf("plan %d has an invalid ID", i+1)
		}
		if seen[plan.ID] {
			return nil, fmt.Errorf("duplicate plan ID %q", plan.ID)
		}
		seen[plan.ID] = true

		plan.Name = strings.TrimSpace(plan.Name)
		if plan.Name == "" {
			return nil, fmt.Errorf("plan %q needs a name", plan.ID)
		}
		plan.Description = strings.TrimSpace(plan.Description)
		if plan.Price <= 0 {
			return nil, fmt.Errorf("plan %q needs a positive price", plan.ID)
		}
		plan.Currency = normalizePaymentCurrency(firstNonEmpty(plan.Currency, defaultCurrency))
		if plan.Profile == "" {
			plan.Profile = paymentDefaultProfile
		}
		if plan.AccessMonths < 0 || plan.AccessDays < 0 {
			return nil, fmt.Errorf("plan %q has a negative access duration", plan.ID)
		}
		if plan.AccessMonths == 0 && plan.AccessDays == 0 {
			return nil, fmt.Errorf("plan %q needs an access duration", plan.ID)
		}
		if plan.Recurring {
			plan.StripeInterval = normalizeStripeInterval(plan.StripeInterval)
			if plan.StripeIntervalCount <= 0 {
				plan.StripeIntervalCount = 1
			}
		} else {
			plan.StripeInterval = ""
			plan.StripeIntervalCount = 0
		}
		out = append(out, plan)
	}
	if len(out) == 0 {
		return nil, errors.New("at least one payment plan is required")
	}
	return out, nil
}

func (app *appContext) paymentPlans() []PaymentPlan {
	defaultCurrency := app.config.Section("stripe").Key("price_currency").MustString("usd")
	defaultMonthly := app.config.Section("stripe").Key("price_monthly").MustInt64(200)
	raw := strings.TrimSpace(app.config.Section("stripe").Key(stripePaymentPlansSetting).String())
	if raw == "" {
		return defaultPaymentPlans(defaultMonthly, defaultCurrency)
	}

	var plans []PaymentPlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		if app.err != nil {
			app.err.Printf("Failed to parse Stripe payment plans, using defaults: %v", err)
		}
		return defaultPaymentPlans(defaultMonthly, defaultCurrency)
	}
	plans, err := normalizePaymentPlans(plans, defaultCurrency)
	if err != nil {
		if app.err != nil {
			app.err.Printf("Invalid Stripe payment plans, using defaults: %v", err)
		}
		return defaultPaymentPlans(defaultMonthly, defaultCurrency)
	}
	return plans
}

func (app *appContext) publicPaymentPlans() []PaymentPlan {
	plans := app.paymentPlans()
	out := make([]PaymentPlan, 0, len(plans))
	for _, plan := range plans {
		if plan.Enabled {
			out = append(out, plan)
		}
	}
	return out
}

func (app *appContext) paymentPlanByID(id string) (PaymentPlan, bool) {
	id = strings.TrimSpace(id)
	normalized := normalizePaymentPlanID(id)
	for _, plan := range app.paymentPlans() {
		if plan.ID == normalized || strings.EqualFold(plan.Name, id) {
			app.warnIfPaymentPlanProfileAdmin(plan)
			return plan, true
		}
	}
	return PaymentPlan{}, false
}

func (app *appContext) savePaymentPlans(plans []PaymentPlan) error {
	plans, err := normalizePaymentPlans(plans, app.config.Section("stripe").Key("price_currency").MustString("usd"))
	if err != nil {
		return err
	}
	for _, plan := range plans {
		app.warnIfPaymentPlanProfileAdmin(plan)
	}
	encoded, err := json.Marshal(plans)
	if err != nil {
		return err
	}

	tempConfig, err := ini.ShadowLoad(app.configPath)
	if err != nil {
		return err
	}
	tempConfig.Section("stripe").Key(stripePaymentPlansSetting).SetValue(string(encoded))
	if len(plans) > 0 {
		tempConfig.Section("stripe").Key("price_currency").SetValue(plans[0].Currency)
		if monthly, ok := planByLegacyName(plans, paymentPlanMonthly); ok {
			tempConfig.Section("stripe").Key("price_monthly").SetValue(strconv.FormatInt(monthly.Price, 10))
		}
	}
	if err = tempConfig.SaveTo(app.configPath); err != nil {
		return err
	}
	app.ReloadConfig()
	app.PatchConfigBase()
	return nil
}

func (app *appContext) warnIfPaymentPlanProfileAdmin(plan PaymentPlan) {
	if app == nil || app.storage == nil || app.err == nil || plan.Profile == "" {
		return
	}
	profile, ok := app.storage.GetProfileKey(plan.Profile)
	if !ok || !profile.Policy.IsAdministrator {
		return
	}
	app.err.Printf("Stripe payment plan %q uses profile %q with Jellyfin administrator privileges; paid users created with this plan will become admins.", plan.Name, plan.Profile)
}

func planByLegacyName(plans []PaymentPlan, name string) (PaymentPlan, bool) {
	for _, plan := range plans {
		if strings.EqualFold(plan.Name, name) || strings.EqualFold(plan.ID, name) {
			return plan, true
		}
	}
	return PaymentPlan{}, false
}

func (p PaymentPlan) metadata() map[string]string {
	return map[string]string{
		stripeMetadataPlanID:        p.ID,
		stripeMetadataPlan:          p.Name,
		stripeMetadataProfile:       p.Profile,
		stripeMetadataAccessMonths:  strconv.Itoa(p.AccessMonths),
		stripeMetadataAccessDays:    strconv.Itoa(p.AccessDays),
		stripeMetadataRecurring:     strconv.FormatBool(p.Recurring),
		stripeMetadataInterval:      p.StripeInterval,
		stripeMetadataIntervalCount: strconv.FormatInt(p.StripeIntervalCount, 10),
	}
}

func paymentPlanSnapshotFromMetadata(metadata map[string]string) paymentPlanSnapshot {
	if metadata == nil {
		return paymentPlanSnapshot{}
	}
	months, _ := strconv.Atoi(metadata[stripeMetadataAccessMonths])
	days, _ := strconv.Atoi(metadata[stripeMetadataAccessDays])
	intervalCount, _ := strconv.ParseInt(metadata[stripeMetadataIntervalCount], 10, 64)
	recurring, _ := strconv.ParseBool(metadata[stripeMetadataRecurring])
	return paymentPlanSnapshot{
		ID:                  metadata[stripeMetadataPlanID],
		Name:                normalizePaymentPlanNameFromMetadata(metadata[stripeMetadataPlan]),
		Profile:             metadata[stripeMetadataProfile],
		AccessMonths:        months,
		AccessDays:          days,
		Recurring:           recurring,
		StripeInterval:      metadata[stripeMetadataInterval],
		StripeIntervalCount: intervalCount,
	}
}

func normalizePaymentPlanNameFromMetadata(plan string) string {
	plan = strings.TrimSpace(plan)
	if plan == "" {
		return ""
	}
	return normalizePaymentPlan(plan)
}

type paymentPlanSnapshot struct {
	ID                  string
	Name                string
	Profile             string
	AccessMonths        int
	AccessDays          int
	Recurring           bool
	StripeInterval      string
	StripeIntervalCount int64
}

func (s paymentPlanSnapshot) apply(payment *Payment) {
	if s.ID != "" && payment.PlanID == "" {
		payment.PlanID = s.ID
	}
	if s.Name != "" && payment.Plan == "" {
		payment.Plan = s.Name
	}
	if s.Profile != "" && payment.Profile == "" {
		payment.Profile = s.Profile
	}
	if s.AccessMonths > 0 && payment.AccessMonths == 0 {
		payment.AccessMonths = s.AccessMonths
	}
	if s.AccessDays > 0 && payment.AccessDays == 0 {
		payment.AccessDays = s.AccessDays
	}
	if s.Recurring && !payment.Recurring {
		payment.Recurring = true
	}
	if s.StripeInterval != "" && payment.StripeInterval == "" {
		payment.StripeInterval = s.StripeInterval
	}
	if s.StripeIntervalCount > 0 && payment.StripeIntervalCount == 0 {
		payment.StripeIntervalCount = s.StripeIntervalCount
	}
}

func paymentPlanSnapshotFromPayment(payment Payment) paymentPlanSnapshot {
	return paymentPlanSnapshot{
		ID:                  payment.PlanID,
		Name:                payment.Plan,
		Profile:             payment.Profile,
		AccessMonths:        payment.AccessMonths,
		AccessDays:          payment.AccessDays,
		Recurring:           payment.Recurring,
		StripeInterval:      payment.StripeInterval,
		StripeIntervalCount: payment.StripeIntervalCount,
	}
}

func (app *appContext) paymentPlanSnapshotForSubscription(subscriptionID string, metadata map[string]string) paymentPlanSnapshot {
	snapshot := paymentPlanSnapshotFromMetadata(metadata)
	if snapshot.Name != "" || snapshot.AccessMonths > 0 || snapshot.AccessDays > 0 {
		return snapshot
	}
	for _, payment := range app.storage.GetPayments() {
		if payment.SubscriptionID == subscriptionID {
			snapshot = paymentPlanSnapshotFromPayment(payment)
			if snapshot.Name != "" || snapshot.AccessMonths > 0 || snapshot.AccessDays > 0 {
				return snapshot
			}
		}
	}
	return paymentPlanSnapshot{Name: paymentPlanMonthly, AccessMonths: 1, Recurring: true, StripeInterval: "month", StripeIntervalCount: 1}
}

func paymentPlanExpiry(plan string, accessMonths, accessDays int, base time.Time) (time.Time, bool) {
	if accessMonths > 0 || accessDays > 0 {
		return base.AddDate(0, accessMonths, accessDays), true
	}
	switch normalizePaymentPlan(plan) {
	case paymentPlanMonthly:
		return base.AddDate(0, 1, 0), true
	case paymentPlanStandard:
		return base.AddDate(10, 0, 0), false
	default:
		return base.AddDate(0, 1, 0), true
	}
}

func storePlanViews(plans []PaymentPlan) []storePlanView {
	views := make([]storePlanView, 0, len(plans))
	for _, plan := range plans {
		views = append(views, storePlanView{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Price:       fmt.Sprintf("%.2f", float64(plan.Price)/100.0),
			Currency:    strings.ToUpper(plan.Currency),
			Billing:     planBillingLabel(plan),
			Access:      planAccessLabel(plan),
		})
	}
	return views
}

func planBillingLabel(plan PaymentPlan) string {
	if !plan.Recurring {
		return "One-time payment"
	}
	count := plan.StripeIntervalCount
	if count <= 1 {
		return "Billed every " + plan.StripeInterval
	}
	return fmt.Sprintf("Billed every %d %ss", count, plan.StripeInterval)
}

func planAccessLabel(plan PaymentPlan) string {
	parts := []string{}
	if plan.AccessMonths > 0 {
		unit := "months"
		if plan.AccessMonths == 1 {
			unit = "month"
		}
		parts = append(parts, fmt.Sprintf("%d %s", plan.AccessMonths, unit))
	}
	if plan.AccessDays > 0 {
		unit := "days"
		if plan.AccessDays == 1 {
			unit = "day"
		}
		parts = append(parts, fmt.Sprintf("%d %s", plan.AccessDays, unit))
	}
	if len(parts) == 0 {
		return "Access duration not set"
	}
	return strings.Join(parts, " + ") + " access"
}

func (app *appContext) GetPaymentPlans(gc *gin.Context) {
	gc.JSON(200, paymentPlansDTO{Plans: app.paymentPlans()})
}

func (app *appContext) SetPaymentPlans(gc *gin.Context) {
	var req paymentPlansDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}
	if err := app.savePaymentPlans(req.Plans); err != nil {
		respond(400, err.Error(), gc)
		return
	}
	gc.JSON(200, paymentPlansDTO{Plans: app.paymentPlans()})
}
