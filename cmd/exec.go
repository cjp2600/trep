package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/cjp2600/trep/tui"
	textpkg "github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	parserpkg "github.com/cjp2600/trep/parser"
	reportpkg "github.com/cjp2600/trep/report"
)

var ExecCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Executes the given go test and formats its output",
	Args:  cobra.MinimumNArgs(1),
	Run:   executeCommand,
}

var onlyFail bool
var report bool
var reportPath string
var mode string
var reportName string

func init() {
	ExecCmd.Flags().StringVarP(&reportName, "report-name", "n", "", "Custom report name Example: report")
	ExecCmd.Flags().BoolVarP(&onlyFail, "only-fail", "f", false, "Only display failed tests")
	ExecCmd.Flags().BoolVarP(&report, "report", "r", false, "Generate a report")
	ExecCmd.Flags().StringVarP(&reportPath, "report-path", "p", "./", "Path to save the report (default is current directory)")
	ExecCmd.Flags().StringVarP(&mode, "mode", "m", "cli", "Run mode (e.g. 'cli', 'ci')")
}

type Exec struct {
	parser parserpkg.Parser
}

func NewExec() *Exec {
	return &Exec{
		parser: parserpkg.NewParser(),
	}
}

// parseArguments parses the given arguments and returns them as a slice
func parseArguments(inputArgs []string) []string {
	if len(inputArgs) == 0 {
		return inputArgs
	}

	firstArg := inputArgs[0]
	if strings.Contains(firstArg, " ") {
		splitArgs := strings.Split(firstArg, " ")
		return append(splitArgs, inputArgs[1:]...)
	}

	return inputArgs
}

// checkAndAddJSONFlag checks if the given args contain the -json flag and adds it if not
func checkAndAddFlags(args []string, flags ...string) []string {
	for _, flag := range flags {
		found := false
		for _, arg := range args {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			args = append(args, flag)
		}
	}

	return args
}

// executeCommand executes the given command and formats its output
func executeCommand(cmd *cobra.Command, args []string) {
	parsedArgs := parseArguments(args)
	parsedArgsWithRequiredFlag := checkAndAddFlags(parsedArgs, "--json", "-v", "--cover")
	if !strings.Contains(strings.Join(parsedArgsWithRequiredFlag, " "), "go test") {
		fmt.Println(textpkg.FgRed.Sprintf("Error: exec command only supports go test commands"))
		os.Exit(1)
		return
	}

	if err := runCommand(parsedArgsWithRequiredFlag[0], parsedArgsWithRequiredFlag[1:]...); err != nil {
		fmt.Println(textpkg.FgRed.Sprintf(err.Error()))
		os.Exit(1)
		return
	}

	os.Exit(0)
}

// collectNonJSONLines collects non JSON lines and returns them as a string
func collectNonJSONLines(line string, nonJSONLines *[]string) bool {
	if !isJSON(line) {
		*nonJSONLines = append(*nonJSONLines, line)
		return true
	}
	return false
}

// checkBuildFailure checks if there are any build failures
func checkBuildFailure(line string) error {
	if strings.HasPrefix(line, "FAIL") && strings.Contains(line, "[build failed]") {
		return fmt.Errorf("error: there are build issues. please check the logs and source code")
	}

	return nil
}

// isJSON checks if the given string is a valid JSON
func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// groupActionHandler groups actions by package and executes the handler function
func groupActionHandler(actions []*parserpkg.Action, handler func(action *parserpkg.Action)) {
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Time.Before(actions[j].Time)
	})

	var groupActions = make(map[string][]*parserpkg.Action)
	for _, action := range actions {
		groupActions[action.Package] = append(groupActions[action.Package], action)
	}

	for k, actionsList := range groupActions {
		for _, action := range actionsList {
			if action.Action == "skip" {
				delete(groupActions, k)
			}
		}
	}

	if len(groupActions) > 0 {
		for _, actionsList := range groupActions {
			for _, action := range actionsList {
				handler(action)
			}
		}
	}
}

// printAction prints the given action
func parseAction(actionStr string) (*parserpkg.Action, error) {
	if isJSON(actionStr) {
		var action parserpkg.Action
		err := json.Unmarshal([]byte(actionStr), &action)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling action: %w", err)
		}
		return &action, nil
	}

	return nil, fmt.Errorf("error parsing action: %s", actionStr)
}

// runCommand runs the given command and formats its output
func runCommand(name string, args ...string) error {
	ex := NewExec()

	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error getting stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error getting stderr pipe: %w", err)
	}

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	stopCh := make(chan bool)
	{
		loader(stopCh)
	}

	var actions []*parserpkg.Action
	var buildFailureErr error
	nonJSONLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if collectNonJSONLines(line, &nonJSONLines) {
			if err = checkBuildFailure(line); err != nil {
				buildFailureErr = err
			}
		}

		action, err := parseAction(line)
		if err != nil {
			continue
		}
		actions = append(actions, action)
	}

	if buildFailureErr != nil {
		stopCh <- true
		return fmt.Errorf("%w:\n\n %s", buildFailureErr, strings.Join(nonJSONLines, "\n"))
	}

	// Stop the loader
	stopCh <- true

	// group actions by package
	groupActionHandler(actions, func(action *parserpkg.Action) {
		ex.parser.Parse(action)
	})

	sum := ex.parser.GetSummary()
	{
		var opts []tui.RenderOptionFunc
		if onlyFail {
			if sum.TotalFailed > 0 {
				opts = append(opts, tui.WithOnlyFail())
			} else {
				if report {
					if err = reportpkg.GenerateAndSaveReport(sum, reportPath, reportName); err != nil {
						return fmt.Errorf("error save report output: %w", err)
					}
				}

				fmt.Println(textpkg.FgGreen.Sprint("All tests passed!"))
				fmt.Println(textpkg.FgGreen.Sprint(sum.TotalPackages, " tests total, ", sum.TotalPassed, " tests passed, ", sum.TotalFailed, " tests failed"))
				return nil
			}
		}
		if mode == "ci" {
			opts = append(opts, tui.WithEnableCIMode(true))
		}

		tui.BuildTable(sum, opts...).Render()
	}

	if report {
		if err = reportpkg.GenerateAndSaveReport(sum, reportPath, reportName); err != nil {
			return fmt.Errorf("error save report output: %w", err)
		}
	}

	if err = cmd.Wait(); err != nil {
		fmt.Println(textpkg.FgRed.Sprint(sum.TotalPackages, " tests total, ", sum.TotalPassed, " tests passed, ", sum.TotalFailed, " tests failed"))
		return fmt.Errorf("tests failed: %w", err)
	}

	fmt.Println(textpkg.FgGreen.Sprint(sum.TotalPackages, " tests total, ", sum.TotalPassed, " tests passed, ", sum.TotalFailed, " tests failed"))
	fmt.Println(textpkg.FgGreen.Sprint("All tests passed!"))
	return nil
}

// loader displays a loader while the tests are running
func loader(stopCh chan bool) {
	go func() {
		if mode == CIMode {
			fmt.Printf("Running tests...")
			for {
				select {
				case <-stopCh:
					fmt.Printf("\r")
				}
			}
		} else {
			loaderChars := `-\|/`
			i := 0
			for {
				select {
				case <-stopCh:
					fmt.Printf("\r")
					return
				default:
					fmt.Printf("\r%s Running tests... ", textpkg.FgCyan.Sprintf(string(loaderChars[i%len(loaderChars)])))
					time.Sleep(100 * time.Millisecond)
					i++
				}
			}
		}
	}()
}

const (
	CIMode = "ci"
)
