package strs

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	EMPTY = ""
)

// Int64Of String to number
// str: Corresponding string
// d: Default return value if not found
func Int64Of(str string, d int64) int64 {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return d
	}
	return i
}

func BoolOf(str string) bool {
	if str == "true" ||
		str == "TRUE" ||
		str == "yes" ||
		str == "YES" {
		return true
	}
	return false
}

func KeyVerify(str string) bool {
	if len(str) > 64 {
		return false
	}
	match, err := regexp.MatchString("^[a-zA-Z0-9_@-]+$", str)
	if err != nil {
		return false
	}
	return match
}

func PureNameVerify(str string) bool {
	return !strings.ContainsAny(str, " \t\n\r!@#$%^&*()_+{}|:\"<>?`-=[]\\;',./")
}

// FBCut Intercept before and after, remaining specified length
func FBCut(str string, max int) string {
	l := len(str)
	if l <= max {
		return str
	}
	if max <= 10 {
		return str[:max]
	}
	f := (max - 10) / 2
	b := l - ((max - 10) - f)
	return str[:f] + "**********" + str[b:]
}
