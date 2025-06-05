package domainx

import (
	"github.com/jom-io/gorig/serv"
	"github.com/jom-io/gorig/utils/sys"
)

const ServiceCode = "DATABASE"

func init() {
	if service == nil {
		service = new(serviceInfo)
	}
	if err := serv.RegisterService(
		serv.Service{
			Code:     ServiceCode,
			Startup:  service.Start,
			Shutdown: service.End,
		},
	); err != nil {
		sys.Exit(err)
	}
}
