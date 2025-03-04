package variable

import configure "github.com/jom-io/gorig/utils/cofigure"

var (
	DateFormat = "2006-01-02 15:04:05"
	TBPrefix   = ""
)

func init() {
	TBPrefix = configure.GetString("db.prefix", TBPrefix)
}
