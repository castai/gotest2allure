# gotest2allure

`gotest2allure` consumes `go test -json` output for e2e tests and transforms it into Allure results.

## Test Name Structure

Given a `go test` name like:

```
TestCounterMigrations/controller_orchestrates_counter_migration,_with_tcp=false,_with_files=true/controller_is_ready
```

The tool parses it as:

| Part | Example | Role |
|------|---------|------|
| Suite | `TestCounterMigrations` | Top-level test function |
| Test | `controller_orchestrates_counter_migration` | Sub-test name; parameters (`tcp=false`, `files=true`) are extracted automatically |
| Step | `controller_is_ready` | Individual step within the test |

## Installation

```bash
go install github.com/castai/gotest2allure@latest
```

## Usage

```bash
go test -json ./... | gotest2allure -i <input-file> -o <output-dir> [-e key=value ...]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--input` | `-i` | Path to the `go test -json` output file |
| `--outputDir` | `-o` | Directory where Allure result files are written |
| `--tags` | `-e` | One or more `key=value` pairs added as labels/tags to every result |

### Example

```bash
go test -json ./tests/... > results.json
gotest2allure -i results.json -o allure-results -e provider=eks -e env=staging
```

## Notes

- Labels are not just "tags" — the label name has special meaning and must be manually enabled / supported by Allure TestOps.
- To set a custom field or an environment item, pass it as a label with `--tags`; the `name` part maps to the field name in TestOps.
- Tags are shared among runs of the same test case. Setting different tag values across runs produces a changelog entry for each change, for example:

  ```
  2025-07-11 04:23:09  _system  Added tag    provider=eks
  2025-07-11 04:23:09  _system  Removed tag  provider=gke
  2025-07-11 04:19:55  _system  Added tag    provider=gke
  2025-07-11 04:19:55  _system  Removed tag  provider=kind
  ```

## References

- [Allure test result file format](https://allurereport.org/docs/how-it-works-test-result-file/#test-result-file)
- [Standard labels used in Allure Framework](https://docs.qameta.io/allure-testops/briefly/test-cases/labels/#list-of-standard-labels-used-in-allure-framework) (see also the full "Test cases" section in the sidebar)
