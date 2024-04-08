#!/bin/bash
# 此脚本用于weops一体机初始化时修改服务ip并执行服务初始化
export LC_ALL=en_US.UTF-8
set -eo pipefail
source /root/.bkrc
source /data/install/functions
source /data/install/utils.fc
export SOURCE_LAN_IP=20.144.0.100

step_echo() {
    echo -e "[green]$1[white]"
}

info_echo() {
    echo -e "[blue]$1[white]"
}

if [[ -n "$SSH_CONNECTION" ]]; then
    ssh_info=($SSH_CONNECTION)
    LAN_IP=${ssh_info[2]}
    info_echo "auto guess current LAN_IP is $LAN_IP"
    bash /data/install/configure_ssh_without_pass
    setcap 'cap_net_bind_service=+ep' /usr/bin/consul
else
    info_echo "can't auto-get LAN_IP for this host"
    exit 1
fi

step_echo "WEOPS_LAN_IP ${LAN_IP}"
echo $LAN_IP > /data/install/.controller_ip
echo "LAN_IP=$LAN_IP" > /etc/blueking/env/local.env

step_echo "render install.config"
sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /data/install/install.config

step_echo "add resolve"
if grep -w 127.0.0.1 /etc/resolv.conf > /dev/null 2>&1;then
    info_echo "nameserver 127.0.0.1 already exist"
else
    sed -i "1i nameserver 127.0.0.1" /etc/resolv.conf
fi

step_echo "restart third compoents"
echo consul mysql redis rabbitmq mongodb zk kafka es7 influxdb | xargs -n 1 /data/install/bkcli restart

step_echo "restart blueking compoents"
echo license bkiam usermgr paas appo cmdb bknodeman bkmonitorv3 | xargs -n 1 /data/install/bkcli restart

sleep 1m

step_echo "reinstall paas"

cd /data/install && ./bk_install common && ./health_check/check_bk_controller.sh && ./bk_install paas && ./bk_install app_mgr \
&& ./bk_install cmdb && ./bk_install job \
&& ./bk_install bknodeman \
&& ./bk_install saas-o bk_iam && ./bk_install saas-o bk_user_manage

step_echo "replace zk ip"
sed -i "s/${SOURCE_LAN_IP}/${LAN_IP}/g" /etc/zookeeper/zoo.cfg /etc/consul.d/service/zk.json
systemctl restart zookeeper

step_echo "replace redis ip"
if grep -w ${LAN_IP} /etc/redis/default.conf > /dev/null 2>&1;then
    echo "${LAN_IP} already in /etc/redis/default.conf"
else 
    sed -i "s/bind 20.144.0.100/bind 20.144.0.100 ${LAN_IP}/g" /etc/redis/default.conf
fi

systemctl restart redis@default

step_echo "replace mongodb ip"
if grep -w ${LAN_IP} /etc/mongod.conf > /dev/null 2>&1;then
    echo "${LAN_IP} already in /etc/mongod.conf"
else 
    sed -i "s/  bindIp: 127.0.0.1, 20.144.0.100/  bindIp: 127.0.0.1, 20.144.0.100, ${LAN_IP}/g" /etc/mongod.conf
fi

systemctl restart mongod

step_echo "render consul"
sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /etc/consul.d/service/cmdb* /etc/consul.d/service/gse* /etc/consul.d/service/nodeman* /etc/consul.d/service/redis*

step_echo "restart consul"
/data/install/bkcli restart consul

sleep 30

step_echo "restart kafka"
/data/install/bkcli restart kafka

step_echo "restart gse"
/data/install/bkcli render gse && /data/install/bkcli restart gse

step_echo "add resolve"
if grep -w 127.0.0.1 /etc/resolv.conf > /dev/null 2>&1;then
    echo "nameserver 127.0.0.1 already exist"
else
    sed -i "1i nameserver 127.0.0.1" /etc/resolv.conf
fi

step_echo "restart nodeman"
/data/install/bkcli render bknodeman && /data/install/bkcli restart bknodeman

step_echo "update agent ip"
jq -r ".agentip=\"${LAN_IP}\"| .identityip=\"${LAN_IP}\" | .zkhost=\"${LAN_IP}:2181\"" /usr/local/gse/agent/etc/agent.conf > /tmp/agent.conf && \
mv -vf /tmp/agent.conf /usr/local/gse/agent/etc/agent.conf

step_echo "restart agent"
pushd /usr/local/gse/agent/bin
./gsectl restart
popd

sed -i "s/$SOURCE_LAN_IP/$LAN_IP/" /usr/local/gse/plugins/etc/bkmonitorbeat.conf
pushd /usr/local/gse/plugins/bin
./restart.sh bkmonitorbeat
popd

step_echo "replace default access point"
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

step_echo "restart job"
/data/install/bkcli render job && /data/install/bkcli restart job && systemctl restart bk-job-execute


step_echo "update weops_saas environment values"
mysql --login-path=mysql-default --database=open_paas <<EOF
update paas_app_envvars set value="${LAN_IP}" where app_code="weops_saas" and \`name\`="BKAPP_SOURCE_IP";
update paas_app_envvars set value="${LAN_IP}:9292" where app_code="weops_saas" and \`name\`="BKAPP_KAFKA_HOST";
update paas_app_envvars set value="http://${LAN_IP}:9001" where app_code="weops_saas" and \`name\`="BKAPP_CMDB_HOST";
update paas_app_envvars set value="http://${LAN_IP}:9090" where app_code="weops_saas" and \`name\`="BKAPP_GRAYLOG_URL";
EOF

step_echo "deploy saas"
for i in monitorcenter_saas cw_uac_saas bk_itsm weops_saas;do 
    /data/install/bk_install saas-o ${i} 2>&1
done

step_echo "up docker services"
docker start $(docker ps -aq)

step_echo "update datainsight kafka lan ip"
sed -i "s/$SOURCE_LAN_IP:9292/$LAN_IP:9292/" /data/weops/datainsight/docker-compose.yaml
docker-compose -f /data/weops/datainsight/docker-compose.yaml up -d

step_echo "unseal vault"
docker exec vault sh -c "export VAULT_ADDR=http://127.0.0.1:8200 && vault operator unseal ${VAULT_UNSEAL_CODE}"

step_echo "reset casbin-mesh"
docker stop casbin_mesh
rm -rvf /data/weops/casbin-mesh/*
docker restart casbin_mesh

step_echo "init topo"
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