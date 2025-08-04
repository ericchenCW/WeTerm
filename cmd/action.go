package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"weterm/pages/action"
	"weterm/utils"

	"github.com/spf13/cobra"
)

func actionCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "action [action_name]",
		Short: "Execute predefined actions",
		Long:  "Execute predefined action scripts. Run 'action' without arguments to see available actions.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// 显示所有可用的动作
				fmt.Println("Available actions:")
				for name, info := range action.GetAllActions() {
					fmt.Printf("  - %s: %s\n", name, info.Description)
				}
				return
			}
			actionName := args[0]
			executeAction(actionName)
		},
	}

	return command
}

// colorizeOutput converts color tags like [yellow] to ANSI color codes
func colorizeOutput(text string) string {
	colorMap := map[string]utils.Paint{
		"[yellow]":  utils.Yellow,
		"[white]":   utils.LightGray,
		"[green]":   utils.Green,
		"[red]":     utils.Red,
		"[blue]":    utils.Blue,
		"[cyan]":    utils.Cyan,
		"[magenta]": utils.Magenta,
	}

	// Replace color tags with ANSI color codes
	re := regexp.MustCompile(`\[(yellow|white|green|red|blue|cyan|magenta)\]`)
	result := re.ReplaceAllStringFunc(text, func(match string) string {
		if color, exists := colorMap[match]; exists {
			return fmt.Sprintf("\x1b[%dm", color)
		}
		return match
	})

	return result
}

func executeAction(actionName string) {
	// 获取动作信息
	actionInfo, exists := action.GetAction(actionName)
	if !exists {
		fmt.Printf("Unknown action: %s\n", actionName)
		fmt.Println("Available actions:")

		// 动态显示所有可用的动作
		for name, info := range action.GetAllActions() {
			fmt.Printf("  - %s: %s\n", name, info.Description)
		}
		return
	}

	fmt.Printf("Executing action: %s\n", actionName)
	fmt.Println(strings.Repeat("-", 50))

	// 直接通过管道执行脚本，不需要创建临时文件
	cmd := exec.Command("bash")

	// Create pipes for stdin, stdout and stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Error creating stdin pipe: %v\n", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stderr pipe: %v\n", err)
		return
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Error starting script: %v\n", err)
		return
	}

	// Write script content to stdin and close it
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(actionInfo.Script))
	}()

	// Handle stdout with color processing
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			colorizedLine := colorizeOutput(line)
			fmt.Println(colorizedLine)
		}
	}()

	// Handle stderr with color processing
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			colorizedLine := colorizeOutput(line)
			fmt.Fprintln(os.Stderr, colorizedLine)
		}
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Error executing script: %v\n", err)
		return
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Action %s completed\n", actionName)
}
