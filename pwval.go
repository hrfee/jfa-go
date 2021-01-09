package main

import (
	"unicode"
)

// Validator allows for validation of passwords.
type Validator struct {
	minLength, upper, lower, number, special int
	criteria                                 ValidatorConf
}

type ValidatorConf map[string]int

func (vd *Validator) init(criteria ValidatorConf) {
	vd.criteria = criteria
}

// This isn't used, its for swagger
type PasswordValidation struct {
	Characters bool `json:"length,omitempty"`    // Number of characters
	Lowercase  bool `json:"lowercase,omitempty"` // Number of lowercase characters
	Uppercase  bool `json:"uppercase,omitempty"` // Number of uppercase characters
	Numbers    bool `json:"number,omitempty"`    // Number of numbers
	Specials   bool `json:"special,omitempty"`   // Number of special characters
}

func (vd *Validator) validate(password string) map[string]bool {
	count := map[string]int{}
	for key := range vd.criteria {
		count[key] = 0
	}
	for _, c := range password {
		count["length"] += 1
		if unicode.IsUpper(c) {
			count["uppercase"] += 1
		} else if unicode.IsLower(c) {
			count["lowercase"] += 1
		} else if unicode.IsNumber(c) {
			count["number"] += 1
		} else if unicode.ToUpper(c) == unicode.ToLower(c) {
			count["special"] += 1
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

func (vd *Validator) getCriteria() ValidatorConf {
	criteria := ValidatorConf{}
	for key, num := range vd.criteria {
		if num != 0 {
			criteria[key] = num
		}
	}
	return criteria
}
