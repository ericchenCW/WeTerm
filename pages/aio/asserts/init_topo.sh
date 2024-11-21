#!/bin/bash
export LC_ALL=en_US.UTF-8
set -eo pipefail
source /root/.bkrc
source /data/install/functions
source /data/install/utils.fc
exec 2>&1

if [[ -d /data/bkee ]];then
    BASE_PATH=/data/bkee
else
    BASE_PATH=/data/bkce
fi

step_echo() {
    echo -e "[green]$1[white]"
}

info_echo() {
    echo -e "[blue]$1[white]"
}
step_echo "init topo"
mongo -u $BK_CMDB_MONGODB_USERNAME -p $BK_CMDB_MONGODB_PASSWORD mongodb://$LAN_IP:$BK_CMDB_MONGODB_PORT/cmdb --authenticationDatabase cmdb << "EOF"
db.cc_ServiceTemplate.remove({"bk_biz_id":2})
db.cc_ProcessTemplate.remove({"bk_biz_id":2})
db.cc_SetTemplate.remove({"bk_biz_id":2})
db.cc_SetBase.remove({$and:[{"bk_set_id":{$gt:2}},{"bk_biz_id":{$eq:2}}]},{"bk_set_id":1,"bk_set_name":1,"bk_biz_id":1,"bk_biz_name":1});
EOF
# step_echo "restart cmdb"
# /data/install/bkcli restart cmdb
# i=1
# until /data/install/bkcli check cmdb 2>&1 >/dev/null;do
#     info_echo "waiting cmdb ready $i"
#     i=$i+1
#     sleep 10
# done
# sleep 10
step_echo "start init topo"
# 重新初始化拓扑
/data/install/bkcli initdata topo
step_echo "初始化成功"