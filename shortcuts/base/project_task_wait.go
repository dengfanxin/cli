// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/larksuite/cli/shortcuts/common"
)

var ProjectTaskWait = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-wait",
	Description: "Block until a task reaches a target status (default: done)",
	Risk:        "read",
	Scopes:      []string{"base:app:read", "base:record:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "task-id", Desc: "record ID of the task to wait for", Required: true},
		{Name: "status", Desc: "target status to wait for (default: done)", Default: "done",
			Enum: []string{"done", "blocked", "in_progress"}},
		{Name: "interval", Type: "int", Default: "5", Desc: "polling interval in seconds"},
		{Name: "timeout", Type: "int", Default: "600", Desc: "max wait time in seconds (0 = unlimited)"},
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskWait(ctx, runtime)
	},
}

func executeProjectTaskWait(ctx context.Context, runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	taskID := runtime.Str("task-id")
	targetStatus := runtime.Str("status")
	if targetStatus == "" {
		targetStatus = "done"
	}
	interval := runtime.Int("interval")
	if interval < 1 {
		interval = 5
	}
	timeout := runtime.Int("timeout")

	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	stderr := runtime.IO().ErrOut
	fmt.Fprintf(stderr, "Waiting for task %s to reach status=%s (interval=%ds, timeout=%ds)\n",
		taskID, targetStatus, interval, timeout)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	start := time.Now()

	// Check immediately, then on each tick.
	check := func() (bool, error) {
		status, fields, err := getTaskStatus(runtime, baseToken, tasksTableID, taskID)
		if err != nil {
			fmt.Fprintf(stderr, "poll error: %v\n", err)
			return false, nil // don't exit on transient errors
		}

		if status == targetStatus {
			result := map[string]interface{}{
				"reached":   true,
				"task_id":   taskID,
				"status":    status,
				"title":     fields[fieldTaskTitle],
				"summary":   fields[fieldTaskSummary],
				"waited_ms": time.Since(start).Milliseconds(),
			}
			runtime.Out(result, nil)
			return true, nil
		}

		// If task is blocked or done while waiting for something else, warn.
		if status == "blocked" && targetStatus != "blocked" {
			fmt.Fprintf(stderr, "warning: task %s is blocked\n", taskID)
		}

		return false, nil
	}

	if done, err := check(); err != nil {
		return err
	} else if done {
		return nil
	}

	for {
		select {
		case <-ticker.C:
			if timeout > 0 && time.Since(start) > time.Duration(timeout)*time.Second {
				return fmt.Errorf("timeout: task %s did not reach status=%s within %ds", taskID, targetStatus, timeout)
			}
			if done, err := check(); err != nil {
				return err
			} else if done {
				return nil
			}
		case <-sigCh:
			fmt.Fprintf(stderr, "\nInterrupted. Task %s has not reached status=%s\n", taskID, targetStatus)
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// getTaskStatus reads a specific task record and returns its current status and fields.
func getTaskStatus(runtime *common.RuntimeContext, baseToken, tasksTableID, taskID string) (string, map[string]interface{}, error) {
	data, err := baseV3Call(runtime, "GET",
		baseV3Path("bases", baseToken, "tables", tasksTableID, "records", taskID),
		nil, nil)
	if err != nil {
		return "", nil, err
	}

	// Response may be compact format or standard format.
	// Try to extract fields.
	fields, _ := data["fields"].(map[string]interface{})
	if fields != nil {
		status, _ := fields[fieldTaskStatus].(string)
		return status, fields, nil
	}

	// Compact format: data has "data", "fields", "record_id_list".
	fieldNames := toStringSlice(data["fields"])
	rawRows, _ := data["data"].([]interface{})
	if len(fieldNames) > 0 && len(rawRows) > 0 {
		row, _ := rawRows[0].([]interface{})
		parsedFields := map[string]interface{}{}
		for i, col := range row {
			if i < len(fieldNames) {
				parsedFields[fieldNames[i]] = col
			}
		}
		status := canonicalValue(parsedFields[fieldTaskStatus])
		return status, parsedFields, nil
	}

	return "", nil, fmt.Errorf("could not parse task record %s", taskID)
}
