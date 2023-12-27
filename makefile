# 可执行文件名
BINARY_NAME=bin/weterm

# 确保命令在 $PATH 找不到是，使用绝对路径
GOBUILD=go build
GIT=git
DATE=date
GITHASH=$(shell $(GIT) rev-parse HEAD)
BUILDTIME=$(shell $(DATE) +%FT%T%z)
GITUSER=$(shell $(GIT) config user.name)
VERSION?=0.0.0
BINARY_NAME_AMD64=bin/weterm_amd64
BINARY_NAME_ARM64=bin/weterm_arm64

setup:
	go mod tidy && go mod vendor

build: build_amd64 build_aarch64

build_aarch64:
	GOARCH=arm64 GOOS=linux go build -ldflags="-X weterm/cmd.Hash=$(GITHASH) -X weterm/cmd.BuildTime=$(BUILDTIME) -X weterm/cmd.GitUser=$(GITUSER) -X weterm/cmd.Version=$(VERSION)" -o $(BINARY_NAME_ARM64) main.go

build_amd64:
	GOARCH=amd64 GOOS=linux go build -ldflags="-X weterm/cmd.Hash=$(GITHASH) -X weterm/cmd.BuildTime=$(BUILDTIME) -X weterm/cmd.GitUser=$(GITUSER) -X weterm/cmd.Version=$(VERSION)" -o $(BINARY_NAME_AMD64) main.go


clean:
	rm -vf $(BINARY_NAME)

run:
	go run main.go