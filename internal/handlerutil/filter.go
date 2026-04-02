package handlerutil

import (
	"errors"
	"regexp/syntax"
	"slices"
)

const maxFilterLength = 200

func ValidateFilterRegex(filter string) error {
	if len(filter) > maxFilterLength {
		return errors.New("filter exceeds maximum length of 200 characters")
	}

	re, err := syntax.Parse(filter, syntax.Perl)
	if err != nil {
		return errors.New("invalid filter regex: " + err.Error())
	}

	if hasNestedQuantifier(re) {
		return errors.New("filter contains nested quantifiers which are not allowed")
	}

	return nil
}

func hasNestedQuantifier(re *syntax.Regexp) bool {
	if isQuantifier(re.Op) {
		if slices.ContainsFunc(re.Sub, subtreeContainsQuantifier) {
			return true
		}
	}

	return slices.ContainsFunc(re.Sub, hasNestedQuantifier)
}

func isQuantifier(op syntax.Op) bool {
	return op == syntax.OpStar || op == syntax.OpPlus || op == syntax.OpQuest || op == syntax.OpRepeat
}

func subtreeContainsQuantifier(re *syntax.Regexp) bool {
	if isQuantifier(re.Op) {
		return true
	}

	return slices.ContainsFunc(re.Sub, subtreeContainsQuantifier)
}
