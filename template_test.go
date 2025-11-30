package main

import (
	"strings"
	"testing"
)

// In == Out when nothing is meant to be templated.
func TestBlankTemplate(t *testing.T) {
	in := `Success, user! Your account has been created. Log in at myAccountURL with your username to get started.`

	out, err := templateEmail(in, []string{}, []string{}, map[string]any{})

	if err != nil {
		t.Fatalf("error: %+v", err)
	}

	if out != in {
		t.Fatalf(`returned string doesn't match input: "%+v" != "%+v"`, out, in)
	}
}

func testConditional(isTrue bool, t *testing.T) {
	in := `Success, {username}! Your account has been created. {if myCondition}Log in at {myAccountURL} with username {username} to get started.{endif}`

	vars := []string{"username", "myAccountURL", "myCondition"}
	conds := vars
	vals := map[string]any{
		"username":     "TemplateUsername",
		"myAccountURL": "TemplateURL",
		"myCondition":  isTrue,
	}

	out, err := templateEmail(in, vars, conds, vals)

	target := ""
	if isTrue {
		target = `Success, {username}! Your account has been created. Log in at {myAccountURL} with username {username} to get started.`
	} else {
		target = `Success, {username}! Your account has been created. `
	}

	target = strings.ReplaceAll(target, "{username}", vals["username"].(string))
	target = strings.ReplaceAll(target, "{myAccountURL}", vals["myAccountURL"].(string))

	if err != nil {
		t.Fatalf("error: %+v", err)
	}

	if out != target {
		t.Fatalf(`returned string doesn't match desired output: "%+v" != "%+v"`, out, target)
	}
}

func TestConditionalTrue(t *testing.T) {
	testConditional(true, t)
}

func TestConditionalFalse(t *testing.T) {
	testConditional(false, t)
}

// Template mistakenly double-braced values, but return a warning.
func TestTemplateDoubleBraceGracefulHandling(t *testing.T) {
	in := `Success, {{username}}! Your account has been created. Log in at {myAccountURL} with username {username} to get started.`

	vars := []string{"username", "myAccountURL"}
	vals := map[string]any{
		"username":     "TemplateUsername",
		"myAccountURL": "TemplateURL",
	}

	target := strings.ReplaceAll(in, "{{username}}", vals["username"].(string))
	target = strings.ReplaceAll(target, "{username}", vals["username"].(string))
	target = strings.ReplaceAll(target, "{myAccountURL}", vals["myAccountURL"].(string))

	out, err := templateEmail(in, vars, []string{}, vals)

	if err == nil {
		t.Fatal("no error when given double-braced variable")
	}

	if out != target {
		t.Fatalf(`returned string doesn't match desired output: "%+v" != "%+v"`, out, target)
	}
}

func TestVarAtAnyPosition(t *testing.T) {
	in := `Success, user! Your account has been created. Log in at myAccountURL with your username to get started.`
	vars := []string{"username", "myAccountURL"}
	vals := map[string]any{
		"username":     "TemplateUsername",
		"myAccountURL": "TemplateURL",
	}

	for i := range in {
		newIn := in[0:i] + "{" + vars[0] + "}" + in[i:]

		target := strings.ReplaceAll(newIn, "{"+vars[0]+"}", vals["username"].(string))

		out, err := templateEmail(newIn, vars, []string{}, vals)

		if err != nil {
			t.Fatalf("error: %+v", err)
		}

		if out != target {
			t.Fatalf(`returned string doesn't match desired output: "%+v" != "%+v, from "%+v""`, out, target, newIn)
		}
	}
}

func TestIncompleteBlock(t *testing.T) {
	in := `Success, user! Your account has been created. Log in at myAccountURL with your username to get started.`
	for i := range in {
		newIn := in[0:i] + "{" + in[i:]

		out, err := templateEmail(newIn, []string{"a"}, []string{"a"}, map[string]any{"a": "a"})

		if out != newIn {
			t.Fatalf(`returned string for position %d/%d doesn't match desired output: "%+v" != "%+v"`, i+1, len(newIn), out, newIn)
		}
		if err == nil {
			t.Fatalf("no error when given incomplete block with brace at position %d/%d", i+1, len(newIn))
		}

	}
}
