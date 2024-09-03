package errors

import (
	"errors"
	"fmt"
)

type Type = int32

const (
	Coding      Type = -999 // code error
	System      Type = -444 // system error
	Application Type = -777 // application error
)

type Error struct {
	Type    Type   `bson:"type" json:"type"`
	Code    string `bson:"code"  json:"code"`
	Message string `bson:"message" json:"message"`

	Err error `bson:"-" json:"-"` // Native error
}

func Of(t Type, code string, message string, nativeErr ...error) *Error {
	err := &Error{
		Type:    t,
		Code:    code,
		Message: message,
	}
	if len(nativeErr) > 0 && nativeErr[0] != nil {
		err.Err = nativeErr[0]
	}
	return err
}

func Assert(expect string, actual string, nativeErr ...error) *Error {
	return Of(Coding,
		"assert",
		fmt.Sprintf("assert failed, expect: %s, but %s", expect, actual),
		nativeErr...)
}

func Sys(message string, nativeErr ...error) *Error {
	return Of(System,
		"sys",
		message,
		nativeErr...)
}

func Verify(message string, nativeErr ...error) *Error {
	return Of(Application, "verify", message, nativeErr...)
}

func VerifyCode(code int, message string, nativeErr ...error) *Error {
	return Of(Application, fmt.Sprintf("%d", code), message, nativeErr...)
}

func (e *Error) CodeInt() int {
	if e.Code == "" {
		return 0
	}
	var code int
	_, err := fmt.Sscanf(e.Code, "%d", &code)
	if err != nil {
		return 0
	}
	return code
}

func (e *Error) Error() string {
	if e.Err != nil && !errors.Is(e, e.Err) {
		return fmt.Sprintf("[%s] %s. ******** %s ********", e.Code, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *Error) IsApplication() bool {
	return e.Type == Application
}

func (e *Error) IsCoding() bool {
	return e.Type == Coding
}

func (e *Error) IsSystem() bool {
	return e.Type == System
}

func (e *Error) Native() error {
	return e.Err
}
