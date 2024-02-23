[white]主机IP: [yellow]{{.IP}}

[white]操作系统信息: [yellow]{{.LinuxDistro}}

[white]内核版本: [yellow]{{.KernelVersion}}

[white]内存大小: [yellow]{{.Memory}}

[white]CPU信息: 
  [white]品牌: [yellow]{{.CPUDetail.Brand}}
  [white]核心数: [yellow]{{.CPUDetail.Cores}}

[white]网卡信息:
{{range $key, $value := .NetworkInterface}}  [white]设备名: [aqua]{{$key}}
    [white]IP: [yellow]{{$value}}
{{end}}
[white]磁盘信息:
{{range $key, $value := .Disk}}  [white]设备名: [aqua]{{$key}}
    [white]总大小: [yellow]{{$value.Total}}
    [white]已使用: [yellow]{{$value.Used}}
    [white]剩余空间: [yellow]{{$value.Free}}
    [white]使用率: [yellow]{{$value.Percent}}%
{{end}}
[white]DNS服务器:[yellow]
{{range .DNSServers}}  {{.}}    
{{end}}
[white]HOSTS记录:[yellow]
{{range .Hosts }} {{.}}
{{end}}