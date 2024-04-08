package config

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"weterm/utils"

	"github.com/rs/zerolog/log"

	capi "github.com/hashicorp/consul/api"
)

//go:embed asserts/*
var fs embed.FS

var basePath = "/etc/consul-template/templates/"
var embedPath = "asserts/"
var reloadScript = "reload_openresty.sh"
var files = []string{
	"paas.conf",
	"cmdb.conf",
	"job.conf",
}

var consul_key = "weopscfg/backendAccess"

func Sync() bytes.Buffer {
	output := &bytes.Buffer{}
	hosts := strings.Split(os.Getenv("BK_NGINX_IP_COMMA"), ",")

	for _, host := range hosts {
		str := fmt.Sprintf("Sync to Host [aqua]%s[white]\n", host)
		io.WriteString(output, str)
		for _, file := range files {
			f, _ := fs.Open(embedPath + file)
			target := basePath + file
			log.Debug().Str("host", host).Str("file", file).Str("target", target).Msg("CopyFileBySSH")
			utils.CopyFileBySSH(host, f, target, output, file)
		}
	}
	script, err := fs.ReadFile(embedPath + reloadScript)
	if err != nil {
		str := fmt.Sprintf("[red]Read reload script failed[white]: [yellow]%s[white]\n", err)
		io.WriteString(output, str)
	}
	log.Logger.Debug().Str("script", string(script)).Msg("Reload Openresty")
	for _, host := range hosts {
		utils.RunSSH(host, string(script), "bash")
		str := fmt.Sprintf("Reload Openresty on Host [aqua]%s[white] [green]success[white]\n", host)
		io.WriteString(output, str)
	}

	return *output
}

func EnableBackendAccess(output io.Writer) {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	kv := client.KV()
	_, err = kv.Put(&capi.KVPair{Key: consul_key, Value: []byte("true")}, nil)
	if err != nil {
		str := fmt.Sprintf("EnableBackendAccess [red]failed[white]: [yellow]%s[white]\n", err)
		io.WriteString(output, str)
	}
	io.WriteString(output, "EnableBackendAccess [green]success[white]\n")
	hosts := strings.Split(os.Getenv("BK_NGINX_IP_COMMA"), ",")
	script, err := fs.ReadFile(embedPath + reloadScript)
	if err != nil {
		str := fmt.Sprintf("[red]Read reload script failed[white]: [yellow]%s[white]\n", err)
		io.WriteString(output, str)
	}
	log.Logger.Debug().Str("script", string(script)).Msg("Reload Openresty")
	for _, host := range hosts {
		utils.RunSSH(host, string(script), "bash")
		str := fmt.Sprintf("Reload Openresty on Host [aqua]%s[white] [green]success[white]\n", host)
		io.WriteString(output, str)
	}
}

func DisableBackendAccess(output io.Writer) {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	kv := client.KV()
	_, err = kv.Delete(consul_key, nil)
	if err != nil {
		str := fmt.Sprintf("DisableBackendAccess [red]failed[white]: [yellow]%s[white]\n", err)
		io.WriteString(output, str)
	}
	io.WriteString(output, "DisableBackendAccess [green]success[white]\n")
	hosts := strings.Split(os.Getenv("BK_NGINX_IP_COMMA"), ",")
	script, err := fs.ReadFile(embedPath + reloadScript)
	if err != nil {
		str := fmt.Sprintf("[red]Read reload script failed[white]: [yellow]%s[white]\n", err)
		io.WriteString(output, str)
	}
	log.Logger.Debug().Str("script", string(script)).Msg("Reload Openresty")
	for _, host := range hosts {
		utils.RunSSH(host, string(script), "bash")
		str := fmt.Sprintf("Reload Openresty on Host [aqua]%s[white] [green]success[white]\n", host)
		io.WriteString(output, str)
	}
}
