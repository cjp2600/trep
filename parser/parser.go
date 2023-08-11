package parser

import (
	"encoding/json"
	"strings"
	"time"
)

type Parser interface {
	Parse(line string)
	GetSummary() *Summary
}

type parser struct {
	line string
	sum  *Summary

	currentPackage *PackageResult
	currentTest    *TestResult
}

func NewParser() Parser {
	return &parser{
		sum:            &Summary{},
		currentPackage: &PackageResult{},
		currentTest:    &TestResult{},
	}
}

func (p *parser) Parse(actionStr string) {
	var action Action
	err := json.Unmarshal([]byte(actionStr), &action)
	if err != nil {
		return
	}

	switch action.Action {
	case "start":
		p.currentPackage = &PackageResult{
			PackageName: action.Package,
			StartTime:   action.Time,
			TestResults: make(map[string]*TestResult), // Initialize the map
		}
	case "output":
		if action.Test == "" {
			p.currentPackage.Output = append(p.currentPackage.Output, action.Output)
			return
		}
		p.currentTest.Output = append(p.currentTest.Output, action.Output)

	case "run":
		testNames := strings.Split(action.Test, "/")
		testName := testNames[len(testNames)-1]
		parentTestName := ""
		if len(testNames) > 1 {
			parentTestName = strings.Join(testNames[:len(testNames)-1], "/")
		}

		p.currentTest = &TestResult{
			TestName:  testName,
			StartTime: action.Time,
			IsPassed:  true,
		}
		p.sum.TotalPassed++
		p.sum.TotalPackages++

		if parentTestName == "" {
			p.currentPackage.TestResults[action.Test] = p.currentTest
		} else {
			parentTest := findTest(parentTestName, p.currentPackage.TestResults)
			if parentTest != nil {
				parentTest.Subtests = append(parentTest.Subtests, p.currentTest)
			} else {
				// If the parent test is not found, create a new one
			}
		}

	case "pass":
		if action.Test == "" {
			break
		}

		p.currentTest.EndTime = action.Time
		p.currentTest.ElapsedTime = action.Elapsed
		p.currentTest.IsPassed = checkSubtestsPassed(p.currentTest)

		if v, ok := p.currentPackage.TestResults[action.Test]; ok {
			v.IsPassed = p.currentTest.IsPassed
		}
	case "skip":
		return

	case "fail":
		if action.Test == "" {
			break
		}

		p.currentTest.EndTime = action.Time
		p.currentTest.ElapsedTime = action.Elapsed
		p.currentTest.IsPassed = false
		p.sum.TotalFailed++
		p.sum.TotalPassed--

		if p.currentPackage.PackageName == action.Package {
			p.currentPackage.IsPassed = false
		}

		setParentTestsFailed(action.Test, p.currentPackage.TestResults)
	}

	// If the package test ended
	if (action.Action == "pass" || action.Action == "fail") && action.Test == "" {
		p.currentPackage.EndTime = action.Time
		p.currentPackage.ElapsedTime = action.Elapsed
		p.currentPackage.IsPassed = true
		for _, test := range p.currentPackage.TestResults {
			if !test.IsPassed {
				p.currentPackage.IsPassed = false
				break
			}
		}

		p.sum.PackageResults = append(p.sum.PackageResults, *p.currentPackage)
	}
}

func setParentTestsFailed(testName string, testResults map[string]*TestResult) {
	parts := strings.Split(testName, "/")
	for i := len(parts) - 1; i > 0; i-- {
		parentTestName := strings.Join(parts[:i], "/")
		parentTest := findTest(parentTestName, testResults)
		if parentTest != nil {
			parentTest.IsPassed = false
		}
	}
}

func checkSubtestsPassed(test *TestResult) bool {
	for _, subTest := range test.Subtests {
		if !subTest.IsPassed || !checkSubtestsPassed(subTest) {
			return false
		}
	}
	return true
}

func findTest(testName string, testResults map[string]*TestResult) *TestResult {
	if result, ok := testResults[testName]; ok {
		return result
	}

	for _, result := range testResults {
		for _, subTest := range result.Subtests {
			if foundTest := findTest(testName, map[string]*TestResult{subTest.TestName: subTest}); foundTest != nil {
				return foundTest
			}
		}
	}

	return nil
}

func (p *parser) GetSummary() *Summary {
	return p.sum
}

type Action struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Elapsed float64   `json:"Elapsed"`
}

type TestResult struct {
	TestName    string
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime float64
	IsPassed    bool
	Output      []string
	Subtests    []*TestResult // This line is new
}

type PackageResult struct {
	PackageName string
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime float64
	IsPassed    bool
	Output      []string
	TestResults map[string]*TestResult // Change this from slice to map
}

type Summary struct {
	TotalPackages  int
	TotalPassed    int
	TotalFailed    int
	PackageResults []PackageResult
}
