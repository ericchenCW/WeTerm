package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	var short bool

	command := cobra.Command{
		Use:   "version",
		Short: "Print version/build info",
		Long:  "Print version/build information",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion(short)
		},
	}

	command.PersistentFlags().BoolVarP(&short, "short", "s", false, "Prints K9s version info in short format")

	return &command
}

func printVersion(short bool) {
	const fmat = "%-20s %s\n"
	var outputColor Paint

	if short {
		outputColor = -1
	} else {
		outputColor = Cyan
		printLogo(outputColor)
	}
	printTuple(fmat, "Version", "1.0.0", outputColor)
}

var LogoSmall = []string{
	`__        __    _____                      `,
	`\ \      / /___|_   _|___ _ __  _ __   ___ `,
	` \ \ /\ / // _ \| | / _ \| '__|| '_ \ / _ \`,
	`  \ V  V /|  __/| ||  __/ |   | | | |  __/`,
	`   \_/\_/  \___||_| \___||_|   |_| |_|\___|`,
}

func printLogo(c Paint) {
	for _, l := range LogoSmall {
		fmt.Fprintln(out, Colorize(l, c))
	}
	fmt.Fprintln(out)
}

func printTuple(fmat, section, value string, outputColor Paint) {
	if outputColor != -1 {
		fmt.Fprintf(out, fmat, Colorize(section+":", outputColor), value)
		return
	}
	fmt.Fprintf(out, fmat, section, value)
}
