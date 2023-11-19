package template

import (
	"bufio"
	"fmt"
	"github.com/rivo/tview"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"weterm/component"
	"weterm/model"
)

type Validation struct {
	Required bool
	Type     string // The type of validation to perform: "none", "numeric", "ip", "range"
	Range    []int  // The range for numeric validation
}

type FormItem struct {
	Type        string // "drop_down", "checkbox", "text_area", "input_field"
	Label       string
	Options     []string // For drop-down items
	Default     int      // Default selected option for drop-down items
	DefaultText string   // Default text for text area and input field
	Validation  *Validation
}

// CreateForm creates a form with the given items.
func CreateForm(formItems []FormItem) *tview.Form {
	form := tview.NewForm()
	for _, item := range formItems {
		switch item.Type {
		case "drop_down":
			form.AddDropDown(item.Label, item.Options, item.Default, nil)
		case "checkbox":
			form.AddCheckbox(item.Label, false, nil)
		case "text_area":
			form.AddTextArea(item.Label, item.DefaultText, 10, 0, 20, nil)
		case "input_field":
			form.AddInputField(item.Label, item.DefaultText, 20, nil, nil)
		}
	}
	return form
}

// ShowShellFormExecutePage sets up the form page and executes the shell command.
func ShowShellFormExecutePage(receiver *model.AppModel, title string, shellCommandTemplate string, formItems []FormItem) {
	formPage := CreateForm(formItems)
	formPage.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)

	// Create a TextView for output
	outputTextView := tview.NewTextView().SetDynamicColors(true)
	outputTextView.SetBorder(true).SetTitle("输出").SetTitleAlign(tview.AlignCenter)
	outputTextView.SetText("")

	formPage.AddButton("执行", func() {
		outputTextView.SetText("")
		// Read values from the form and pass them to the shell command
		shellCommand := shellCommandTemplate
		for i, item := range formItems {
			var value string
			switch item.Type {
			case "text_area":
				value = formPage.GetFormItem(i).(*tview.TextArea).GetText()
				shellCommand = strings.Replace(shellCommand, "{"+item.Label+"}", value, -1)
			case "checkbox":
				value = "false"
				if formPage.GetFormItem(i).(*tview.Checkbox).IsChecked() {
					value = "true"
				}
				shellCommand = strings.Replace(shellCommand, "{"+item.Label+"}", value, -1)
			case "drop_down":
				_, option := formPage.GetFormItem(i).(*tview.DropDown).GetCurrentOption()
				shellCommand = strings.Replace(shellCommand, "{"+item.Label+"}", option, -1)
			case "input_field":
				value = formPage.GetFormItem(i).(*tview.InputField).GetText()
				shellCommand = strings.Replace(shellCommand, "{"+item.Label+"}", value, -1)
			}

			// Perform validation
			if item.Validation != nil {
				// Check if the value is required and not empty
				if item.Validation.Required && value == "" {
					alert := component.NewAlert()
					alert.ShowAlert(receiver.CorePages, item.Label+"不能为空")
					return
				}

				switch item.Validation.Type {
				case "numeric":
					// Check if the value is a number
					if _, err := strconv.Atoi(value); err != nil {
						alert := component.NewAlert()
						alert.ShowAlert(receiver.CorePages, item.Label+" must be a number")
						return
					}
				case "ip":
					// Check if the value is a valid IP address
					if net.ParseIP(value) == nil {
						alert := component.NewAlert()
						alert.ShowAlert(receiver.CorePages, item.Label+" must be a valid IP address")
						return
					}
				case "range":
					// Check if the value is within the specified range
					// Make sure Range has at least 2 elements before accessing its elements
					if num, err := strconv.Atoi(value); err != nil || len(item.Validation.Range) >= 2 && (num < item.Validation.Range[0] || num > item.Validation.Range[1]) {
						alert := component.NewAlert()
						alert.ShowAlert(receiver.CorePages, fmt.Sprintf("%s must be a number between %d and %d", item.Label, item.Validation.Range[0], item.Validation.Range[1]))
						return
					}
				}
			}
		}

		// Execute the shell command
		cmd := exec.Command("bash", "-c", shellCommand)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(outputTextView, "Error: %v\n", err)
			return
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(outputTextView, "Error: %v\n", err)
			return
		}
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				receiver.CoreApp.QueueUpdateDraw(func() {
					fmt.Fprintf(outputTextView, "%s\n", scanner.Text())
					outputTextView.ScrollToEnd()
				})
			}
			if err := scanner.Err(); err != nil {
				receiver.CoreApp.QueueUpdateDraw(func() {
					fmt.Fprintf(outputTextView, "Error: %v\n", err)
				})
			}

			// Wait for the command to finish
			if err := cmd.Wait(); err != nil {
				receiver.CoreApp.QueueUpdateDraw(func() {
					fmt.Fprintf(outputTextView, "Error: %v\n", err)
				})
			}
		}()
	})

	// Add the form and the output view to a Flex and add the Flex to the pages
	flex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(formPage, 0, 1, true).
		AddItem(outputTextView, 0, 3, false)
	receiver.CorePages.AddPage("shell_form_execute", flex, true, false)
}
