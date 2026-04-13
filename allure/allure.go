package allure

import "github.com/google/uuid"

// Attachment - is an implementation of the attachments to the report in allure. It is most often used to contain
// screenshots, responses, files and other data obtained during the test.
type Attachment struct {
	Name   string   `json:"name,omitempty"`   // Attachment name
	Source string   `json:"source,omitempty"` // Path to the Attachment file (name)
	Type   MimeType `json:"type,omitempty"`   // Mime-type of the Attachment
}

// MimeType is Attachment's mime type.
// See more: https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
type MimeType string

const (
	Text MimeType = "text/plain"
)

type StatusDetail struct {
	Message string `json:"message,omitempty"` // Abridged version of the message
	Trace   string `json:"trace,omitempty"`   // Full message
}

type Status string

const (
	Passed  Status = "passed"
	Failed  Status = "failed"
	Skipped Status = "skipped"
	Broken  Status = "broken"
	Unknown Status = "unknown"
)

func (s Status) Less(other Status) bool {
	// unknown < passed < failed < skipped < broken
	switch s {
	case Broken:
		return false
	case Skipped:
		return other == Broken
	case Failed:
		return other == Skipped || other == Broken
	case Passed:
		return other == Failed || other == Skipped || other == Broken
	default: // Unknown
		return other != Unknown
	}
}

func (s Status) String() string {
	return string(s)
}

type Stage string

const (
	Scheduled   Stage = "scheduled"
	Running     Stage = "running"
	Finished    Stage = "finished"
	Pending     Stage = "pending"
	Interrupted Stage = "interrupted"
)

// Label is the implementation of the label.
// A label is an entity used by Allure to make metrics and grouping of tests.
type Label struct {
	Name  string `json:"name"`  // Label's name
	Value string `json:"value"` // Label's value
}

// Parameter is an implementation of the Parameter entity,
// which Allure uses as additional information describing the test Step
// (for example - request host or server address)
type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (p Parameter) String() string {
	return p.Name + "=" + p.Value
}

///

type Step struct {
	Name          string       `json:"name,omitempty"`
	Status        Status       `json:"status,omitempty"`
	StatusDetails StatusDetail `json:"statusDetails,omitempty"`
	Start         int64        `json:"start,omitempty"`
	Stop          int64        `json:"stop,omitempty"`
	Steps         []Step       `json:"steps,omitempty"`
	Parameters    []Parameter  `json:"parameters,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
}

// Link is an implementation of the Link entity used by Allure to specify the links needed for test reports.
// Such as:
// - A link to a task in Issue tracker.
// - A link to a test case in the TMS
// - Any other link (e.g. a link to an environment pod)
type Link struct {
	Name string `json:"name"` // Link name
	Type string `json:"type"` // Link's Type (issue, test case or any other)
	URL  string `json:"url"`  // Link URL
}

type Result struct {
	Name          string       `json:"name,omitempty"`          // Test name
	FullName      string       `json:"fullName,omitempty"`      // Full path to the test
	Stage         Stage        `json:"stage,omitempty"`         // Stage of test execution
	Status        Status       `json:"status,omitempty"`        // Status of the test execution
	StatusDetails StatusDetail `json:"statusDetails,omitempty"` // Details about the test (for example, errors during test execution will be recorded here)
	Start         int64        `json:"start,omitempty"`         // Start of test execution "in the UNIX timestamp format"
	Stop          int64        `json:"stop,omitempty"`          // End of test execution "in the UNIX timestamp format"
	UUID          uuid.UUID    `json:"uid,omitempty"`           // Unique test ID
	Description   string       `json:"description,omitempty"`   // Test description

	// "Two runs of the same test with the same set of parameters will always have the same historyId."
	HistoryID string `json:"historyId,omitempty"` // ID in the allure history

	// "Two runs of the same test will always have the same testCaseId."
	TestCaseID string `json:"testCaseId,omitempty"` // ID of the test case (based on the hash of the full call)

	Parameters  []Parameter  `json:"parameters,omitempty"`  // Test case parameters
	Labels      []Label      `json:"labels,omitempty"`      // Array of labels
	Links       []Link       `json:"links,omitempty"`       // Array of references
	Steps       []Step       `json:"steps,omitempty"`       // Array of steps
	Attachments []Attachment `json:"attachments,omitempty"` // Test case attachments

	ContinuousLog string `json:"-"` // Continuous log of the test execution (if any)
}
