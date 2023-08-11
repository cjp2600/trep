# TREP - Testing Results Enhanced Presentation

**TREP** is a CLI tool designed to provide an intuitive and human-readable interface for Go's built-in testing functionality (`go test`).

The motivation behind TRAP's creation arose from a clear need in CI/CD pipelines to make test results more accessible and actionable, not only for developers but also for non-technical team members such as managers and QA specialists. Traditionally, the output from `go test` is highly technical and can be overwhelming for those not deeply familiar with Go or software development in general.

TRAP serves as a wrapper around `go test`, executing the tests and capturing their output, but then goes further by processing this output to present it in a more user-friendly manner. This includes color-coding different types of results (like passes, failures, and skips), providing clear and concise summaries of the test outcomes, and even offering more detailed analysis where necessary.

This level of presentation and analysis has several benefits. Developers can quickly and easily identify problem areas in their code, while non-technical team members gain a clearer understanding of the test results, allowing them to better track project progress and quality.

In addition to facilitating local development, TRAP shines in a CI/CD pipeline. Its readable output can be captured and shared as part of build logs, reports, or notifications, providing all stakeholders with the crucial information they need in a form they can understand.

In essence, TRAP embodies the philosophy that effective communication and understanding between all team members - technical or not - is essential to the successful delivery of a software project. By making test results more accessible and actionable, TRAP helps to bridge the gap between developers, managers, and QA specialists, fostering a more collaborative and productive team environment.

## Installation

```shell
go install github.com/cjp2600/trep@latest
```

## `exec` Command

### Description

The `exec` command is used to execute Go tests and format their output. It provides various flags to modify its behavior, such as filtering only failed tests, generating reports, and configuring the run mode.

### Usage

```shell
./trep exec "[golang test command]"
```

#### Options

- **`-f`, `--only-fail`**: If set, only failed tests will be displayed. Default is `false`.
- **`-r`, `--report`**: If set, a report will be generated. Default is `false`.
- **`-p`, `--report-path`**: Specifies the path where the report will be saved. Default is the current directory (`./`).
- **`-m`, `--mode`**: Specifies the run mode. Available options are `'cli'`, `'ci'`. Default is `'cli'`.
- **`-n`, `--report-name`**: Allows you to provide a custom name for the report. Example: `'report'`.

### Examples

1. **Executing Tests and Displaying Only Failures**

   Execute the Go tests and display only the failed tests:

```shell
./trep exec "go test ./... -v -cover" --only-fail
```

2. **Executing Tests and Generating a Report**

   Execute the Go tests and generate a report:

```shell
./trep exec "go test ./... -v -cover" --report
```

3. Specifying a Custom Report Path and Name

Execute the Go tests and save the report in a custom directory with a custom name:

```shell
./trep exec "go test ./... -v -cover" --report --report-path ./reports --report-name report
```

4. **Running in CI Mode**

   Execute the Go tests in CI mode:

```shell
./trep exec "go test ./... -v -cover" --mode ci
```
**Notes**

Make sure that the specified report path exists, or an error may occur when trying to save the report.

