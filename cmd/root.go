package cmd

import (
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
)

var (
	rootCmd = &cobra.Command{
		Use:  "WeTerm",
		RunE: run,
	}
	out = colorable.NewColorableStdout()
)

func run(cmd *cobra.Command, args []string) error {
	err := setupLogInstance()
	if err != nil {
		return err
	}

	bootstrap := NewApp()
	bootstrap.Start()
	return nil
}

func setupLogInstance() error {
	mod := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	file, err := os.OpenFile("./weterm.log", mod, 0777)
	if err != nil {
		return err
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: file})
	log.Info().Msg("🐶 WeTerm starting up...")
	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
