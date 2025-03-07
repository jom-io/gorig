package variable

import configure "github.com/jom-io/gorig/utils/cofigure"

var (
	SysName  = ""
	TBPrefix = ""
	JwtKey   = ""
	OMKey    = ""
)

func init() {
	TBPrefix = configure.GetString("db.prefix", TBPrefix)
	JwtKey = configure.GetString("jwt.key", "")
	SysName = configure.GetString("sys.name", "")
	OMKey = configure.GetString("om.key", "")
}
