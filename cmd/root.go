package cmd

import (
	colorable "github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"weterm/ui"
)

var (
	rootCmd = &cobra.Command{
		Use:  "WeTrem",
		RunE: run,
	}
	out = colorable.NewColorableStdout()
)

func run(cmd *cobra.Command, args []string) error {
	mod := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	file, err := os.OpenFile("./weterm.log", mod, 777)
	if err != nil {
		return err
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: file})
	log.Info().Msg("🐶 WeTerm starting up...")
	bootstrap := ui.NewBootStrap()
	bootstrap.Start()
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
