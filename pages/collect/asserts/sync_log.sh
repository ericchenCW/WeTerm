#!/bin/bash
set -euo pipefail

source /data/install/utils.fc

tmpdir=$(mktemp -d)

echo "[aqua]开始采集平台组件日志[white]"
IFS=',' read -ra IPs <<< "$ALL_IP_COMMA"
for IP in "${IPs[@]}"; do
  echo "[green]采集${IP} 组件日志[white]"
  rsync -avz --progress --no-links --exclude=*.sock --exclude=*.gz --exclude=*.log.* --exclude=*log-*.log* --exclude=*lock root@${IP}:/data/bkce/logs/ ${tmpdir}
  echo "[green]采集${IP}平台组件日志完成[white]"
done

echo "[aqua]开始采集应用日志[white]"
IFS=',' read -ra IPs <<< "$BK_APPO_IP_COMMA"
for IP in "${IPs[@]}"; do
  echo "[green]开始采集${IP} saas日志[white]"
  rsync -avz --progress --no-links --exclude=*.sock --exclude=*.gz --exclude=*.log* --exclude=*.lock root@${IP}:/data/bkce/paas_agent/apps/logs/ ${tmpdir}
  echo "[green]采集${IP}应用组件日志完成[white]"
done

echo "[aqua]开始打包日志[white]"
filename=weops-log-$(date +%Y%m%d).tar.gz
pushd ${tmpdir}
tar -czf /tmp/${filename} .
popd ${tmpdir}

echo "[aqua]开始分发日志[white]"
rsync -avzP /tmp/${filename} $BK_NGINX_IP:/data/weops/logs/${filename}

echo "[green]日志同步完成"
echo "可以从  [aqua]$BK_PAAS_PUBLIC_URL/logs/${filename}[green]  下载日志"
rm -rf ${tmpdir} 