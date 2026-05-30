# procmon 部署

需要 root + Linux + cron。

## 1. 装 procmon

在装了 Go 的机器上编译：

```bash
cd cmd/procmon && make build-linux
# 产出 ./procmon-linux-amd64 (静态链接,跨发行版可跑)
```

推到目标机：

```bash
HOST=ctrl
scp procmon-linux-amd64 root@$HOST:/tmp/procmon.new
ssh root@$HOST 'install -m 0755 /tmp/procmon.new /usr/local/bin/procmon \
              && rm /tmp/procmon.new \
              && mkdir -p /var/log/procmon \
              && /usr/local/bin/procmon version'
```

多机循环：

```bash
for HOST in ctrl host-a host-b; do
    scp procmon-linux-amd64 root@$HOST:/tmp/procmon.new
    ssh root@$HOST 'install -m 0755 /tmp/procmon.new /usr/local/bin/procmon \
                  && rm /tmp/procmon.new \
                  && mkdir -p /var/log/procmon'
done
```

## 2. 配 cron

```bash
ssh root@$HOST 'cat > /etc/cron.d/procmon <<EOF
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
* * * * * root /usr/local/bin/procmon collect --data-dir /var/log/procmon
0 3 * * * root /usr/local/bin/procmon prune   --data-dir /var/log/procmon --keep-days 7
EOF
chmod 0644 /etc/cron.d/procmon'
```

`/etc/cron.d/` 下的文件 cron 会自动加载，不用重启服务。

等 2 分钟验证：

```bash
ssh root@$HOST 'jq -r .ts /var/log/procmon/*.jsonl | sort -u | tail -3'
```

相邻 ts 应该相差 60 秒。

## 3. 取数据

```bash
mkdir -p data
for HOST in ctrl host-a host-b; do
    rsync -av "root@$HOST:/var/log/procmon/" "data/"
done
```

`rsync` 增量同步，重复跑只传新增文件。
