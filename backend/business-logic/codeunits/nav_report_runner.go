package codeunits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hansjlachmann/openerp/backend/business-logic/tables"
	fcodeunits "github.com/hansjlachmann/openerp/backend/foundation/codeunits"
	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

// NavReportRunner - Codeunit 50022: NAV Report Runner
// Executes reports via external NAV proxy service with progress tracking
const NavReportRunnerID = 50022
const NavReportRunnerName = "nav-report-runner"

func init() {
	Register(NavReportRunnerID, NavReportRunnerName, NewNavReportRunner)
}

type NavReportRunner struct {
	db      database.Executor
	company string
	dbType  database.DBType
	client  *http.Client
}

// NewNavReportRunner creates a new instance of the codeunit
func NewNavReportRunner(db database.Executor, company string, dbType database.DBType) fcodeunits.Codeunit {
	return &NavReportRunner{
		db:      db,
		company: company,
		dbType:  dbType,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ID returns the codeunit ID
func (c *NavReportRunner) ID() int {
	return NavReportRunnerID
}

// Name returns the codeunit name
func (c *NavReportRunner) Name() string {
	return NavReportRunnerName
}

// SourceTable returns the table this codeunit operates on
func (c *NavReportRunner) SourceTable() string {
	return "Job_Queue"
}

// UsesProgress returns true - this codeunit uses progress dialog updates
func (c *NavReportRunner) UsesProgress() bool {
	return true
}

// StartJobRequest is the request body for starting a NAV job
type StartJobRequest struct {
	JobID     string `json:"JobId"`
	InputJSON string `json:"InputJson"`
}

// StartJobResponse is the response from the start job endpoint
type StartJobResponse struct {
	Result string `json:"result"`
}

// CheckJobResponse is the response from the check job endpoint
type CheckJobResponse struct {
	Result   string `json:"result"`
	Progress int    `json:"progress,omitempty"`
}

// GetJobPdfResponse is the response from the PDF endpoint
type GetJobPdfResponse struct {
	JobID    string `json:"JobId"`
	FileName string `json:"FileName"`
	Base64   string `json:"Base64"`
}

// Run executes the codeunit - calls external NAV service and tracks progress
func (c *NavReportRunner) Run(record interface{}) (fcodeunits.Result, error) {
	jobQueue := record.(*tables.JobQueue)

	// Get the dialog for progress updates
	dialog := fcodeunits.GetCurrentDialog()

	// NAV proxy service URL (hardcoded for customer environment)
	baseURL := "http://10.217.10.86:5009"

	// Generate a unique job ID (exactly 20 alphanumeric characters)
	// Format: YYMMDDHHMMSS (12 chars) + random alphanumeric (8 chars)
	jobID := generateJobID()

	// Build the input JSON from job queue parameters
	// For now, we expect the job queue description to contain the report ID
	inputJSON := fmt.Sprintf(`{"reportId":%s,"format":"PDF"}`, jobQueue.Description.String())

	// Update progress: Starting
	if dialog != nil {
		dialog.UpdateWithMessage(1, 0, "Starting report generation...")
	}

	// Step 1: Start the job via POST
	startReq := StartJobRequest{
		JobID:     jobID,
		InputJSON: inputJSON,
	}

	reqBody, err := json.Marshal(startReq)
	if err != nil {
		return fcodeunits.Error("Failed to marshal request: " + err.Error()), nil
	}

	resp, err := c.client.Post(
		baseURL+"/api/nav/startjob",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fcodeunits.Error("Failed to start job: " + err.Error()), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fcodeunits.Error(fmt.Sprintf("Start job failed (status %d): %s", resp.StatusCode, string(body))), nil
	}

	// Update progress: Job started
	if dialog != nil {
		dialog.UpdateWithMessage(1, 5, "Report job started, waiting for completion...")
	}

	// Step 2: Poll for progress every 5 seconds
	pollInterval := 5 * time.Second
	maxWaitTime := 15 * time.Minute
	startTime := time.Now()

	for {
		// Check if we've exceeded max wait time
		if time.Since(startTime) > maxWaitTime {
			return fcodeunits.Error("Report generation timed out after 15 minutes"), nil
		}

		// Wait before polling
		time.Sleep(pollInterval)

		// Check job status
		checkResp, err := c.client.Get(baseURL + "/api/nav/checkjob/" + jobID)
		if err != nil {
			// Log error but continue polling
			if dialog != nil {
				dialog.UpdateWithMessage(1, -1, "Connection error, retrying...")
			}
			continue
		}

		body, err := io.ReadAll(checkResp.Body)
		checkResp.Body.Close()

		if err != nil {
			continue
		}

		var checkResult CheckJobResponse
		if err := json.Unmarshal(body, &checkResult); err != nil {
			// Try to parse the simple "Progress X" format
			checkResult = parseProgressResponse(string(body))
		}

		// Update progress bar
		progress := checkResult.Progress
		if progress == 0 {
			// Try to extract progress from result string "Progress X"
			progress = extractProgress(checkResult.Result)
		}

		if dialog != nil {
			msg := fmt.Sprintf("Generating report... %d%%", progress)
			if progress >= 100 {
				msg = "Report complete, downloading..."
			}
			dialog.UpdateWithMessage(1, progress, msg)
		}

		// Check if job is complete
		if progress >= 100 || checkResult.Result == "Completed" {
			// Job complete - fetch the PDF from dedicated endpoint
			if dialog != nil {
				dialog.UpdateWithMessage(1, 100, "Downloading PDF...")
			}

			pdfResp, err := c.client.Get(baseURL + "/api/nav/job/" + jobID + "/pdf")
			if err != nil {
				return fcodeunits.Error("Failed to download PDF: " + err.Error()), nil
			}
			defer pdfResp.Body.Close()

			if pdfResp.StatusCode != http.StatusOK {
				pdfBody, _ := io.ReadAll(pdfResp.Body)
				return fcodeunits.Error(fmt.Sprintf("PDF download failed (status %d): %s", pdfResp.StatusCode, string(pdfBody))), nil
			}

			pdfBody, err := io.ReadAll(pdfResp.Body)
			if err != nil {
				return fcodeunits.Error("Failed to read PDF response: " + err.Error()), nil
			}

			var pdfResult GetJobPdfResponse
			if err := json.Unmarshal(pdfBody, &pdfResult); err != nil {
				return fcodeunits.Error("Failed to parse PDF response: " + err.Error()), nil
			}

			return fcodeunits.Result{
				Success: true,
				Message: "Report generated successfully",
				Data: map[string]interface{}{
					"pdf":      pdfResult.Base64,
					"filename": pdfResult.FileName,
				},
			}, nil
		}
	}
}

// parseProgressResponse parses the response body into CheckJobResponse
func parseProgressResponse(body string) CheckJobResponse {
	var result CheckJobResponse

	// Try JSON parse first
	if err := json.Unmarshal([]byte(body), &result); err == nil {
		return result
	}

	// Try to extract from {"result": "Progress X"} format
	var simpleResult struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &simpleResult); err == nil {
		result.Result = simpleResult.Result
		result.Progress = extractProgress(simpleResult.Result)
	}

	return result
}

// extractProgress extracts the progress number from "Progress X" string
func extractProgress(s string) int {
	re := regexp.MustCompile(`Progress\s+(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		if p, err := strconv.Atoi(matches[1]); err == nil {
			return p
		}
	}
	return 0
}

// generateJobID generates a unique 20-character alphanumeric job ID
// Format: YYMMDDHHMMSS (12 chars) + random alphanumeric (8 chars)
func generateJobID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	now := time.Now()
	// YYMMDDHHMMSS = 12 characters
	timestamp := now.Format("060102150405")

	// Generate 8 random alphanumeric characters
	random := make([]byte, 8)
	for i := range random {
		random[i] = charset[rand.Intn(len(charset))]
	}

	return timestamp + string(random)
}
