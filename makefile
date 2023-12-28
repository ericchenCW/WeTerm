# 可执行文件名
BINARY_NAME=bin/weterm

# 确保命令在 $PATH 找不到是，使用绝对路径
GOBUILD=go build
GIT=git
GOX=gox
DATE=date
GITHASH=$(shell $(GIT) rev-parse HEAD)
BUILDTIME=$(shell $(DATE) +%FT%T%z)
GITUSER=$(shell $(GIT) config user.name)
VERSION?=0.0.0
BINARY_PATH=bin/

setup:
	go mod tidy && go mod vendor

build: build_gox

build_gox:
	$(GOX) -os="linux darwin" -arch="amd64 arm64" -output="bin/weterm_{{.OS}}_{{.Arch}}" -ldflags="-X weterm/cmd.Hash=$(GITHASH) -X weterm/cmd.BuildTime=$(BUILDTIME) -X weterm/cmd.GitUser=$(GITUSER) -X weterm/cmd.Version=$(VERSION)"

clean:
	rm -vf $(BINARY_PATH)*

run: setup
	go run main.go