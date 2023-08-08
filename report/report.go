package report

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"time"

	parserpkg "github.com/cjp2600/trep/parser"
	"github.com/cjp2600/trep/tui"
)

// GenerateAndSaveReport generates a report and saves it to the given path
func GenerateAndSaveReport(sum *parserpkg.Summary, reportPath string) error {
	html, err := captureStdout(func() {
		tui.BuildTable(sum, tui.WithReportColors()).RenderHTML()
	})
	if err != nil {
		return fmt.Errorf("error rendering html: %w", err)
	}
	return saveReport(html, reportPath)
}

// saveReport saves the report to the given path
func saveReport(tableHTML string, path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directories: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")

	type ReportData struct {
		ReportName string
		Table      template.HTML
	}

	t := template.Must(template.New("report").Parse(reportTemplate))

	data := ReportData{
		ReportName: fmt.Sprintf("Report %s", timestamp),
		Table:      template.HTML(html.UnescapeString(tableHTML)),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	filename := fmt.Sprintf("%s/report_%s.html", path, timestamp) // Использование path здесь

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
</style>
</head>
<body>
    {{ .Table }}
</body>
</html>`
