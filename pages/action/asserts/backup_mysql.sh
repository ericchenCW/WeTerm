#!/bin/bash
source /data/install/utils.fc
ssh "$BK_MYSQL_IP" '
exec 2>&1
set -euo pipefail
source /data/install/utils.fc
TMP_BACKUP_DIR=$(mktemp -d)
BACKUP_DIR="/data/backup/$(date +%Y%m%d)"

if [[ -d "$BACKUP_DIR/mysql" ]]; then
    echo "[yellow]备份目录 $BACKUP_DIR/mysql 已存在[white]"
    echo "[yellow]清空备份目录 $BACKUP_DIR/mysql[white]"
    rm -vf "$BACKUP_DIR/mysql"/*
else
    mkdir -p "$BACKUP_DIR/mysql"
    echo "[green]备份目录 $BACKUP_DIR/mysql 创建成功[white]"
fi

export MYSQL_PWD="$BK_MYSQL_ADMIN_PASSWORD"

BACKUP_TIME=$(date +%Y%m%d%H%M%S)
MYSQLDUMP_STRUCT_CMD="mysqldump -uroot -hmysql-default.service.consul -d --skip-opt --create-options --single-transaction --max-allowed-packet=1G --net_buffer_length=10M -e -E -R -q --no-autocommit --hex-blob --all-databases | gzip -c > $TMP_BACKUP_DIR/struct_${BACKUP_TIME}.sql.gz"
MYSQLDUMP_CMD="mysqldump -uroot -hmysql-default.service.consul --skip-opt --create-options --single-transaction --max-allowed-packet=1G --net_buffer_length=10M -e -E -R -q --no-autocommit --hex-blob --all-databases | gzip -c > $TMP_BACKUP_DIR/data_${BACKUP_TIME}.sql.gz"

echo "[green]开始备份MySQL[white]"
eval $MYSQLDUMP_STRUCT_CMD && echo "[green]结构备份成功[white]" || echo "[red]结构备份失败[white]"
eval $MYSQLDUMP_CMD && echo "[green]数据备份成功[white]" || echo "[red]数据备份失败[white]"

echo "[green]开始将备份文件移动到备份目录[white]"
mv "$TMP_BACKUP_DIR"/* "$BACKUP_DIR/mysql/" && echo "[green]移动成功[white]" || echo "[red]移动失败[white]"

echo "[green]清理临时备份目录[white]"
rm -rf "$TMP_BACKUP_DIR" && echo "[green]清理成功[white]" || echo "[red]清理失败[white]"

echo "[green]备份完成[white]"
echo "[green]备份目录: $BACKUP_DIR/mysql[white]"
echo "[green]备份文件: [white]"
ls -lh "$BACKUP_DIR/mysql"
'
echo "[green]备份主机${BK_MYSQL_IP}[white]"