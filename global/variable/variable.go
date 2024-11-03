package variable

import configure "github.com/jom-io/gorig/utils/cofigure"

var (
	DateFormat = "2006-01-02 15:04:05" //  设置全局日期时间格式
	TBPrefix   = ""
)

func init() {
	TBPrefix = configure.GetString("db.prefix", TBPrefix)
}
