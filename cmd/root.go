package cmd

import (
	colorable "github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
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
