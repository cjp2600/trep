package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	parserpkg "github.com/cjp2600/trep/parser"
	tablepkg "github.com/jedib0t/go-pretty/v6/table"
	textpkg "github.com/jedib0t/go-pretty/v6/text"
)

// BuildTable builds a table with the given summary
func BuildTable(sum *parserpkg.Summary, opts ...RenderOptionFunc) tablepkg.Writer {
	options := &renderOption{}
	for _, o := range opts {
		o(options)
	}

	// Check if any test has output
	hasOutput := false
	checkOutput := func(test *parserpkg.TestResult) {
		output := getOutput(test)
		if len(output) > 0 {
			hasOutput = true
		}
	}

	for _, tr := range sum.PackageResults {
		for _, test := range tr.TestResults {
			checkOutput(test)
			for _, s := range test.Subtests {
				checkOutput(s)
			}
		}
	}

	t := tablepkg.NewWriter()
	t.SetOutputMirror(os.Stdout)

	headerRows := tablepkg.Row{"Name", "Status"}
	if !options.ciMode && hasOutput {
		headerRows = append(headerRows, "Output")
	}
	t.AppendHeader(headerRows)

	processTest := func(test *parserpkg.TestResult, isSubtest bool, isLast bool) {
		if options.onlyFail != nil && *options.onlyFail && test.IsPassed {
			return
		}
		if options.onlyPass != nil && *options.onlyPass && !test.IsPassed {
			return
		}

		var testName = test.TestName
		var isBold = len(test.Subtests) > 0 && !isSubtest
		if test.IsPassed {
			testName = formatWithColor(testName, textpkg.FgGreen, options.ReportColors(), isBold)
		} else {
			testName = formatWithColor(testName, textpkg.FgRed, options.ReportColors(), isBold)
		}

		if isSubtest {
			symbol := getSymbol(isLast, options.ReportColors())
			testName = symbol + testName
		}

		tRows := []tablepkg.Row{
			{testName, getIsPassedStr(test.IsPassed, options.ReportColors())},
		}

		if !options.ciMode && hasOutput {
			tRows[0] = append(tRows[0], getOutput(test))
		}
		t.AppendRows(tRows)
	}

	for _, tr := range sum.PackageResults {
		for _, test := range tr.TestResults {
			processTest(test, false, false)

			for i, s := range test.Subtests {
				isLast := i == len(test.Subtests)-1
				processTest(s, true, isLast)
			}
			t.AppendSeparator()
		}
	}

	t.SetStyle(tablepkg.StyleLight)
	t.SetAutoIndex(true)
	return t
}

// extractErrorOrPanic extracts error or panic from the given text
func extractErrorOrPanic(text string) (string, error) {
	re := regexp.MustCompile(`Error:(?s)(.*?)(\n\s*Test:)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		errorText := matches[1]
		errorText = strings.ReplaceAll(errorText, "\t", " ")
		return strings.TrimSpace(errorText), nil
	}

	rePanic := regexp.MustCompile(`panic:(?s)(.*?)(\n\sgoroutine)`)
	matchesPanic := rePanic.FindStringSubmatch(text)
	if len(matchesPanic) > 1 {
		panicText := matchesPanic[1]
		panicText = strings.ReplaceAll(panicText, "\t", " ")
		return strings.TrimSpace(panicText), nil
	}

	reLog := regexp.MustCompile(`(?m)^\s*---\sLOG:(.*)$`)
	matchesLog := reLog.FindStringSubmatch(text)
	if len(matchesLog) > 1 {
		logText := matchesLog[1]
		logText = strings.ReplaceAll(logText, "\t", " ")
		return strings.TrimSpace(logText), nil
	}

	reBuildFailed := regexp.MustCompile(`FAIL\s*[^\s]+ \[build failed\](.*)`)
	matchesBuildFailed := reBuildFailed.FindStringSubmatch(text)
	if len(matchesBuildFailed) > 0 {
		buildFailedText := matchesBuildFailed[0]
		buildFailedText = strings.ReplaceAll(buildFailedText, "\t", " ")
		return strings.TrimSpace(buildFailedText), nil
	}

	return text, nil
}

// formatWithColor formats the given output with the given color
func formatWithColor(output string, color textpkg.Color, applyColor bool, isBold bool) string {
	if !applyColor {
		if isBold {
			return textpkg.Bold.Sprintf(color.Sprintf(output))
		}
		return color.Sprintf(output)
	}

	if isBold {
		return fmt.Sprintf("<span %s><b>%s</b></span>", color.HTMLProperty(), output)
	}
	return fmt.Sprintf("<span %s>%s</span>", color.HTMLProperty(), output)
}

// getIsPassedStr returns the string representation of the given isPassed value
func getIsPassedStr(isPassed bool, reportColors bool) string {
	var passedStr string
	if isPassed {
		passedStr = formatWithColor("✓ pass", textpkg.FgGreen, reportColors, true)
	} else {
		passedStr = formatWithColor("× fail", textpkg.FgRed, reportColors, true)
	}

	return passedStr
}

// getSymbol returns the symbol for the given test
func getSymbol(isLast bool, reportColors bool) string {
	var symbol = " ├─ "
	if isLast {
		symbol = " ╰─ "
	}

	if reportColors {
		symbol = "&nbsp;&nbsp;&nbsp;&nbsp;"
	}

	return symbol
}

// formatCompactOutput formats the given string to a compact output
func formatCompactOutput(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n\n", "\n")
	s = strings.ReplaceAll(s, "  ", " ")
	s = strings.ReplaceAll(s, "   ", " ")
	return s
}

// getOutput returns the output of the given test
func getOutput(test *parserpkg.TestResult) string {
	output, err := extractErrorOrPanic(strings.TrimSpace(strings.Join(test.Output, "\n")))
	if err != nil {
		return ""
	}

	if !test.IsPassed {
		return formatCompactOutput(output)
	}

	return ""
}

type renderOption struct {
	onlyFail     *bool
	onlyPass     *bool
	reportColors *bool
	ciMode       bool
}

func (r renderOption) ReportColors() bool {
	if r.reportColors == nil {
		return false
	}
	return *r.reportColors
}

func WithEnableCIMode(ciMode bool) RenderOptionFunc {
	return func(opt *renderOption) {
		opt.ciMode = ciMode
	}
}

type RenderOptionFunc func(*renderOption)

func WithOnlyFail() RenderOptionFunc {
	return func(opt *renderOption) {
		b := true
		opt.onlyFail = &b
	}
}

func WithReportColors() RenderOptionFunc {
	return func(opt *renderOption) {
		b := true
		opt.reportColors = &b
	}
}

func WithOnlyPass() RenderOptionFunc {
	return func(opt *renderOption) {
		b := true
		opt.onlyPass = &b
	}
}
