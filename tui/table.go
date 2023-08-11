package tui

import (
	"fmt"
	parserpkg "github.com/cjp2600/trep/parser"
	tablepkg "github.com/jedib0t/go-pretty/v6/table"
	textpkg "github.com/jedib0t/go-pretty/v6/text"
	"os"
	"regexp"
	"strings"
)

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

// color returns the colored output if reportColors is true, else returns the
func color(output string, color textpkg.Color, opts ...bool) string {
	var isReportColors = len(opts) > 0 && opts[0]
	var isBold = len(opts) > 1 && opts[1]

	if !isReportColors {
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

func getIsPassedStr(isPassed bool, reportColors bool) string {
	var passedStr string
	if isPassed {
		passedStr = color("✓ pass", textpkg.FgGreen, reportColors)
	} else {
		passedStr = color("× fail", textpkg.FgRed, reportColors)
	}

	return passedStr
}

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

func BuildTable(sum *parserpkg.Summary, opts ...RenderOptionFunc) tablepkg.Writer {
	options := &renderOption{}
	for _, o := range opts {
		o(options)
	}

	t := tablepkg.NewWriter()
	t.SetOutputMirror(os.Stdout)

	rows := tablepkg.Row{"Name", "Status"}
	if !options.ciMode {
		rows = append(rows, "Output")
	}

	t.AppendHeader(rows)

	for _, tr := range sum.PackageResults {
		for _, test := range tr.TestResults {
			if options.onlyFail != nil && *options.onlyFail && test.IsPassed {
				continue
			}
			if options.onlyPass != nil && *options.onlyPass && !test.IsPassed {
				continue
			}

			var output = strings.Join(test.Output, "\n")
			output = strings.ReplaceAll(output, "\n\n", "\n")
			if len(output) > 0 && output[len(output)-1] == '\n' {
				output = output[:len(output)-1]
			}
			output, _ = extractErrorOrPanic(output)

			var testName = test.TestName
			var isBold = len(test.Subtests) > 0
			if test.IsPassed {
				testName = color(testName, textpkg.FgGreen, options.ReportColors(), isBold)
				output = ""
			} else {
				testName = color(testName, textpkg.FgRed, options.ReportColors(), isBold)
			}

			tRows := []tablepkg.Row{
				{testName, getIsPassedStr(test.IsPassed, options.ReportColors())},
			}

			if !options.ciMode {
				tRows[0] = append(tRows[0], output)
			}
			t.AppendRows(tRows)

			for _, s := range test.Subtests {
				if options.onlyFail != nil && *options.onlyFail && s.IsPassed {
					continue
				}
				if options.onlyPass != nil && *options.onlyPass && !s.IsPassed {
					continue
				}

				var out = strings.Join(s.Output, "\n")
				out = strings.ReplaceAll(out, "\n\n", "\n")
				out = strings.ReplaceAll(out, "\t\t", "\t")
				if len(out) > 0 && out[len(out)-1] == '\n' {
					out = out[:len(out)-1]
				}
				out, _ = extractErrorOrPanic(out)

				var tName = s.TestName
				if s.IsPassed {
					tName = color(tName, textpkg.FgGreen, options.ReportColors())
					out = ""
				} else {
					tName = color(tName, textpkg.FgRed, options.ReportColors())
				}

				// Добавим отступ к имени Subtest для его выделения
				isLast := s == test.Subtests[len(test.Subtests)-1]
				var symbol = getSymbol(isLast, options.ReportColors())
				sRows := []tablepkg.Row{
					{symbol + tName, getIsPassedStr(s.IsPassed, options.ReportColors())},
				}

				if !options.ciMode {
					sRows[0] = append(sRows[0], out)
				}
				t.AppendRows(sRows)
			}
			t.AppendSeparator()
		}
	}
	t.SetStyle(tablepkg.StyleLight)

	t.SetAutoIndex(true)
	return t
}

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
