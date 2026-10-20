package handlerutil

import (
	"errors"
	"regexp/syntax"
	"slices"

	"github.com/distr-sh/distr/internal/validation"
)

const maxFilterLength = 200

func ValidateFilterRegex(filter string) error {
	if re, err := validation.ParseRegex(filter, maxFilterLength); err != nil {
		return err
	} else if hasNestedQuantifier(re) {
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
