// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/larksuite/cli/shortcuts/common"
)

var ProjectTaskListen = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-listen",
	Description: "Listen for pending tasks and optionally execute a command for each",
	Risk:        "write",
	Scopes:      []string{"base:app:read", "base:record:read", "base:record:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "filter", Type: "string_array", Desc: "filter by field value, e.g. agent=画图Agent"},
		{Name: "interval", Type: "int", Default: "5", Desc: "polling interval in seconds"},
		{Name: "exec", Desc: "command to execute for each task (task prompt passed via stdin)"},
		{Name: "once", Type: "bool", Desc: "exit after processing one task"},
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskListen(ctx, runtime)
	},
}

func executeProjectTaskListen(ctx context.Context, runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	filters := runtime.StrArray("filter")
	interval := runtime.Int("interval")
	if interval < 1 {
		interval = 5
	}
	execCmd := strings.TrimSpace(runtime.Str("exec"))
	once := runtime.Bool("once")

	// Find tasks table up front.
	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	// Handle graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	stderr := runtime.IO().ErrOut
	fmt.Fprintf(stderr, "Listening for tasks (interval=%ds, filter=%v)\n", interval, filters)

	processed := 0
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Poll immediately on start, then on each tick.
	poll := func() (bool, error) {
		task, err := claimNextTask(runtime, baseToken, tasksTableID, filters)
		if err != nil {
			fmt.Fprintf(stderr, "poll error: %v\n", err)
			return false, nil // don't exit on transient errors
		}
		if task == nil {
			return false, nil
		}

		processed++
		taskJSON, _ := json.Marshal(task)

		if execCmd == "" {
			// No --exec: output NDJSON to stdout.
			fmt.Fprintln(runtime.IO().Out, string(taskJSON))
		} else {
			// Build prompt with project context.
			prompt, err := buildTaskPrompt(runtime, baseToken, task)
			if err != nil {
				fmt.Fprintf(stderr, "prompt build error: %v\n", err)
				return false, nil
			}

			// Execute the command with prompt on stdin.
			fmt.Fprintf(stderr, "Executing: %s (task: %s)\n", execCmd, task["title"])
			if err := runExecCommand(execCmd, prompt); err != nil {
				fmt.Fprintf(stderr, "exec error: %v\n", err)
				// Mark task as blocked on exec failure.
				taskID, _ := task["record_id"].(string)
				if taskID != "" {
					_, _ = baseV3Call(runtime, "PATCH",
						baseV3Path("bases", baseToken, "tables", tasksTableID, "records", taskID),
						nil, map[string]interface{}{
							fieldTaskStatus:    "blocked",
							fieldTaskUpdatedAt: nowTimestamp(),
							fieldTaskSummary:   fmt.Sprintf("exec error: %v", err),
						})
				}
				return false, nil
			}

			// Mark task as done after successful exec.
			taskID, _ := task["record_id"].(string)
			if taskID != "" {
				_, _ = baseV3Call(runtime, "PATCH",
					baseV3Path("bases", baseToken, "tables", tasksTableID, "records", taskID),
					nil, map[string]interface{}{
						fieldTaskStatus:    "done",
						fieldTaskUpdatedAt: nowTimestamp(),
					})
			}
		}

		return once, nil // if --once, signal to exit
	}

	// Initial poll.
	if shouldExit, err := poll(); err != nil {
		return err
	} else if shouldExit {
		fmt.Fprintf(stderr, "Processed %d task(s)\n", processed)
		return nil
	}

	for {
		select {
		case <-ticker.C:
			if shouldExit, err := poll(); err != nil {
				return err
			} else if shouldExit {
				fmt.Fprintf(stderr, "Processed %d task(s)\n", processed)
				return nil
			}
		case <-sigCh:
			fmt.Fprintf(stderr, "\nShutting down. Processed %d task(s)\n", processed)
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// claimNextTask finds the highest-priority pending task matching filters,
// claims it (marks in_progress), and returns the task info. Returns nil if none found.
func claimNextTask(runtime *common.RuntimeContext, baseToken, tasksTableID string, filters []string) (map[string]interface{}, error) {
	records, err := searchRecordsByField(runtime, baseToken, tasksTableID, fieldTaskStatus, "pending")
	if err != nil {
		return nil, err
	}

	records = applyRecordFilters(records, filters)
	if len(records) == 0 {
		return nil, nil
	}

	// Find highest priority.
	best := records[0]
	bestPriority := taskPriorityRank(best)
	for _, r := range records[1:] {
		if p := taskPriorityRank(r); p < bestPriority {
			best = r
			bestPriority = p
		}
	}

	// Claim it.
	recordID := recordIDFromMap(best)
	_, err = baseV3Call(runtime, "PATCH",
		baseV3Path("bases", baseToken, "tables", tasksTableID, "records", recordID),
		nil, map[string]interface{}{
			fieldTaskStatus:    "in_progress",
			fieldTaskUpdatedAt: nowTimestamp(),
		})
	if err != nil {
		return nil, err
	}

	fields, _ := best["fields"].(map[string]interface{})
	return map[string]interface{}{
		"record_id":   recordID,
		"project_id":  baseToken,
		"title":       fields[fieldTaskTitle],
		"description": fields[fieldTaskDescription],
		"priority":    fields[fieldTaskPriority],
		"assignee":    fields[fieldTaskAssignee],
		"status":      "in_progress",
		"fields":      fields,
	}, nil
}

// buildTaskPrompt assembles a prompt with project context + task details + relevant KV data.
func buildTaskPrompt(runtime *common.RuntimeContext, baseToken string, task map[string]interface{}) (string, error) {
	var b strings.Builder

	b.WriteString("# Task\n\n")
	b.WriteString(fmt.Sprintf("Title: %v\n", task["title"]))
	if desc := task["description"]; desc != nil && desc != "" {
		b.WriteString(fmt.Sprintf("Description: %v\n", desc))
	}
	if assignee := task["assignee"]; assignee != nil && assignee != "" {
		b.WriteString(fmt.Sprintf("Assignee: %v\n", assignee))
	}
	b.WriteString(fmt.Sprintf("Project ID: %s\n", baseToken))
	b.WriteString(fmt.Sprintf("Task ID: %v\n", task["record_id"]))

	// Append all extra fields from the task.
	if fields, ok := task["fields"].(map[string]interface{}); ok {
		extras := []string{}
		for k, v := range fields {
			if !knownTaskFields[k] && v != nil {
				extras = append(extras, fmt.Sprintf("%s: %v", k, v))
			}
		}
		if len(extras) > 0 {
			b.WriteString("\n## Extra Fields\n\n")
			for _, e := range extras {
				b.WriteString(e + "\n")
			}
		}
	}

	// Append project KV data.
	kvTableID, err := findProjectTableID(runtime, baseToken, projectKVTable)
	if err == nil {
		records, err := listAllRecords(runtime, baseToken, kvTableID)
		if err == nil && len(records) > 0 {
			b.WriteString("\n## Project Knowledge (KV)\n\n")
			for _, r := range records {
				key := recordFieldValue(r, fieldKVKey)
				value := recordFieldValue(r, fieldKVValue)
				if key != "" {
					b.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
				}
			}
		}
	}

	b.WriteString("\n## Instructions\n\n")
	b.WriteString("Complete this task. When done, the system will automatically mark it as completed.\n")
	b.WriteString(fmt.Sprintf("Use `lark-cli base +project-kv-set --base-token %s --key \"output-KEY\" --value \"...\"` to store any outputs.\n", baseToken))

	return b.String(), nil
}

// runExecCommand runs a shell command with the given input on stdin.
func runExecCommand(cmdStr, stdinInput string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdin = strings.NewReader(stdinInput)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
