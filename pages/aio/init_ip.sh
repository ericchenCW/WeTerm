#!/bin/bash
# 此脚本用于weops一体机初始化时修改服务ip并执行服务初始化
export LC_ALL=en_US.UTF-8
set -eo pipefail
source /root/.bkrc
source /data/install/functions
source /data/install/utils.fc
export SOURCE_LAN_IP=20.144.0.100

if [[ -n "$SSH_CONNECTION" ]]; then
    ssh_info=($SSH_CONNECTION)
    LAN_IP=${ssh_info[2]}
    echo "auto guess current LAN_IP is $LAN_IP"
else
    echo "can't auto-get LAN_IP for this host"
    exit 1
fi

echo "[green]WEOPS_LAN_IP ${LAN_IP}[white]"
echo $LAN_IP > /data/install/.controller_ip
echo "LAN_IP=$LAN_IP" > /etc/blueking/env/local.env

echo "[green]render install.config[white]"
sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /data/install/install.config

echo "[green]add resolve[white]"
if grep -w 127.0.0.1 /etc/resolv.conf > /dev/null 2>&1;then
    echo "nameserver 127.0.0.1 already exist"
else
    sed -i "1i nameserver 127.0.0.1" /etc/resolv.conf
fi

echo "[green]restart third compoents[white]"
echo consul mysql redis rabbitmq mongodb zk kafka es7 influxdb | xargs -n 1 /data/install/bkcli restart

echo "[green]restart blueking compoents[white]"
echo license bkiam usermgr paas appo cmdb bknodeman bkmonitorv3 | xargs -n 1 /data/install/bkcli restart

sleep 1m

echo "[green]reinstall paas[white]"
bash ./configure_ssh_without_pass

cd /data/install && ./bk_install common && ./health_check/check_bk_controller.sh && ./bk_install paas && ./bk_install app_mgr \
&& ./bk_install cmdb && ./bk_install job \
&& ./bk_install bknodeman \
&& ./bk_install saas-o bk_iam && ./bk_install saas-o bk_user_manage

echo "[green]replace zk ip[white]"
sed -i "s/${SOURCE_LAN_IP}/${LAN_IP}/g" /etc/zookeeper/zoo.cfg /etc/consul.d/service/zk.json
systemctl restart zookeeper

echo "[green]replace redis ip[white]"
if grep -w ${LAN_IP} /etc/redis/default.conf > /dev/null 2>&1;then
    echo "${LAN_IP} already in /etc/redis/default.conf"
else 
    sed -i "s/bind 20.144.0.100/bind 20.144.0.100 ${LAN_IP}/g" /etc/redis/default.conf
fi

systemctl restart redis@default

echo "[green]replace mongodb ip[white]"
if grep -w ${LAN_IP} /etc/mongod.conf > /dev/null 2>&1;then
    echo "${LAN_IP} already in /etc/mongod.conf"
else 
    sed -i "s/  bindIp: 127.0.0.1, 20.144.0.100/  bindIp: 127.0.0.1, 20.144.0.100, ${LAN_IP}/g" /etc/mongod.conf
fi

systemctl restart mongod

echo "[green]render consul[white]"
sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /etc/consul.d/service/cmdb* /etc/consul.d/service/gse* /etc/consul.d/service/nodeman* /etc/consul.d/service/redis*

echo "[green]restart consul[white]"
/data/install/bkcli restart consul

sleep 30

echo "[green]restart kafka[white]"
/data/install/bkcli restart kafka

echo "[green]restart gse[white]"
/data/install/bkcli render gse && /data/install/bkcli restart gse

echo "[green]add resolve[white]"
if grep -w 127.0.0.1 /etc/resolv.conf > /dev/null 2>&1;then
    echo "nameserver 127.0.0.1 already exist"
else
    sed -i "1i nameserver 127.0.0.1" /etc/resolv.conf
fi

echo "[green]restart nodeman[white]"
/data/install/bkcli render bknodeman && /data/install/bkcli restart bknodeman

echo "[green]update agent ip[white]"
jq -r ".agentip=\"${LAN_IP}\"| .identityip=\"${LAN_IP}\" | .zkhost=\"${LAN_IP}:2181\"" /usr/local/gse/agent/etc/agent.conf > /tmp/agent.conf && \
mv -vf /tmp/agent.conf /usr/local/gse/agent/etc/agent.conf

echo "[green]restart agent[white]"
pushd /usr/local/gse/agent/bin
./gsectl restart
popd

sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /usr/local/gse/plugins/etc/bkmonitorbeat.conf
pushd /usr/local/gse/plugins/bin
./restart.sh bkmonitorbeat
popd

echo "[green]replace default access point[white]"
pushd /data/bkce/bknodeman/nodeman
source bin/environ.sh
/data/bkce/.envs/bknodeman-nodeman/bin/python manage.py shell <<EOF
from apps.node_man.models import AccessPoint
target_ip = "${LAN_IP}"
print(f"target_ip is {target_ip}")
try:
    de = AccessPoint.get_default_ap()
    de.zk_hosts[0]["zk_ip"] = target_ip
    de.btfileserver[0]["inner_ip"] = target_ip
    de.btfileserver[0]["outer_ip"] = target_ip
    de.dataserver[0]["inner_ip"] = target_ip
    de.dataserver[0]["outer_ip"] = target_ip
    de.taskserver[0]["inner_ip"] = target_ip
    de.taskserver[0]["outer_ip"] = target_ip
    de.package_inner_url = f"http://{target_ip}:80/download"
    de.package_outer_url = f"http://{target_ip}:80/download"
    de.save()
    print(f"update_success")
except Exception as e:
    print(f"update fail! error message{e}")
EOF
popd
/data/install/bkcli restart bknodeman

echo "[green]restart job[white]"
/data/install/bkcli render job && /data/install/bkcli restart job && systemctl restart bk-job-execute


echo "[green]update weops_saas environment values[white]"
mysql --login-path=mysql-default --database=open_paas <<EOF
update paas_app_envvars set value="${LAN_IP}" where app_code="weops_saas" and \`name\`="BKAPP_SOURCE_IP";
update paas_app_envvars set value="${LAN_IP}:9292" where app_code="weops_saas" and \`name\`="BKAPP_KAFKA_HOST";
EOF

echo "[green]deploy weops[white]"
find /data/src/official_saas/weops_saas* | sort -r | head -n 1 | xargs -I{} /opt/py36/bin/python /data/install/bin/saas.py -e appo -n weops_saas -k {} -f /data/install/bin/04-final/paas.env

echo "[green]up docker services[white]"
docker start $(docker ps -aq)

echo "[green]update datainsight kafka lan ip[white]"
sed -i "s/$SOURCE_LAN_IP:9292/$LAN_IP:9292/" /data/weops/datainsight/docker-compose.yaml
docker-compose -f /data/weops/datainsight/docker-compose.yaml up -d

echo "[green]init topo[white]"
# 清理存量拓扑记录
mongo -u $BK_CMDB_MONGODB_USERNAME -p $BK_CMDB_MONGODB_PASSWORD mongodb://$LAN_IP:$BK_CMDB_MONGODB_PORT/cmdb --authenticationDatabase cmdb << "EOF"
db.cc_ServiceTemplate.remove({"bk_biz_id":2})
db.cc_ProcessTemplate.remove({"bk_biz_id":2})
db.cc_SetTemplate.remove({"bk_biz_id":2})
db.cc_SetBase.remove({$and:[{"bk_set_id":{$gt:2}},{"bk_biz_id":{$eq:2}}]},{"bk_set_id":1,"bk_set_name":1,"bk_biz_id":1,"bk_biz_name":1});
EOF

# 重新初始化拓扑
/data/install/bkcli initdata topo

# 重启监控平台
/data/install/bkcli restart bkmonitorv3


echo ""
echo "如果以上步骤执行没有报错, 说明WeOps一体机初始化已完成, 现在可以通过 [green]${BK_PAAS_PUBLIC_URL}[white] 进行访问"
echo "host记录: [green] ${LAN_IP} paas.${BK_DOMAIN}[white]"
echo "登陆用户名(login user): [green] ${BK_PAAS_ADMIN_USERNAME}[white]"
echo "登陆密码(login password): [green] ${BK_PAAS_ADMIN_PASSWORD} [white]"
echo