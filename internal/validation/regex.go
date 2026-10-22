package validation

import (
	"errors"
	"fmt"
	"regexp/syntax"
)

// ParseRegex parses pattern with the flags regexp.Compile uses, so a pattern it accepts compiles.
func ParseRegex(pattern string, maxLength int) (*syntax.Regexp, error) {
	if len(pattern) > maxLength {
		return nil, fmt.Errorf("regular expression must not be longer than %d characters", maxLength)
	}
	re, err := syntax.Parse(pattern, syntax.Perl)
	if syntaxErr, ok := errors.AsType[*syntax.Error](err); ok {
		return nil, fmt.Errorf("invalid regular expression: %v", syntaxErr.Code)
	} else if err != nil {
		return nil, errors.New("invalid regular expression")
	}
	return re, nil
}
