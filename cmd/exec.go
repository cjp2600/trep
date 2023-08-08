package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	ExecCmd.Flags().BoolVarP(&onlyFail, "only-fail", "f", false, "Only display failed tests")
	ExecCmd.Flags().BoolVarP(&report, "report", "r", false, "Generate a report")
	ExecCmd.Flags().StringVarP(&reportPath, "report-path", "p", "./", "Path to save the report (default is current directory)")
	ExecCmd.Flags().StringVarP(&mode, "mode", "m", "cli", "Run mode (e.g. 'cli', 'ci')")
	ExecCmd.Flags().StringVarP(&reportName, "report-name", "n", "", "Custom report name Example: report") // Новый флаг
}

type Exec struct {
	parser parserpkg.Parser
}

func NewExec() *Exec {
	return &Exec{
		parser: parserpkg.NewParser(),
	}
}

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

func checkAndAddJSONFlag(args []string) []string {
	for _, arg := range args {
		if arg == "-json" {
			return args
		}
	}

	return append(args, "-json")
}

func executeCommand(cmd *cobra.Command, args []string) {
	parsedArgs := parseArguments(args)
	parsedArgsWithJSONFlag := checkAndAddJSONFlag(parsedArgs)
	if !strings.Contains(strings.Join(parsedArgsWithJSONFlag, " "), "go test") {
		fmt.Println(textpkg.FgRed.Sprintf("Error: exec command only supports go test commands"))
		os.Exit(1)
		return
	}

	if err := runCommand(parsedArgsWithJSONFlag[0], parsedArgsWithJSONFlag[1:]...); err != nil {
		fmt.Println(textpkg.FgRed.Sprintf(err.Error()))
		os.Exit(1)
		return
	}

	os.Exit(0)
}

func (e *Exec) parseLine(line string) {
	e.parser.Parse(line)
}

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

	for scanner.Scan() {
		ex.parser.Parse(scanner.Text())
	}

	// Stop the loader
	stopCh <- true

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
		if mode == "ci" {
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
