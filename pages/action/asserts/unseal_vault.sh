source ~/.bkrc
source /data/install/utils.fc

if [[ -z $BK_APPT_IP_COMMA ]]; then
    echo "[yellow]未找到应用主机[white]"
    if [[ -z $AIO ]]; then
        echo "[yellow]非一体机环境,无法定位vault主机[white]"
        exit 1
    else
        echo "[yellow]一体机环境,使用本机作为vault主机[white]"
        BK_APPT_IP_COMMA=20.144.0.100
    fi
fi

for ip in $BK_APPT_IP_COMMA;do
    echo "[green]解锁vault主机 [yellow]${ip}[white]"
    ssh $ip 'source /data/install/utils.fc;exec 2>&1;docker exec vault sh -c "export VAULT_ADDR=http://127.0.0.1:8200 && vault operator unseal ${VAULT_UNSEAL_CODE}" && echo "[green]解锁成功" || echo "[red]解锁失败"'
done