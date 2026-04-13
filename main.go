package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/pflag"

	"github.com/castai/gotest2allure/allure"
)

var (
	input     = pflag.StringP("input", "i", "", "input file")
	outputDir = pflag.StringP("outputDir", "o", "", "output dir")

	tags = pflag.StringSliceP("tags", "e", nil, "tags (key=value) to add to the results")
)

func main() {
	pflag.Parse()

	if *input == "" {
		pflag.Usage()
		os.Exit(1)
	}
	if *outputDir == "" {
		slog.Error("output directory is required")
		pflag.Usage()
		os.Exit(1)
	}

	if err := _main(*input, *outputDir, *tags); err != nil {
		slog.Error("error", "err", err)
		os.Exit(1)
	}
}

func _main(inputFile, outputDir string, env []string) error {
	in, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer in.Close()

	l := slog.Default()

	results := GoTestJsonLinesToAllure{results: make(map[string]allure.Result)}
	results.WithEnvironment(env)

	sc := bufio.NewScanner(in)
	// Increase scanner buffer for long lines in test output.
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			continue
		}
		if line[0] != '{' && line[0] != '[' {
			l.Error("not a json line", "line", line)
			continue
		}

		var entry GoTestLogLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			l.Error("failed to unmarshal json", "line", line, "err", err.Error())
			continue
		}

		results.Add(l, entry)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := results.WriteEnvironment(outputDir); err != nil {
		return err
	}
	return results.WriteResults(outputDir)
}

///

/*
`go test -json` single line example:

	{
	  "Time": "2025-07-09T21:12:27.650192645+03:00",
	  "Action": "pass",
	  "Package": "github.com/castai/workload-autoscaler/tests/scenarios",
	  "Test": "TestTCPMigrations/controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true/test-app_test_finished_lease_created",
	  "Elapsed": 0.01
	}
*/
type GoTestLogLine struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`

	Elapsed *float64 `json:"Elapsed,omitempty"` // action != "output"
	// Output is a raw string, it has `\n` and `\t` in it.
	Output string `json:"Output,omitempty"` // action == "output"
}

func parseTestName(name string) (suite, test, step string) {
	parts := strings.Split(name, "/")
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return parts[0], "", ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], strings.Join(parts[2:], "/")
	}
}

func actionToStatus(action string) allure.Status {
	var st allure.Status
	switch action {
	case "pass":
		st = allure.Passed
	case "fail":
		st = allure.Failed
	case "skip":
		st = allure.Skipped
	default:
		st = allure.Unknown
	}
	return st
}

func appendOutput(entry GoTestLogLine, dst *string) {
	if entry.Output != "" {
		*dst += entry.Time.Format(RFC3339Milli) + "\t" + entry.Output
	}
}

///

/*
parameters section is about extracting:
`client=false` and `tcp=true`
from test name like
`controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true`
*/

const punctuation = `,;._/`

var fieldRe = regexp.MustCompile(fmt.Sprintf("_?(?:with_)?([^%s]+)=([^%s]+)", punctuation, punctuation))

func extractParamsViaRegex(str string) []allure.Parameter {
	var parameters []allure.Parameter
	fields := fieldRe.FindAllStringSubmatch(str, -1)
	for _, field := range fields {
		if len(field) != 3 {
			continue
		}
		parameters = append(parameters, allure.Parameter{
			Name:  field[1],
			Value: field[2],
		})
	}
	return parameters
}

func withoutParams(str string) string {
	// remove all parameters from the string
	return strings.TrimRight(fieldRe.ReplaceAllString(str, ""), punctuation)
}

type GoTestJsonLinesToAllure struct {
	results map[string]allure.Result
	env     map[string]string
}

func (g *GoTestJsonLinesToAllure) Add(logger *slog.Logger, entry GoTestLogLine) {
	suiteID, runID, stepID := parseTestName(entry.Test)
	if runID == "" {
		return
	}

	testName := suiteID + "/" + runID
	testParameters := extractParamsViaRegex(runID)
	testNameWithoutParams := withoutParams(testName)
	resultID := hash(
		testNameWithoutParams,
		strings.Join(Map(testParameters, allure.Parameter.String), ","),
	)

	if _, ok := g.results[resultID]; !ok {
		result := allure.Result{
			Stage:      allure.Finished,
			Name:       testNameWithoutParams,
			FullName:   testName,
			Parameters: testParameters,
			Start:      entry.Time.UnixMilli(),
			Stop:       entry.Time.UnixMilli(),
			UUID:       uuid.New(),

			HistoryID:  resultID,
			TestCaseID: hash(testNameWithoutParams),
		}
		for k, v := range g.env {
			result.Labels = append(result.Labels, allure.Label{Name: k, Value: v})
			result.Labels = append(result.Labels, allure.Label{Name: strings.ToUpper(k[:1]) + k[1:], Value: v})
		}

		g.results[resultID] = result
	}
	result := g.results[resultID]
	defer func() { g.results[resultID] = result }()

	result.Start = min(entry.Time.UnixMilli(), result.Start)

	deducedStatus := actionToStatus(entry.Action)
	if deducedStatus != allure.Unknown {
		if result.Status.Less(deducedStatus) {
			result.Status = deducedStatus
		} else {
			if result.Status != deducedStatus {
				logger.Warn("inconsistent status for result", "result", testName, "oldStatus", result.Status, "newStatus", deducedStatus)
			}
		}
	}

	if stepID != "" {
		g.addStep(entry, &result, stepID)
	} else {
		appendOutput(entry, &result.StatusDetails.Message)
		if entry.Elapsed != nil {
			result.Stop = result.Start + int64(*entry.Elapsed*1000)
		}
	}
	appendOutput(entry, &result.ContinuousLog)
}

func (g *GoTestJsonLinesToAllure) addStep(entry GoTestLogLine, result *allure.Result, stepID string) {
	idx := -1
	for i, s := range result.Steps {
		if s.Name == stepID {
			idx = i
			break
		}
	}

	var step allure.Step
	if idx == -1 {
		step = allure.Step{
			Name:  stepID,
			Start: entry.Time.UnixMilli(),
		}
		result.Steps = append(result.Steps, step)
		idx = len(result.Steps) - 1
	} else {
		step = result.Steps[idx]
	}
	defer func() { result.Steps[idx] = step }()

	step.Status = result.Status

	appendOutput(entry, &step.StatusDetails.Message)
	if entry.Elapsed != nil {
		step.Stop = step.Start + int64(*entry.Elapsed*1000)
	}
}

func (g *GoTestJsonLinesToAllure) WriteResults(dir string) error {
	for _, result := range g.results {
		if result.ContinuousLog != "" {
			attachFile := result.UUID.String() + "-attachment.txt"

			logFile, err := os.Create(path.Join(dir, attachFile))
			if err != nil {
				return err
			}
			defer logFile.Close()

			if _, err := logFile.WriteString(result.ContinuousLog); err != nil {
				return err
			}

			result.Attachments = append(result.Attachments, allure.Attachment{
				Name:   "Continuous Log",
				Type:   allure.Text,
				Source: attachFile,
			})
		}

		out, err := os.Create(path.Join(dir, result.UUID.String()+"-result.json"))
		if err != nil {
			return err
		}
		defer out.Close()

		if err := json.NewEncoder(out).Encode(result); err != nil {
			return err
		}
	}
	return nil
}

///

func (g *GoTestJsonLinesToAllure) WithEnvironment(envs []string) {
	fields := make(map[string][]string)

	for _, field := range envs {
		fs := strings.SplitN(field, "=", 2)

		key := fs[0]
		var val string
		if len(fs) == 1 {
			val = ""
		} else {
			val = fs[1]
		}
		fields[key] = append(fields[key], val)
	}

	env := make(map[string]string)
	for key, values := range fields {
		env[key] = strings.Join(lo.Uniq(values), ",")
	}
	g.env = env
}

// WriteEnvironment sets 'variables' table content for a single Allure launch.
// They don't seem to have any effect on the actual results.
func (g *GoTestJsonLinesToAllure) WriteEnvironment(dir string) error {
	f, err := os.Create(path.Join(dir, "environment.properties"))
	if err != nil {
		return err
	}
	defer f.Close()

	for key, value := range g.env {
		if _, err := fmt.Fprintf(f, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}
