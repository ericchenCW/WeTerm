package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "weterm",
		Short: "weterm - A terminal-based operations tool",
		Long:  `weterm is a terminal-based operations tool for managing infrastructure and services.`,
		RunE:  run,
	}
	out = colorable.NewColorableStdout()
)

func run(cmd *cobra.Command, args []string) error {
	err := setupLogInstance()
	if err != nil {
		return err
	}
	files, err := filepath.Glob("/data/install/bin/*/*.env")
	if err != nil {
		log.Fatal().Err(err).Msg("Load env fail...")
	}
	log.Debug().Msg(fmt.Sprintf("%d env files will be load", len(files)))
	for _, file := range files {
		godotenv.Load(file)
	}
	app := NewApp()
	app.Start()

	return nil
}

func setupLogInstance() error {
	mod := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	file, err := os.OpenFile("./weterm.log", mod, 0644)
	if err != nil {
		return err
	}
	if os.Getenv("DEBUG") == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: file})
	log.Info().Msg("WeTerm starting up...")
	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(actionCmd())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
