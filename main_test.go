package main

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/castai/gotest2allure/allure"
)

func Test_extractParamsViaRegex(t *testing.T) {
	const str = "controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true"

	fields := extractParamsViaRegex(str)

	require.Len(t, fields, 2)
	require.Contains(t, fields, allure.Parameter{Name: "client", Value: "false"})
	require.Contains(t, fields, allure.Parameter{Name: "tcp", Value: "true"})
}

func Test_withoutParams(t *testing.T) {
	const str = "controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true"

	got := withoutParams(str)

	require.Equal(t, "controller_orchestrates_echo_migration,_to_the_same_node_as", got)
}

func Test_parseTestName(t *testing.T) {
	const str = "TestTCPMigrations/controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true/test-app_test_finished_lease_created"

	suite, test, step := parseTestName(str)

	require.Equal(t, "TestTCPMigrations", suite)
	require.Equal(t, "controller_orchestrates_echo_migration,_to_the_same_node_as_client=false_with_tcp=true", test)
	require.Equal(t, "test-app_test_finished_lease_created", step)
}

///

func TestFull(t *testing.T) {
	const in = "./testdata/log.jsonl"

	dir := t.TempDir()

	err := _main(in, dir, []string{"provider=kind", "pr_title=feat(e2e): upload test results to Allure TestOps (LIVE-192)"})
	require.NoError(t, err)

	const envPropsFile = "environment.properties"

	t.Run(envPropsFile, func(t *testing.T) {
		r := require.New(t)

		r.FileExists(path.Join(dir, envPropsFile))

		content, err := os.ReadFile(path.Join(dir, envPropsFile))
		r.NoError(err)

		r.Contains(string(content), "provider=kind")
		r.Contains(string(content), "pr_title=feat(e2e): upload test results to Allure TestOps (LIVE-192)")
	})

	t.Run("tests", func(t *testing.T) {
		r := require.New(t)

		filesD, err := os.ReadDir(dir)
		r.NoError(err)
		files := lo.Without(Map(filesD, os.DirEntry.Name), envPropsFile)

		r.Len(files, 5*2) // 5 test files, each with an attachment and a result file

		results := make(map[string]allure.Result)

		for _, file := range files {
			if !strings.HasSuffix(file, "-result.json") {
				continue
			}

			var result allure.Result
			f, err := os.Open(path.Join(dir, file))
			r.NoError(err)
			err = json.NewDecoder(f).Decode(&result)
			r.NoError(err)

			results[result.FullName] = result
		}

		const sampleTestName = "TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=true"

		r.Len(results, 5)
		r.ElementsMatch(lo.Keys(results), []string{
			"TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=false#01",
			sampleTestName,
			"TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=false#02",
			"TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=false",
			"TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=false#03",
		})

		t.Run("the sample test", func(t *testing.T) {
			r := require.New(t)

			result, ok := results[sampleTestName]
			r.True(ok)

			r.Equal("TestCounterMigrations/controller_orchestrates_counter_migration", result.Name)
			r.Equal(sampleTestName, result.FullName)
			r.Equal(allure.Passed, result.Status)
			r.Equal(allure.Finished, result.Stage)
			r.ElementsMatch(result.Parameters, []allure.Parameter{
				{Name: "tcp", Value: "false"},
				{Name: "files", Value: "true"},
			})
			r.ElementsMatch(result.Labels, []allure.Label{
				{Name: "provider", Value: "kind"},
				{Name: "Provider", Value: "kind"},
				{Name: "pr_title", Value: "feat(e2e): upload test results to Allure TestOps (LIVE-192)"},
				{Name: "Pr_title", Value: "feat(e2e): upload test results to Allure TestOps (LIVE-192)"},
			})

			r.Len(result.Steps, 9)
			stepsNames := Map(result.Steps, func(s allure.Step) string { return s.Name })
			r.ElementsMatch(stepsNames, []string{
				"controller_is_ready",
				"test-app_test_finished_lease_created",
				"pod_for_migration_created",
				"start_pod_migration",
				"migration_completed",
				"wait_for_migrated_pod_desired_state",
				"notify_test-app_test_finished",
				"wait_for_migrated_pod_desired_state#01",
				"numbers_match",
			})
		})
	})
}
