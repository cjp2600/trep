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
)

var ExecCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Executes the given go test and formats its output",
	Args:  cobra.MinimumNArgs(1),
	Run:   executeCommand,
}

var onlyFail bool

func init() {
	ExecCmd.Flags().BoolVar(&onlyFail, "onlyFail", false, "Only display failed tests")
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
	go func() {
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
	}()

	for scanner.Scan() {
		ex.parser.Parse(scanner.Text())
	}

	// Stop the loader
	stopCh <- true

	sum := ex.parser.GetSummary()
	{
		if onlyFail {
			if sum.TotalFailed > 0 {
				tui.Render(sum, tui.WithOnlyFail())
			} else {
				fmt.Println(textpkg.FgGreen.Sprint("All tests passed!"))
				return nil
			}
		} else {
			tui.Render(sum)
		}
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	fmt.Println(textpkg.FgGreen.Sprint("All tests passed!"))
	return nil
}
