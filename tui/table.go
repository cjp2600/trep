package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	parserpkg "github.com/cjp2600/trep/parser"
	tablepkg "github.com/jedib0t/go-pretty/v6/table"
	textpkg "github.com/jedib0t/go-pretty/v6/text"
)

type renderOption struct {
	onlyFail *bool
	onlyPass *bool
}

type RenderOptionFunc func(*renderOption)

func WithOnlyFail() RenderOptionFunc {
	return func(opt *renderOption) {
		b := true
		opt.onlyFail = &b
	}
}

func WithOnlyPass() RenderOptionFunc {
	return func(opt *renderOption) {
		b := true
		opt.onlyPass = &b
	}
}

func Render(sum *parserpkg.Summary, opts ...RenderOptionFunc) {
	options := &renderOption{}
	for _, o := range opts {
		o(options)
	}

	t := tablepkg.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(tablepkg.Row{"Name", "Status", "Time", "Output"})

	for _, tr := range sum.PackageResults {
		//table.Title = t.PackageName
		for _, test := range tr.TestResults {
			if options.onlyFail != nil && *options.onlyFail && test.IsPassed {
				continue
			}
			if options.onlyPass != nil && *options.onlyPass && !test.IsPassed {
				continue
			}

			var isPassed string
			if test.IsPassed {
				isPassed = textpkg.FgGreen.Sprintf("✓ pass")
			} else {
				isPassed = textpkg.FgRed.Sprintf("× fail")
			}

			var elapsedTime string
			if test.ElapsedTime == 0 {
				elapsedTime = "0.000s"
			} else {
				elapsedTime = fmt.Sprintf("%.3fs", float64(test.ElapsedTime)/float64(time.Second))
			}

			var output = strings.Join(test.Output, "\n")
			output = strings.ReplaceAll(output, "\n\n", "\n")
			if len(output) > 0 && output[len(output)-1] == '\n' {
				output = output[:len(output)-1]
			}
			output, _ = extractError(output)

			var testName = test.TestName
			if test.IsPassed {
				testName = textpkg.FgGreen.Sprintf(testName)
				output = ""
			} else {
				testName = textpkg.FgRed.Sprintf(testName)
			}

			t.AppendRows([]tablepkg.Row{
				{testName, isPassed, elapsedTime, output},
			})

			for _, s := range test.Subtests {
				if options.onlyFail != nil && *options.onlyFail && s.IsPassed {
					continue
				}
				if options.onlyPass != nil && *options.onlyPass && !s.IsPassed {
					continue
				}

				if s.IsPassed {
					isPassed = textpkg.FgGreen.Sprintf("✓ pass")
				} else {
					isPassed = textpkg.FgRed.Sprintf("× fail")
				}

				if s.ElapsedTime == 0 {
					elapsedTime = "0.000s"
				} else {
					elapsedTime = fmt.Sprintf("%.3fs", float64(s.ElapsedTime)/float64(time.Second))
				}

				var out = strings.Join(s.Output, "\n")
				out = strings.ReplaceAll(out, "\n\n", "\n")
				out = strings.ReplaceAll(out, "\t\t", "\t")
				if len(out) > 0 && out[len(out)-1] == '\n' {
					out = out[:len(out)-1]
				}
				out, _ = extractError(out)

				var tName = s.TestName
				if s.IsPassed {
					tName = textpkg.FgGreen.Sprintf(tName)
					out = ""
				} else {
					tName = textpkg.FgRed.Sprintf(tName)
				}

				// Добавим отступ к имени Subtest для его выделения
				isLast := s == test.Subtests[len(test.Subtests)-1]
				var symbol = " ├─ "
				if isLast {
					symbol = " ╰─ "
				}
				t.AppendRows([]tablepkg.Row{
					{symbol + tName, isPassed, elapsedTime, out},
				})
			}
			t.AppendSeparator()
		}
	}
	t.SetStyle(tablepkg.StyleLight)

	t.SetAutoIndex(true)
	t.Render()
}

func extractError(text string) (string, error) {
	re := regexp.MustCompile(`Error:(?s)(.*?)(\n\s*Test:)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		errorText := matches[1]
		errorText = strings.ReplaceAll(errorText, "\t", " ")
		return strings.TrimSpace(errorText), nil
	}
	return "", fmt.Errorf("error block not found")
}
