#!/bin/bash
source /data/install/utils.fc
for ip in ${BK_RABBITMQ_IP[@]}; do
    echo "[yellow]${ip}[white]"
    ssh -T $ip <<"EOF"
#!/bin/bash
source /data/install/utils.fc
for i in bk_bknodeman bk_usermgr bk_bkmonitorv3 prod_monitorcenter_saas prod_bk_monitorv3 prod_bk_sops prod_ops-digital_saas prod_weops_saas job prod_bk_itsm;do
    queues=$(rabbitmqctl -p $i list_queues name 2>&1 | tail -n +4|grep celeryev)
    echo "[yellow]清空${i}的celeryev队列[white]"
    for queue in $queues; do
        rabbitmqctl -p $i purge_queue $queue
    done
done
EOF
done
echo "[green]RabbitMQ队列清理完成[white]"