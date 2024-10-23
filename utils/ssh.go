package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/kevinburke/ssh_config"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

type ProgressWriter struct {
	total          int64     // 已经传输的字节数
	length         int64     // 总字节数
	realTimeOutput io.Writer // 实时输出
	name           string
	target         string
}

func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.total += int64(n)
	percentage := 100 * float64(pw.total) / float64(pw.length)
	str := fmt.Sprintf("filename: [green]%s[white] target: [green]%s[white] %.2f%%\t%d/%d bytes transferred \n", pw.name, pw.target, percentage, pw.total, pw.length)
	pw.realTimeOutput.Write([]byte(str))
	return n, nil
}

func CopyFileBySSH(host string, src fs.File, dst string, output io.Writer, filename string) {
	privateKeyPath := os.Getenv("HOME") + "/.ssh/id_rsa"
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Error().Err(err).Msg("ReadPrivateKey failed")
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Error().Err(err).Msg("ParsePrivateKey failed")
	}
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	ssh_config.Get(host, "User")
	port := ssh_config.Get(host, "Port")
	if port == "" {
		port = "22"
	}
	addr := host + ":" + port
	log.Debug().Str("addr", addr).Msg("Start SSH Dial")
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Error().Err(err).Msg("SSH Dial failed")
	}
	defer client.Close()
	s, err := sftp.NewClient(client)
	if err != nil {
		log.Error().Err(err).Msg("SFTP NewClient failed")
	}
	defer s.Close()
	fileInfo, _ := src.Stat()
	fileSize := fileInfo.Size()
	progressWriter := &ProgressWriter{length: fileSize, realTimeOutput: output, name: filename, target: dst}

	dstFile, err := s.Create(dst)
	if err != nil {
		log.Error().Str("dst", dst).Str("host", host).Str("filename", filename).Err(err).Msg("SFTP FILE Create failed")
	}
	defer dstFile.Close()
	log.Debug().Any("src", src).Any("dst", dstFile).Any("writer", progressWriter).Msg("Start copy file")
	_, err = io.Copy(dstFile, io.TeeReader(src, progressWriter))
	if err != nil {
		log.Error().Err(err).Msg("Copy file failed")
	}
}

func RunSSH(host string, script string, shell string) []byte {
	ssh_config.Get(host, "User")
	port := ssh_config.Get(host, "Port")
	if port == "" {
		port = "22"
	}
	addr := host + ":" + port
	privateKeyPath := os.Getenv("HOME") + "/.ssh/id_rsa"
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Error().Err(err).Msg("ReadPrivateKey failed")
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Error().Err(err).Msg("ParsePrivateKey failed")
	}
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Error().Err(err).Msg("SSH Dial failed")
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		log.Error().Err(err).Msg("SSH Session failed")
	}
	defer session.Close()

	output, err := session.Output(shell + " -c '" + script + "'")
	if err != nil {
		log.Error().Err(err).Msg("SSH run script failed")
	}

	return output
}
