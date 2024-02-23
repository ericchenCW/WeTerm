#!/bin/env python
import psutil
import time
import json
from datetime import datetime

def bytes_to_readable(bytes):
    """将字节转换为可读的字符串(kB, MB, GB, TB)"""
    for unit in ["","K","M","G","T"]:
        if bytes < 1024.:
            return "%.1f%sB" % (bytes, unit)
        bytes /= 1024.

disk_partitions = psutil.disk_partitions()

# # 获取磁盘IO和网络IO计数
# disk_io_begin = psutil.disk_io_counters()
# net_io_begin = psutil.net_io_counters()

# sleep_time = 1  # 单位：秒
# time.sleep(sleep_time)

# disk_io_end = psutil.disk_io_counters()
# net_io_end = psutil.net_io_counters()

# disk_io_delta = {
#     "IOPS": disk_io_end.read_count + disk_io_end.write_count - disk_io_begin.read_count - disk_io_begin.write_count,
#     "throughput": bytes_to_readable(disk_io_end.read_bytes + disk_io_end.write_bytes - disk_io_begin.read_bytes - disk_io_begin.write_bytes)
# }

# net_io_delta = {
#     "sent": bytes_to_readable(net_io_end.bytes_sent - net_io_begin.bytes_sent),
#     "received": bytes_to_readable(net_io_end.bytes_recv - net_io_begin.bytes_recv)
# }

# 获取CPU使用率 和 负载
cpu_percent = psutil.cpu_percent(interval=0)
cpu_load = psutil.getloadavg()

# 获取内存使用情况
memory = psutil.virtual_memory()
memory_dict = {
    "total": bytes_to_readable(memory.total),
    "available": bytes_to_readable(memory.available),
    "percent": memory.percent,
    "used": bytes_to_readable(memory.used),
    "free": bytes_to_readable(memory.free)
}

# 获取swap内存使用情况
swap = psutil.swap_memory()
swap_dict = {
    "total": bytes_to_readable(swap.total),
    "used": bytes_to_readable(swap.used),
    "free": bytes_to_readable(swap.free),
    "percent": swap.percent
}

# 获取所有磁盘的使用情况
disk_usage = {}
for partition in disk_partitions:
    usage = psutil.disk_usage(partition.mountpoint)
    disk_usage[partition.device] = {
        "total": bytes_to_readable(usage.total),
        "used": bytes_to_readable(usage.used),
        "free": bytes_to_readable(usage.free),
        "percent": usage.percent
    }

# 整合数据并输出为JSON格式
data = {
    "cpu_percent": cpu_percent,
    "cpu_load": cpu_load,
    "memory": memory_dict,
    "swap": swap_dict,
    "disk": disk_usage,
    "datetime": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    # "disk_io_delta": disk_io_delta,
    # "network_io_delta": net_io_delta,
}
json_data = json.dumps(data, indent=4)

print(json_data)