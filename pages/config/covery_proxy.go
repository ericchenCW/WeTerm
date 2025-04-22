package config

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
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

var backend_consul_key = "weopscfg/backendAccess"
var nginx_status_consul_key = "weopscfg/poc/enableNginxStatus"

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

func setConsulKey(key string, value string) bool {
	config := capi.DefaultConfig()
	client, err := capi.NewClient(config)
	if err != nil {
		panic(err)
	}
	kv := client.KV()
	_, err = kv.Put(&capi.KVPair{Key: key, Value: []byte(value)}, nil)
	if err != nil {
		log.Logger.Err(err).Msg("SetConsulKey")
		return false
	}
	return true
}

func reloadOpenresty() bool {
	hosts := strings.Split(os.Getenv("BK_NGINX_IP_COMMA"), ",")
	script, err := fs.ReadFile(embedPath + reloadScript)
	if err != nil {
		log.Logger.Err(err).Msg("ReloadOpenresty")
		return false
	}
	for _, host := range hosts {
		log.Logger.Info().Str("host", host).Msg("ReloadOpenresty")
		utils.RunSSH(host, string(script), "bash")
		log.Logger.Info().Str("host", host).Msg("ReloadOpenresty success")
	}
	return true
}

func getNginxStatus() string {
	// send http request to nginx status page, return body
	url := os.Getenv("BK_PAAS_PUBLIC_URL") + "/nginx_status"
	resp, err := http.Get(url)
	if err != nil {
		log.Logger.Err(err).Msg("GetNginxStatus")
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Err(err).Msg("ReadNginxStatus")
		return ""
	}
	return string(body)
}

func setNginxConfig(output io.Writer, key string, value string) {
	set_r := setConsulKey(key, "true")
	if set_r == false {
		str := fmt.Sprintf("EnableBackendAccess [red]failed[white]: [yellow]see more info in weterm.log[white]\n")
		io.WriteString(output, str)
		return
	}
	io.WriteString(output, "EnableBackendAccess [green]success[white]\n")
	reload_r := reloadOpenresty()
	if reload_r == false {
		str := fmt.Sprintf("[red]Read reload script failed[white]: [yellow]see more info in weterm.log[white]\n")
		io.WriteString(output, str)
		return
	}
	str := fmt.Sprintf("Reload Openresty [green]success[white]\n")
	io.WriteString(output, str)
}

func EnableBackendAccess(output io.Writer) {
	setNginxConfig(output, backend_consul_key, "true")
}

func DisableBackendAccess(output io.Writer) {
	setNginxConfig(output, backend_consul_key, "false")
}

func EnableNginxStatus(output io.Writer) {
	setNginxConfig(output, nginx_status_consul_key, "true")
	output.Write([]byte("Nginx Status Page:\n"))
	output.Write([]byte(getNginxStatus()))
}

func DisableNginxStatus(output io.Writer) {
	setNginxConfig(output, nginx_status_consul_key, "false")
}
