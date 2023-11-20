package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"weterm/utils"
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

	command.PersistentFlags().BoolVarP(&short, "short", "s", false, "Prints WeTerm version info in short format")

	return &command
}

func printVersion(short bool) {
	const format = "%-20s %s\n"
	var outputColor utils.Paint

	if short {
		outputColor = -1
	} else {
		outputColor = utils.Cyan
		printLogo(outputColor)
	}
	printTuple(format, "Version", "1.0.0", outputColor)
}

var LogoSmall = []string{
	`__        __    _____                      `,
	`\ \      / /___|_   _|___ _ __  _ __   ___ `,
	` \ \ /\ / // _ \| | / _ \| '__|| '_ \ / _ \`,
	`  \ V  V /|  __/| ||  __/ |   | | | |  __/`,
	`   \_/\_/  \___||_| \___||_|   |_| |_|\___|`,
}

func printLogo(c utils.Paint) {
	for _, l := range LogoSmall {
		fmt.Fprintln(out, utils.Colorize(l, c))
	}
	fmt.Fprintln(out)
}

func printTuple(fmat, section, value string, outputColor utils.Paint) {
	if outputColor != -1 {
		fmt.Fprintf(out, fmat, utils.Colorize(section+":", outputColor), value)
		return
	}
	fmt.Fprintf(out, fmat, section, value)
}
