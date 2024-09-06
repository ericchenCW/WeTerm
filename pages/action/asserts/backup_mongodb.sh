#!/bin/bash
source /data/install/utils.fc
ssh $BK_MONGODB_IP '
exec 2>&1
set -euo pipefail
source /data/install/utils.fc
TMP_BACKUP_DIR=$(mktemp -d)
BACKUP_DIR="/data/backup/$(date +%Y%m%d)"

if [[ -d $BACKUP_DIR/mongodb ]]; then
    echo "[yellow]备份目录已存在[white]"
else
    mkdir -p $BACKUP_DIR/mongodb
    echo "[green]备份目录创建成功[white]"
fi

MONGO_DUMP_CMD="mongodump --host mongodb.service.consul -u root -p $BK_MONGODB_ADMIN_PASSWORD --oplog --gzip --out $TMP_BACKUP_DIR"

echo "[green]开始备份MongoDB[white]"
$MONGO_DUMP_CMD && echo "[green]备份成功[white]" || echo "[red]备份失败[white]"

echo "[green]清空备份目录[white]"
rm -rvf $BACKUP_DIR/mongodb/* && echo "[green]清空成功[white]" || echo "[red]清空失败[white]"

echo "[green]开始打包备份文件[white]"
pushd $TMP_BACKUP_DIR
tar -zcf $BACKUP_DIR/mongodb/$(date +%Y%m%d%H%M%S).tar.gz . && echo "[green]打包成功[white]" || echo "[red]打包失败[white]"
popd

echo "[green]清理临时备份目录[white]"
rm -rf $TMP_BACKUP_DIR && echo "[green]清理成功[white]" || echo "[red]清理失败[white]"

echo "[green]备份完成[white]"
echo "[green]备份目录: $BACKUP_DIR/mongodb[white]"
echo "[green]备份文件: [white]"
ls -lh $BACKUP_DIR/mongodb
'
echo "[green]备份主机${BK_MONGODB_IP}[white]"