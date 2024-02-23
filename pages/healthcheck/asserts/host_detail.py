#!/bin/env python
import json
import os
import platform
import psutil
import socket


def get_linux_distro():
    try:
        return " ".join(platform.linux_distribution())
    except:
        return "Unknown"

def get_linux_kernel_version():
    return os.uname()[2]

def get_hostname():
    return socket.gethostname()

def get_all_network_interfaces():
    interfaces_ips = {}
    for interface, snics in psutil.net_if_addrs().items():
        for snic in snics:
            if snic.family == socket.AF_INET:
                interfaces_ips[interface] = snic.address
    return interfaces_ips 

def get_dns_servers():
    dns_servers = []
    resolv_conf = "/run/systemd/resolve/resolv.conf" if os.path.exists("/run/systemd/resolve/resolv.conf") else "/etc/resolv.conf"
    with open(resolv_conf) as f:
        for line in f:
            if line.startswith("nameserver"):
                dns_servers.append(line.split()[1])
    return dns_servers

def get_hosts():
    hosts = []
    hosts_file = "/etc/hosts"
    with open(hosts_file) as f:
        for line in f:
            hosts.append(line.strip("\n"))
    return hosts

def get_cpu_info():
    cpu_info = {}
    with open("/proc/cpuinfo") as f:
        info = f.read()
    model_name_line = [line for line in info.split("\n") if "model name" in line]
    if model_name_line:
        cpu_info["brand_model"] = model_name_line[0].split(": ")[1]
    cpu_info["cores"] = str(psutil.cpu_count())
    return cpu_info

def get_memory_info():
    memory_info = str(round(psutil.virtual_memory().total / (1024**3), 2)) + "GB"
    return memory_info

def get_swap_info():
    swap_info = str(round(psutil.swap_memory().total / (1024**3), 2)) + "GB"
    return swap_info

def get_disk_usages():
    disk_usages = {}
    for partition in psutil.disk_partitions():
        usage = psutil.disk_usage(partition.mountpoint)
        disk_usages[partition.device] = {"total": str(round(usage.total/(1024**3), 2)) + "GB", # GB
                                         "used": str(round(usage.used/(1024**3), 2)) + "GB", # GB
                                         "free": str(round(usage.free/(1024**3), 2)) + "GB", # GB
                                         "percent": usage.percent}
    return disk_usages

def collect_system_info():
    system_info = {
        "linux_distro": get_linux_distro(),
        "kernel_version": get_linux_kernel_version(),
        "hostname": get_hostname(),
        "hosts": get_hosts(),
        "network_interfaces": get_all_network_interfaces(),
        "dns_servers": get_dns_servers(),
        "cpu_info": get_cpu_info(),
        "memory_info": get_memory_info(),
        "swap_info": get_swap_info(),
        "disk_usages": get_disk_usages(),
    }
    return system_info

def main():
    data = collect_system_info()
    print(json.dumps(data, indent=4))

if __name__ == "__main__":
    main()