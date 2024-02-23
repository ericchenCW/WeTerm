package utils

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func RunSSH(host string, script string) []byte {
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
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		log.Error().Err(err).Msg("SSH Dial failed")
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		log.Error().Err(err).Msg("SSH Session failed")
	}
	defer session.Close()

	output, err := session.Output("python -c '" + script + "'")
	if err != nil {
		log.Error().Err(err).Msg("SSH run script failed")
	}

	return output
}
