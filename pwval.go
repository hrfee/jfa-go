package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Validator struct {
	minLength, upper, lower, number, special int
	criteria                                 ValidatorConf
	specialChars                             []rune
}

type ValidatorConf map[string]int

func (vd *Validator) init(criteria ValidatorConf) {
	vd.specialChars = []rune{'[', '@', '_', '!', '#', '$', '%', '^', '&', '*', '(', ')', '<', '>', '?', '/', '\\', '|', '}', '{', '~', ':', ']'}
	vd.criteria = criteria
}

// This isn't used, its for swagger
type PasswordValidation struct {
	Characters bool `json:"characters,omitempty"`           // Number of characters
	Lowercase  bool `json:"lowercase characters,omitempty"` // Number of lowercase characters
	Uppercase  bool `json:"uppercase characters,omitempty"` // Number of uppercase characters
	Numbers    bool `json:"numbers,omitempty"`              // Number of numbers
	Specials   bool `json:"special characters,omitempty"`   // Number of special characters
}

func (vd *Validator) validate(password string) map[string]bool {
	count := map[string]int{}
	for key := range vd.criteria {
		count[key] = 0
	}
	for _, c := range password {
		count["characters"] += 1
		if unicode.IsUpper(c) {
			count["uppercase characters"] += 1
		} else if unicode.IsLower(c) {
			count["lowercase characters"] += 1
		} else if unicode.IsNumber(c) {
			count["numbers"] += 1
		} else {
			for _, s := range vd.specialChars {
				if c == s {
					count["special characters"] += 1
				}
			}
		}
	}
	results := map[string]bool{}
	for criterion, num := range count {
		if num < vd.criteria[criterion] {
			results[criterion] = false
		} else {
			results[criterion] = true
		}
	}
	return results
}

func (vd *Validator) getCriteria() map[string]string {
	lines := map[string]string{}
	for criterion, min := range vd.criteria {
		if min > 0 {
			text := fmt.Sprintf("Must have at least %d ", min)
			if min == 1 {
				text += strings.TrimSuffix(criterion, "s")
			} else {
				text += criterion
			}
			lines[criterion] = text
		}
	}
	return lines
}
