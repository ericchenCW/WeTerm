#!/bin/bash
set -ueo pipefail

source /data/install/functions

emphasize 停止所有容器
docker stop $(docker ps -aq)

emphasize 停止docker服务
systemctl stop docker

emphasize 停止gseagent和相关进程
cd /usr/local/gse/agent/bin
./gsectl stop
cd /usr/local/gse/plugins/bin
echo bkmonitorbeat  bkunifylogbeat  exceptionbeat  gsecmdline | xargs -n 1 ./stop.sh

cd /data/install
emphasize 停止蓝鲸服务
echo bkmonitorv3 appo bknodeman job cmdb gse iam usermgr paas license yum  | xargs -n 1 ./bkcli stop

emphasize 检查服务停止状态
echo bkmonitorv3 appo bknodeman job cmdb gse iam usermgr paas license yum  | xargs -n 1 ./bkcli status

emphasize 停止第三方组件
echo kafka influxdb zk es7 redis mongodb rabbitmq mysql nginx consul | xargs -n 1 ./bkcli stop

emphasize 检查第三方组件状态
echo kafka influxdb zk es7 redis mongodb rabbitmq mysql nginx consul | xargs -n 1 ./bkcli stop

emphasize 清空bkce下的日志文件
find /data/bkce/logs -type f -delete