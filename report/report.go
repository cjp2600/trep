package report

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	parserpkg "github.com/cjp2600/trep/parser"
	"github.com/cjp2600/trep/tui"
	htmlpkg "golang.org/x/net/html"
)

// GenerateAndSaveReport generates a report and saves it to the given path
func GenerateAndSaveReport(sum *parserpkg.Summary, reportPath string, reportName string) error {
	html, err := captureStdout(func() {
		tui.BuildTable(sum, tui.WithReportColors()).RenderHTML()
	})
	if err != nil {
		return fmt.Errorf("error rendering html: %w", err)
	}
	return saveReport(html, reportPath, reportName, sum)
}

func getFailedTests(sum *parserpkg.Summary) []string {
	var failedTests []string
	for _, pkg := range sum.PackageResults {
		for _, test := range pkg.TestResults {
			if !test.IsPassed {
				failedTests = append(failedTests, test.TestName)
			}
			if len(test.Subtests) > 0 {
				for _, subtest := range test.Subtests {
					if !subtest.IsPassed {
						failedTests = append(failedTests, subtest.TestName)
					}
				}
			}
		}
	}
	return failedTests
}

// saveReport saves the report to the given path
func saveReport(tableHTML string, path string, reportName string, sum *parserpkg.Summary) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	tableHTML = insertRowIDs(tableHTML)

	timestamp := time.Now().Format("20060102_150405")

	type ReportData struct {
		ReportName  string
		Table       template.HTML
		Total       int
		Passed      int
		Failed      int
		IsPassed    bool
		GeneratedAt string
		FailedTests []string
	}

	t := template.Must(template.New("report").Parse(reportTemplate))

	data := ReportData{
		ReportName:  fmt.Sprintf("Report %s", timestamp),
		Table:       template.HTML(html.UnescapeString(tableHTML)),
		Total:       sum.TotalPackages,
		Passed:      sum.TotalPassed,
		Failed:      sum.TotalFailed,
		IsPassed:    sum.TotalFailed == 0,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		FailedTests: getFailedTests(sum),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	var rep string
	if reportName != "" {
		rep = reportName
	} else {
		rep = "report_" + timestamp
	}

	filename := fmt.Sprintf("%s/%s.html", path, rep) // Использование path здесь

	if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	fmt.Printf("Report saved to %s\n", filename)
	return nil
}

// captureStdout captures the stdout of the given function
func captureStdout(f func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	originalStdout := os.Stdout
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			// Возможно, вам стоит обработать ошибку здесь
		}
		outC <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = originalStdout
	output := <-outC

	return output, nil
}

func stripHTML(input string) string {
	doc, err := htmlpkg.Parse(strings.NewReader(input))
	if err != nil {
		return input // возвращаем исходный текст, если парсинг не удался
	}

	var textContent string
	var f func(*htmlpkg.Node)
	f = func(n *htmlpkg.Node) {
		if n.Type == htmlpkg.TextNode {
			textContent += n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return textContent
}

func insertRowIDs(htmlTable string) string {
	doc, err := htmlpkg.Parse(strings.NewReader(htmlTable))
	if err != nil {
		fmt.Println("error parsing HTML:", err)
		return ""
	}

	var f func(*htmlpkg.Node)
	f = func(n *htmlpkg.Node) {
		if n.Type == htmlpkg.ElementNode && n.Data == "tr" {
			tdCount := 0
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == htmlpkg.ElementNode && c.Data == "td" {
					tdCount++
					if tdCount == 2 {
						for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
							if cc.Type == htmlpkg.TextNode {
								strID := strings.ReplaceAll(cc.Data, "&nbsp;", "")
								strID = stripHTML(strID)
								n.Attr = append(n.Attr, htmlpkg.Attribute{Key: "id", Val: strID})
								break
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var b bytes.Buffer
	htmlpkg.Render(&b, doc)
	return b.String()
}

// reportTemplate is the template used to generate the report
var reportTemplate = `<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
<title>{{ .ReportName }} </title>
<style>
  body {
      font-family: "Helvetica Neue",Helvetica,Arial,sans-serif;
      font-size: 14px;
      line-height: 1.42857143;
      color: #333;
      background-color: #fff;
  }
  .go-pretty-table {
      border-collapse: collapse;
      border-spacing: 0;
      border: 1px solid #ddd;
      width: 100%;
      margin-bottom: 20px;
      background-color: transparent;
  }
  .go-pretty-table th,
  .go-pretty-table td {
      border: 1px solid #ddd;
      padding: 8px;
      text-align: left;
  }
  .go-pretty-table th {
      background-color: #f5f5f5;
      color: #333;
  }
  .go-pretty-table tr:nth-child(even) {
      background-color: #f2f2f2;
  }
  .go-pretty-table tr:hover {
      background-color: #f5f5f5; /* Цвет выделения при наведении */
  }
  .fg-red {
      color: #a94442;
  }
  .fg-green {
      color: #3c763d;
  }
  .summary {
    border: 1px solid #ddd;
    border-radius: 4px;
    padding: 15px;
    background-color: #f9f9f9;
    margin-bottom: 20px;
  }
  .summary h3 {
    margin-top: 0;
    color: #333;
  }
  .summary ul {
    list-style: none;
    padding-left: 0;
  }
  .summary ul li {
    margin-bottom: 10px;
  }
  .summary-table {
    width: 100%;
    border-collapse: collapse;
  }
  .summary-table th,
  .summary-table td {
    text-align: left;
    padding: 8px;
    border-bottom: 1px solid #ddd;
  }
.failed-tests {
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 15px;
  background-color: #fdfdfd;
  margin-top: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.failed-tests h3 {
  margin-top: 0;
  color: #a94442;
}

.failed-tests ul {
  list-style-type: none;
  padding-left: 0;
}

.failed-tests ul li {
  margin-bottom: 10px;
}

.failed-tests ul li a {
  color: black;
  text-decoration: none;
}

.failed-tests ul li a:hover {
  text-decoration: underline;
}

</style>
</head>
<body>
<div class="summary">
  <h3>{{ .ReportName }}</h3>
  <table class="summary-table">
    <tr>
      <th>Generated at:</th>
      <td>{{ .GeneratedAt }}</td>
    </tr>
    <tr>
      <th>Total:</th>
      <td>{{ .Total }}</td>
    </tr>
    <tr>
      <th>Passed:</th>
      <td>{{ .Passed }}</td>
    </tr>
    <tr>
      <th>Failed:</th>
      <td>{{ .Failed }}</td>
    </tr>
    <tr>
      <th>Status:</th>
      <td>
        {{ if .IsPassed }}
          <span class="fg-green">PASS</span>
        {{ else }}
          <span class="fg-red">FAIL</span>
        {{ end }}
      </td>
    </tr>
  </table>
</div>
{{ if  .IsPassed }}
{{ else }}
<div class="failed-tests">
  <h3>Failed Tests</h3>
  <ul id="failedTestsList">
    {{ range .FailedTests }}
      <li><a href="#{{ . }}" style="color: black;">{{ . }}</a></li>
    {{ end }}
  </ul>
</div>
{{ end }}
{{ .Table }}
</body>
</html>`
