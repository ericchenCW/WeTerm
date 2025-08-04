#!/usr/bin/env bash
# shellcheck disable=SC1090     # utils.fc 在其他路径
set -euo pipefail

source ~/.bkrc
source /data/install/utils.fc

: "${BK_VAULT_INIT_IP:=}"
: "${BK_APPT_IP_COMMA:=}"
: "${AIO:=}"

# ─── 判定 Vault 主机 IP ─────────────────────────────
if [[ -n "$BK_VAULT_INIT_IP" ]]; then
    VAULT_IP="$BK_VAULT_INIT_IP"

elif [[ -z "$BK_APPT_IP_COMMA" ]]; then
    echo_color yellow "未找到应用主机"
    if [[ -z "$AIO" ]]; then
        echo_color yellow "非一体机环境，无法定位 Vault 主机"
        exit 1
    else
        echo_color yellow "一体机环境，使用本机作为 Vault 主机"
        BK_APPT_IP_COMMA="20.144.0.100"
        VAULT_IP="$BK_APPT_IP_COMMA"
    fi

else
    VAULT_IP="$BK_APPT_IP"   # 如果你想用列表里的第一个 IP，可写 ${BK_APPT_IP_COMMA%%,*}
fi

# ─── 解锁 Vault ────────────────────────────────────
for ip in ${BK_APPT_IP_COMMA//,/ }; do           # 逗号分隔 → 空格分隔
    echo "[yellow]解锁 Vault 主机 $ip[white]"
    ssh -T "$ip" <<"EOF"
source /data/install/utils.fc
set -euo pipefail
if docker exec vault sh -c "export VAULT_ADDR=http://127.0.0.1:8200 && vault operator unseal ${VAULT_UNSEAL_CODE}"; then
    echo "[green]解锁成功[white]"
else
    echo "[red]解锁失败[white]"
fi
EOF
done