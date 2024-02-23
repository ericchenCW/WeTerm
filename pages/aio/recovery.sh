#!/bin/bash
# 此脚本用于weops一体机安装后的ip恢复
export LC_ALL=en_US.UTF-8
set -eo pipefail
source /root/.bkrc
source /data/install/functions
source /data/install/utils.fc
export SOURCE_LAN_IP="20.144.0.100"

export LAN_IP="10.10.90.132"

echo $SOURCE_LAN_IP > /data/install/.controller_ip
echo "LAN_IP=$SOURCE_LAN_IP" > /etc/blueking/env/local.env

sed -i "s/$LAN_IP/$SOURCE_LAN_IP/g" /data/install/install.config

sed -i "s/${LAN_IP}/${SOURCE_LAN_IP}/g" /etc/zookeeper/zoo.cfg /etc/consul.d/service/zk.json

sed -i "s/bind 20.144.0.100 10.10.90.132/bind 20.144.0.100/g" /etc/redis/default.conf

sed -i "s/  bindIp: 127.0.0.1, 20.144.0.100, 10.10.90.132/  bindIp: 127.0.0.1, 20.144.0.100/g" /etc/mongod.conf

sed -i "s/$LAN_IP/$SOURCE_LAN_IP/" /etc/consul.d/service/cmdb* /etc/consul.d/service/gse* /etc/consul.d/service/nodeman* /etc/consul.d/service/redis*

jq -r ".agentip=\"${SOURCE_LAN_IP}\"| .identityip=\"${SOURCE_LAN_IP}\" | .zkhost=\"${SOURCE_LAN_IP}:2181\"" /usr/local/gse/agent/etc/agent.conf > /tmp/agent.conf && \
mv -vf /tmp/agent.conf /usr/local/gse/agent/etc/agent.conf

sed -i "s/$LAN_IP/$SOURCE_LAN_IP/" /usr/local/gse/plugins/etc/bkmonitorbeat.conf

sed -i "s/$LAN_IP:9292/$SOURCE_LAN_IP:9292/" /data/weops/datainsight/docker-compose.yaml