package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		Long:  "Execute predefined action scripts like reload_casbin, unseal_vault, backup_mongodb, backup_mysql",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
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
	var script string
	var scriptName string

	switch actionName {
	case "reload_casbin":
		script = action.ReloadCasbin
		scriptName = "reload_casbin.sh"
	case "unseal_vault":
		script = action.UnsealVaultScript
		scriptName = "unseal_vault.sh"
	case "backup_mongodb":
		script = action.BackupMongodb
		scriptName = "backup_mongodb.sh"
	case "backup_mysql":
		script = action.BackupMysql
		scriptName = "backup_mysql.sh"
	default:
		fmt.Printf("Unknown action: %s\n", actionName)
		fmt.Println("Available actions:")
		fmt.Println("  - reload_casbin: Reload casbin mesh rules from WeOps")
		fmt.Println("  - unseal_vault: Unseal vault service")
		fmt.Println("  - backup_mongodb: Backup MongoDB database")
		fmt.Println("  - backup_mysql: Backup MySQL database")
		return
	}

	// Create temporary script file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, scriptName)

	err := os.WriteFile(tmpFile, []byte(script), 0755)
	if err != nil {
		fmt.Printf("Error creating temporary script file: %v\n", err)
		return
	}
	defer os.Remove(tmpFile)

	fmt.Printf("Executing action: %s\n", actionName)
	fmt.Println(strings.Repeat("-", 50))

	// Execute the script with custom output handling
	cmd := exec.Command("bash", tmpFile)

	// Create pipes for stdout and stderr
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

	cmd.Stdin = os.Stdin

	// Start the command
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Error starting script: %v\n", err)
		return
	}

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
