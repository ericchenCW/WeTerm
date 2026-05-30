include .env
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

# procmon 嵌入二进制相关
PROCMON_SRC=./procmon/cmd/procmon
PROCMON_EMBED=pages/procmon/assets/procmon-linux-amd64

# 引入 go.work 后改用 workspace 构建（弃用 vendor）。
# go build / gox 会从模块缓存解析依赖，本地 inspect module 由 go.work 解析。
setup:
	go work sync

build: build_gox

build_gox:
	$(GOX) -os="linux darwin" -arch="amd64 arm64" -output="bin/weterm_{{.OS}}_{{.Arch}}" -ldflags="-X weterm/cmd.Hash=$(GITHASH) -X weterm/cmd.BuildTime=$(BUILDTIME) -X weterm/cmd.GitUser=$(GITUSER) -X weterm/cmd.Version=$(VERSION)"

# 重新生成嵌入的 procmon 采集器二进制（静态 linux/amd64）。
# procmon 是独立 module（不在 go.work），用 GOWORK=off 隔离 workspace，
# 保持其零依赖、可分发到任意 Linux 的特性。
# 注意：procmon 源码改动后必须重跑本目标，否则分发的是旧二进制。
build-procmon:
	cd $(PROCMON_SRC) && GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags "-s -w" -o $(CURDIR)/$(PROCMON_EMBED) .
	@file $(PROCMON_EMBED) || true

clean:
	rm -vf $(BINARY_PATH)*

run: setup
	go run main.go

# 如果存在SYNC_COMMAND变量，则执行同步命令
sync: build
	if [ -n "$(SYNC_COMMAND)" ]; then $(SYNC_COMMAND); fi
