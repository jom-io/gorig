.PHONY:  build-api

apiBinName="gorig-local.linux64"

all:
	go env -w GOARCH=amd64
	go env -w GOOS=linux
	go env -w CGO_ENABLED=0
	go env -w GO111MODULE=on
	#go env -w GOPROXY=https://goproxy.cn,direct
	go mod  tidy

local:
	go env -w GOARCH=arm64
	go env -w GOOS=darwin
	go env -w CGO_ENABLED=0
	go env -w GO111MODULE=on
	go mod  tidy

build-api:all clean-api build-api-bin
build-api-bin:
	go build -o ${apiBinName}  -ldflags "-w -s"  -trimpath  ./simple/main.go

clean-api:
	@if [ -f ${apiBinName} ] ; then rm -rf ${apiBinName} ; fi