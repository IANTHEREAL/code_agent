package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	b "dev_agent/internal/brain"
	cfg "dev_agent/internal/config"
	o "dev_agent/internal/orchestrator"
	t "dev_agent/internal/tools"
)

func main() {
	task := flag.String("task", "", "User task description")
	parent := flag.String("parent-branch-id", "", "Parent branch UUID (required)")
	project := flag.String("project-name", "", "Optional project name override")
	headless := flag.Bool("headless", false, "Run in headless mode (no chat prints)")
	flag.Parse()

	conf, err := cfg.FromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	if *project != "" {
		conf.ProjectName = *project
	}
	if conf.ProjectName == "" {
		fmt.Fprintln(os.Stderr, "Project name must be provided via PROJECT_NAME or --project-name")
		os.Exit(1)
	}
	if *parent == "" {
		fmt.Fprintln(os.Stderr, "--parent-branch-id is required")
		os.Exit(1)
	}

	tsk := *task
	if tsk == "" {
		fmt.Printf("you> Enter task description: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		tsk = strings.TrimSpace(line)
		if tsk == "" {
			fmt.Fprintln(os.Stderr, "error: task is required")
			os.Exit(1)
		}
	}

	brain := b.NewLLMBrain(conf.AzureAPIKey, conf.AzureEndpoint, conf.AzureDeployment, conf.AzureAPIVersion, 3)
	mcp := t.NewMCPClient(conf.MCPBaseURL)
	handler := t.NewToolHandler(mcp, conf.ProjectName, *parent)

	msgs := o.BuildInitialMessages(tsk, conf.ProjectName, conf.WorkspaceDir, *parent)
	publish := o.PublishOptions{
		GitHubToken:    conf.GitHubToken,
		WorkspaceDir:   conf.WorkspaceDir,
		ParentBranchID: *parent,
		ProjectName:    conf.ProjectName,
		Task:           tsk,
		GitUserName:    conf.GitUserName,
		GitUserEmail:   conf.GitUserEmail,
	}

	var report map[string]any
	if *headless {
		report, err = o.Orchestrate(brain, handler, msgs, publish)
	} else {
		report, err = o.ChatLoop(brain, handler, msgs, 0, publish)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Attach observed branch range and instructions
	br := handler.BranchRange()
	if report == nil {
		report = map[string]any{}
	}
	if start, ok := br["start_branch_id"]; ok {
		report["start_branch_id"] = start
	}
	if latest, ok := br["latest_branch_id"]; ok {
		report["latest_branch_id"] = latest
	}
	if _, ok := report["task"]; !ok {
		report["task"] = tsk
	}
	if instr := o.BuildInstructions(report); instr != "" {
		report["instructions"] = instr
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
}
