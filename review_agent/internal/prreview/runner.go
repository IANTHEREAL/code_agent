package prreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	b "review_agent/internal/brain"
	"review_agent/internal/logx"
	"review_agent/internal/streaming"
	t "review_agent/internal/tools"
)

const (
	statusClean         = "clean"
	statusIssues        = "issues_found"
	commentConfirmed    = "confirmed"
	commentUnresolved   = "unresolved"
	defaultMaxExchanges = 1
)

// Options configures the PR review workflow.
type Options struct {
	Task            string
	ProjectName     string
	ParentBranchID  string
	WorkspaceDir    string
	MaxExchangeRuns int
}

// Result captures the high-level outcome plus supporting artifacts.
type Result struct {
	Task           string        `json:"task"`
	Status         string        `json:"status"`
	Summary        string        `json:"summary"`
	ReviewerLogs   []ReviewerLog `json:"reviewer_logs"`
	Issues         []IssueReport `json:"issues"`
	StartBranchID  string        `json:"start_branch_id,omitempty"`
	LatestBranchID string        `json:"latest_branch_id,omitempty"`
}

// ReviewerLog records the raw output from each review_code run.
type ReviewerLog struct {
	BranchID string `json:"branch_id"`
	Report   string `json:"report"`
}

// Transcript records a codex agent's reasoning for an issue confirmation attempt.
type Transcript struct {
	Agent    string `json:"agent"`
	Round    int    `json:"round"`
	BranchID string `json:"branch_id,omitempty"`
	Text     string `json:"text"`
}

// IssueReport stores the consensus outcome for a single ISSUE block.
type IssueReport struct {
	IssueText          string     `json:"issue_text"`
	Status             string     `json:"status"`
	Alpha              Transcript `json:"alpha"`
	Beta               Transcript `json:"beta"`
	ExchangeRounds     int        `json:"exchange_rounds"`
	VerdictExplanation string     `json:"verdict_explanation,omitempty"`
}

// Runner executes the two-phase PR review workflow.
type Runner struct {
	brain    *b.LLMBrain
	handler  *t.ToolHandler
	opts     Options
	streamer *streaming.JSONStreamer
	events   *eventHelper
}

// NewRunner validates options and constructs a workflow runner.
func NewRunner(brain *b.LLMBrain, handler *t.ToolHandler, streamer *streaming.JSONStreamer, opts Options) (*Runner, error) {
	if brain == nil {
		return nil, errors.New("brain is required")
	}
	if handler == nil {
		return nil, errors.New("tool handler is required")
	}
	opts.Task = strings.TrimSpace(opts.Task)
	opts.ProjectName = strings.TrimSpace(opts.ProjectName)
	opts.ParentBranchID = strings.TrimSpace(opts.ParentBranchID)
	opts.WorkspaceDir = strings.TrimSpace(opts.WorkspaceDir)
	if opts.Task == "" {
		return nil, errors.New("task description is required")
	}
	if opts.ProjectName == "" {
		return nil, errors.New("project name is required")
	}
	if opts.ParentBranchID == "" {
		return nil, errors.New("parent branch id is required")
	}
	if opts.MaxExchangeRuns <= 0 {
		opts.MaxExchangeRuns = defaultMaxExchanges
	}
	return &Runner{
		brain:    brain,
		handler:  handler,
		opts:     opts,
		streamer: streamer,
		events:   newEventHelper(streamer),
	}, nil
}

// Run executes the workflow and returns the structured result.
func (r *Runner) Run() (*Result, error) {
	logx.Infof("Starting PR review workflow for parent %s", r.opts.ParentBranchID)
	reviewLog, err := r.runSingleReview()
	if err != nil {
		return nil, err
	}

	result := &Result{
		Task:         r.opts.Task,
		ReviewerLogs: []ReviewerLog{reviewLog},
		Issues:       []IssueReport{},
	}

	if strings.TrimSpace(reviewLog.Report) == "" {
		result.Status = statusClean
		result.Summary = "Clean PR: review_code reported no blocking P0/P1 issue."
		r.attachBranchRange(result)
		return result, nil
	}

	issueText := reviewLog.Report
	report, err := r.confirmIssue(issueText)
	if err != nil {
		return nil, err
	}
	result.Issues = append(result.Issues, report)
	confirmed, unresolved := summarizeIssueCounts(result.Issues)
	result.Status = statusIssues
	result.Summary = fmt.Sprintf("Identified %d P0/P1 issue (%d confirmed, %d unresolved).", len(result.Issues), confirmed, unresolved)
	r.attachBranchRange(result)
	return result, nil
}

func (r *Runner) attachBranchRange(res *Result) {
	if res == nil {
		return
	}
	if r.handler == nil {
		return
	}
	lineage := r.handler.BranchRange()
	if start := lineage["start_branch_id"]; start != "" {
		res.StartBranchID = start
	}
	if latest := lineage["latest_branch_id"]; latest != "" {
		res.LatestBranchID = latest
	}
}

func (r *Runner) runSingleReview() (ReviewerLog, error) {
	prompt := buildReviewPrompt(r.opts.Task)
	data, err := r.executeAgent("review_code", prompt)
	if err != nil {
		return ReviewerLog{}, err
	}
	branchID := stringField(data, "branch_id")
	reviewLog := strings.TrimSpace(stringField(data, "review_report"))
	if reviewLog == "" {
		return ReviewerLog{}, fmt.Errorf("review_code did not include code_review.log contents")
	}
	return ReviewerLog{
		BranchID: branchID,
		Report:   reviewLog,
	}, nil
}

func (r *Runner) confirmIssue(issueText string) (IssueReport, error) {
	alpha, err := r.runCodex("codex-alpha", issueText, "", 1)
	if err != nil {
		return IssueReport{}, err
	}
	beta, err := r.runCodex("codex-beta", issueText, "", 1)
	if err != nil {
		return IssueReport{}, err
	}

	verdict, err := r.checkConsensus(issueText, alpha, beta)
	if err != nil {
		return IssueReport{}, err
	}

	exchanges := 0
	for !verdict.Agree && exchanges < r.opts.MaxExchangeRuns {
		exchanges++
		alpha, err = r.runCodex("codex-alpha", issueText, beta.Text, alpha.Round+1)
		if err != nil {
			return IssueReport{}, err
		}
		beta, err = r.runCodex("codex-beta", issueText, alpha.Text, beta.Round+1)
		if err != nil {
			return IssueReport{}, err
		}
		verdict, err = r.checkConsensus(issueText, alpha, beta)
		if err != nil {
			return IssueReport{}, err
		}
	}

	report := IssueReport{
		IssueText:          issueText,
		Alpha:              alpha,
		Beta:               beta,
		ExchangeRounds:     exchanges,
		VerdictExplanation: verdict.Explanation,
	}
	if verdict.Agree {
		report.Status = commentConfirmed
	} else {
		report.Status = commentUnresolved
	}
	return report, nil
}

func (r *Runner) runCodex(label string, issueText string, peerTranscript string, round int) (Transcript, error) {
	prompt := buildCodexPrompt(label, r.opts.Task, issueText, peerTranscript, round)
	data, err := r.executeAgent("codex", prompt)
	if err != nil {
		return Transcript{}, err
	}
	return Transcript{
		Agent:    label,
		Round:    round,
		BranchID: stringField(data, "branch_id"),
		Text:     strings.TrimSpace(stringField(data, "response")),
	}, nil
}

func (r *Runner) executeAgent(agent, prompt string) (map[string]any, error) {
	args := map[string]any{
		"agent":            agent,
		"prompt":           prompt,
		"project_name":     r.opts.ProjectName,
		"parent_branch_id": r.opts.ParentBranchID,
	}
	return r.callTool("execute_agent", args)
}

func (r *Runner) callTool(name string, args map[string]any) (map[string]any, error) {
	payload, _ := json.Marshal(args)
	tc := t.ToolCall{Type: "function"}
	tc.Function.Name = name
	tc.Function.Arguments = string(payload)

	start := time.Now()
	itemArgs := sanitizeArgsForEvents(name, args)
	itemID := r.events.ToolStarted("tool_call", name, itemArgs)
	defer func() {
		if itemID != "" {
			r.events.ToolCompleted(itemID, "error", time.Since(start), "", "")
		}
	}()

	resp := r.handler.Handle(tc)
	if resp == nil {
		return nil, errors.New("tool handler returned nil response")
	}
	status, _ := resp["status"].(string)
	if status != "success" {
		errMsg := extractError(resp)
		return nil, fmt.Errorf("%s failed: %s", name, errMsg)
	}
	data, _ := resp["data"].(map[string]any)
	if itemID != "" {
		branchID := t.ExtractBranchID(data)
		summary := stringField(data, "response")
		r.events.ToolCompleted(itemID, "success", time.Since(start), branchID, summary)
		itemID = ""
	}
	if data == nil {
		return nil, fmt.Errorf("%s returned no data", name)
	}
	return data, nil
}

func (r *Runner) checkConsensus(issueText string, alpha Transcript, beta Transcript) (consensusVerdict, error) {
	prompt := buildConsensusPrompt(issueText, alpha, beta)
	resp, err := r.brain.Complete([]b.ChatMessage{
		{Role: "system", Content: "Return JSON verdicts comparing codex transcripts. Always reply as JSON."},
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil {
		return consensusVerdict{}, err
	}
	return parseConsensus(resp.Choices[0].Message.Content)
}

type eventHelper struct {
	streamer *streaming.JSONStreamer
	nextID   int
}

func newEventHelper(streamer *streaming.JSONStreamer) *eventHelper {
	if streamer == nil || !streamer.Enabled() {
		return nil
	}
	return &eventHelper{streamer: streamer}
}

func (e *eventHelper) ToolStarted(kind, name string, args map[string]any) string {
	if e == nil {
		return ""
	}
	e.nextID++
	itemID := fmt.Sprintf("item_%d", e.nextID)
	e.streamer.EmitItemStarted(itemID, kind, name, args)
	return itemID
}

func (e *eventHelper) ToolCompleted(itemID, status string, duration time.Duration, branchID, summary string) {
	if e == nil || itemID == "" {
		return
	}
	e.streamer.EmitItemCompleted(itemID, status, duration, branchID, summary)
}

func sanitizeArgsForEvents(name string, args map[string]any) map[string]any {
	out := map[string]any{}
	if args == nil {
		return out
	}
	switch name {
	case "execute_agent":
		if agent, _ := args["agent"].(string); agent != "" {
			out["agent"] = agent
		}
		if project, _ := args["project_name"].(string); project != "" {
			out["project_name"] = project
		}
		if parent, _ := args["parent_branch_id"].(string); parent != "" {
			out["parent_branch_id"] = parent
		}
		if prompt, _ := args["prompt"].(string); prompt != "" {
			out["prompt_preview"] = streaming.PromptPreview(prompt)
		}
	default:
		for k, v := range args {
			switch val := v.(type) {
			case string:
				out[k] = streaming.PromptPreview(val)
			case float64, bool:
				out[k] = val
			}
		}
	}
	return out
}

func stringField(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if v, ok := data[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

func extractError(resp map[string]any) string {
	if resp == nil {
		return "unknown error"
	}
	if errObj, ok := resp["error"]; ok && errObj != nil {
		switch val := errObj.(type) {
		case string:
			return strings.TrimSpace(val)
		case map[string]any:
			if msg, _ := val["message"].(string); msg != "" {
				return strings.TrimSpace(msg)
			}
			if instr, _ := val["instruction"].(string); instr != "" {
				return strings.TrimSpace(instr)
			}
		}
	}
	return "unknown error"
}

func summarizeIssueCounts(reports []IssueReport) (confirmed, unresolved int) {
	for _, r := range reports {
		switch r.Status {
		case commentConfirmed:
			confirmed++
		case commentUnresolved:
			unresolved++
		}
	}
	return
}
