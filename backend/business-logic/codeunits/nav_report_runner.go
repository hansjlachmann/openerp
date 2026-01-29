package codeunits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
			Timeout: 10 * time.Minute, // Long timeout for report generation
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
	_ = record.(*tables.JobQueue) // Not used for now - report ID is hardcoded

	// Get the dialog for progress updates
	dialog := fcodeunits.GetCurrentDialog()

	// NAV proxy service URL (hardcoded for customer environment)
	baseURL := "http://10.217.10.86:5009"

	// Generate a unique job ID (exactly 20 alphanumeric characters)
	// Format: YYMMDDHHMMSS (12 chars) + random alphanumeric (8 chars)
	jobID := generateJobID()
	log.Printf("[NavReportRunner] Generated job ID: %s", jobID)

	// Build the input JSON with hardcoded report ID (for now)
	reportID := 121
	inputJSON := fmt.Sprintf(`{"reportId":%d,"format":"PDF"}`, reportID)
	log.Printf("[NavReportRunner] Input JSON: %s", inputJSON)

	// Update progress: Starting
	if dialog != nil {
		dialog.UpdateWithMessage(1, 0, "Starting report generation...")
	}

	// Step 1: Start the job via POST (fire-and-forget)
	// The NAV service POST is synchronous and can take minutes, so we don't wait for it
	startReq := StartJobRequest{
		JobID:     jobID,
		InputJSON: inputJSON,
	}

	reqBody, err := json.Marshal(startReq)
	if err != nil {
		log.Printf("[NavReportRunner] Failed to marshal request: %v", err)
		return fcodeunits.Error("Failed to marshal request: " + err.Error()), nil
	}

	startURL := baseURL + "/api/nav/startjob"
	log.Printf("[NavReportRunner] POST %s (fire-and-forget) with body: %s", startURL, string(reqBody))

	// Fire POST in background goroutine - don't wait for response
	go func() {
		resp, err := c.client.Post(
			startURL,
			"application/json",
			bytes.NewReader(reqBody),
		)
		if err != nil {
			log.Printf("[NavReportRunner] Background POST failed: %v", err)
			return
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[NavReportRunner] Background POST response (status %d): %s", resp.StatusCode, string(respBody))
	}()

	// Update progress: Job started
	if dialog != nil {
		dialog.UpdateWithMessage(1, 5, "Report job started, waiting for completion...")
	}

	// Step 2: Poll PDF endpoint every 5 seconds to check if ready
	pollInterval := 5 * time.Second
	maxWaitTime := 15 * time.Minute
	startTime := time.Now()
	firstPoll := true

	log.Printf("[NavReportRunner] Starting PDF polling for job %s", jobID)

	for {
		// Check if we've exceeded max wait time
		if time.Since(startTime) > maxWaitTime {
			log.Printf("[NavReportRunner] Job %s timed out", jobID)
			return fcodeunits.Error("Report generation timed out after 15 minutes"), nil
		}

		// Wait before polling (skip wait on first poll)
		if !firstPoll {
			time.Sleep(pollInterval)
		}
		firstPoll = false

		// Check if PDF is ready
		pdfURL := baseURL + "/api/nav/job/" + jobID + "/pdf"
		log.Printf("[NavReportRunner] Polling PDF: GET %s", pdfURL)

		pdfResp, err := c.client.Get(pdfURL)
		if err != nil {
			log.Printf("[NavReportRunner] PDF poll error: %v", err)
			if dialog != nil {
				dialog.UpdateWithMessage(1, -1, "Connection error, retrying...")
			}
			continue
		}

		pdfBody, err := io.ReadAll(pdfResp.Body)
		pdfResp.Body.Close()

		if err != nil {
			log.Printf("[NavReportRunner] Failed to read PDF response body: %v", err)
			continue
		}

		log.Printf("[NavReportRunner] PDF poll response (status %d): %d bytes", pdfResp.StatusCode, len(pdfBody))

		// If PDF is ready (status 200), return it
		if pdfResp.StatusCode == http.StatusOK {
			log.Printf("[NavReportRunner] PDF is ready for job %s", jobID)

			var pdfResult GetJobPdfResponse
			if err := json.Unmarshal(pdfBody, &pdfResult); err != nil {
				log.Printf("[NavReportRunner] Failed to parse PDF response: %v", err)
				return fcodeunits.Error("Failed to parse PDF response: " + err.Error()), nil
			}

			if dialog != nil {
				dialog.UpdateWithMessage(1, 100, "PDF ready, downloading...")
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

		// PDF not ready - get progress from checkjob endpoint
		checkURL := baseURL + "/api/nav/checkjob/" + jobID
		log.Printf("[NavReportRunner] PDF not ready, checking progress: GET %s", checkURL)

		checkResp, err := c.client.Get(checkURL)
		if err != nil {
			log.Printf("[NavReportRunner] Progress check error: %v", err)
			continue
		}

		checkBody, err := io.ReadAll(checkResp.Body)
		checkResp.Body.Close()

		if err != nil {
			continue
		}

		log.Printf("[NavReportRunner] Progress response: %s", string(checkBody))

		// Parse progress
		var checkResult CheckJobResponse
		if err := json.Unmarshal(checkBody, &checkResult); err != nil {
			checkResult = parseProgressResponse(string(checkBody))
		}

		progress := checkResult.Progress
		if progress == 0 {
			progress = extractProgress(checkResult.Result)
		}

		// Cap progress at 99 until PDF is actually ready
		if progress >= 100 {
			progress = 99
		}

		log.Printf("[NavReportRunner] Progress: %d%%", progress)

		if dialog != nil {
			msg := fmt.Sprintf("Generating report... %d%%", progress)
			dialog.UpdateWithMessage(1, progress, msg)
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
// Handles various formats: "Progress 45", "Progress: 45", "progress 45", etc.
func extractProgress(s string) int {
	// Try multiple patterns
	patterns := []string{
		`[Pp]rogress[:\s]+(\d+)`,  // "Progress 45" or "Progress: 45"
		`(\d+)\s*%`,               // "45%" or "45 %"
		`(\d+)`,                   // Just a number
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(s)
		if len(matches) >= 2 {
			if p, err := strconv.Atoi(matches[1]); err == nil {
				return p
			}
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
