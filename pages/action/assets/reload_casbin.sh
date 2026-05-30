source /data/install/utils.fc
echo "[yellow]从WeOps重新同步casbin_mesh规则,执行主机${BK_APPO_IP}[white]"
ssh $BK_APPO_IP 'source /data/install/utils.fc
exec 2>&1
set -euo pipefail
echo "[yellow]停止casbin-mesh服务[white]"
docker stop casbin_mesh && echo "[green]停止成功[white]" || echo "[red]停止失败[white]"
echo "[yellow]清空现有wal[white]"
rm -rvf /data/weops/casbin-mesh/data && echo "[green]清空成功[white]" || echo "[red]清空失败[white]"
echo "[yellow]启动casbin-mesh服务[white]"
docker start casbin_mesh && echo "[green]启动成功[white]" || echo "[red]启动失败[white]"
echo "[yellow]从WeOps重新同步casbin_mesh规则[white]"
docker exec -i $(docker ps -aq -f name=weops_saas*) bash -c "export BK_FILE_PATH=/data/app/code/conf/saas_priv.txt;cd /data/app/code;python manage.py reload_casbin_policy  --delete;" && echo "[green]同步成功[white]" || echo "[red]同步失败[white]"
'