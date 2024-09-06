#!/bin/bash
set -ueo pipefail

source /data/install/functions
exec 2>&1

emphasize 停止所有容器
docker stop $(docker ps -aq) 1>/dev/null

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

_clean_file() {
  emphasize 清空${2}下的日志文件
  find ${1} -type f -print -delete
}

_rm_container() {
  emphasize 删除${1}容器
  docker rm -f $(docker ps -aq -f name=${1}*) 2>/dev/null || echo "删除${1}容器失败"
}

_clean_file /data/bkce/logs paas日志
_clean_file /data/weops/casbin-mesh/ casbin日志
_clean_file /data/weops/prometheus/tsdb/ "prometheus tsdb"
_clean_file /var/log/kafka/ kafka日志
_clean_file /var/log/zookeeper/ zk日志
_clean_file /var/log/gse/ gse日志
for i in bk_itsm weops_saas monitorcenter_saas cw_uac_saas bk_monitorv3 bk_nodeman bk_iam bk_user_manage ops-digital_saas bk_sops;do
  _rm_container $i;
done