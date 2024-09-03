package sys

import "strings"

// Mode Run Mode Defined
type Mode string

// [dev|local|test|pre|prod]
const (
	LOCAL Mode = "local"
	DEV   Mode = "dev"
	TEST  Mode = "test"
	PRE   Mode = "pre"
	PROD  Mode = "prod"
)

func ModeValueOf(mode string) Mode {
	lowerMode := strings.ToLower(mode)
	switch lowerMode {
	case string(LOCAL):
		return LOCAL
	case string(DEV):
		return DEV
	case string(TEST):
		return TEST
	case string(PRE):
		return PRE
	case string(PROD):
		return PROD
	}
	return LOCAL
}

func (r Mode) IsRd() bool {
	return r.IsLocal() ||
		r.IsDev() ||
		r.IsTest()
}

func (r Mode) IsLocal() bool {
	return r == LOCAL
}

func (r Mode) IsDev() bool {
	return r == DEV
}

func (r Mode) IsTest() bool {
	return r == TEST
}

func (r Mode) IsPre() bool {
	return r == PRE
}

func (r Mode) IsProd() bool {
	return r == PROD
}
