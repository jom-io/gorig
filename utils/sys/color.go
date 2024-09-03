package sys

import (
	"fmt"
	"github.com/gookit/color"
	"strings"
)

type colorWriter struct {
}

func (c *colorWriter) Write(p []byte) (n int, err error) {
	Info(string(p))
	return len(p), nil
}

type noneWriter struct {
}

func (c *noneWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var ColorWriter = &colorWriter{}
var NoneWriter = &noneWriter{}

func Info(arg ...any) {
	color.Style{color.FgBlue}.Println(doFormat(arg...))
}

func Success(arg ...any) {
	color.Style{color.FgGreen}.Println(doFormat(arg...))
}

func Warn(arg ...any) {
	color.Style{color.FgYellow}.Println(doFormat(arg...))
}

func Error(arg ...any) {
	color.Style{color.FgRed}.Println(doFormat(arg...))
}

func doFormat(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	buf := strings.Builder{}
	lenIdx := 0
	for _, item := range args {
		str := fmt.Sprintf("%v", item)
		lenIdx += len(str)
		buf.WriteString(str)
	}
	for i := lenIdx; i < 100; i++ {
		buf.WriteString(" ")
	}
	msg := buf.String()
	if consoleToLogger != nil {
		consoleToLogger(msg)
	}
	return "[ " + msg + " ]"
}

var consoleToLogger func(msg string)

func ConsoleToLogger(call func(msg string)) {
	consoleToLogger = call
}
