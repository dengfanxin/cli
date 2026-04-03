// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/larksuite/cli/shortcuts/common"
)

// knownTaskFields lists the standard task field names used by ensureProjectTable.
var knownTaskFields = map[string]bool{
	fieldTaskTitle:           true,
	fieldTaskDescription:     true,
	fieldTaskStatus:          true,
	fieldTaskPriority:        true,
	fieldTaskAssignee:        true,
	fieldTaskSubtasks:        true,
	fieldTaskCreatedAt:       true,
	fieldTaskUpdatedAt:       true,
	fieldTaskSubtaskProgress: true,
	fieldTaskSummary:         true,
}

// Priority order for task sorting (lower index = higher priority).
var taskPriorityOrder = map[string]int{
	"high":   0,
	"medium": 1,
	"low":    2,
}

// ── ProjectTaskAdd ──────────────────────────────────────────────────────

var ProjectTaskAdd = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-add",
	Description: "Add a task to the project",
	Risk:        "write",
	Scopes:      []string{"base:table:create", "base:field:create", "base:field:read", "base:field:update", "base:record:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "title", Desc: "task title", Required: true},
		{Name: "description", Desc: "task description"},
		{Name: "priority", Desc: "task priority (0=highest, or high/medium/low)", Default: "medium"},
		{Name: "assignee", Desc: "task assignee"},
		{Name: "extra", Desc: `extra fields JSON object, e.g. '{"type":"脚本","skill":"Agent-Script"}'`},
	},
	DryRun: dryRunProjectTaskAdd,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskAdd(runtime)
	},
}

func dryRunProjectTaskAdd(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find or create " + projectTasksTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("Create task record").
		Set("base_token", bt).
		Body(map[string]interface{}{
			"fields": map[string]interface{}{
				fieldTaskTitle:    runtime.Str("title"),
				fieldTaskPriority: runtime.Str("priority"),
				fieldTaskStatus:   "pending",
			},
		})
}

func executeProjectTaskAdd(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	title := strings.TrimSpace(runtime.Str("title"))
	description := strings.TrimSpace(runtime.Str("description"))
	priority := strings.TrimSpace(runtime.Str("priority"))
	if priority == "" {
		priority = "medium"
	}
	assignee := strings.TrimSpace(runtime.Str("assignee"))
	extraRaw := strings.TrimSpace(runtime.Str("extra"))

	tasksTableID, err := ensureProjectTable(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	// If --extra is provided, parse it and create any new fields that are not
	// already well-known task fields. Base v3 API requires fields to exist
	// before writing records.
	var extraFields map[string]interface{}
	if extraRaw != "" {
		if err := json.Unmarshal([]byte(extraRaw), &extraFields); err != nil {
			return fmt.Errorf("parse --extra JSON: %w", err)
		}
		created := 0
		for key := range extraFields {
			if knownTaskFields[key] {
				continue
			}
			_, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", tasksTableID, "fields"), nil, map[string]interface{}{
				"type": "text",
				"name": key,
			})
			if err != nil {
				return fmt.Errorf("create field %q: %w", key, err)
			}
			created++
			if created > 0 {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	now := nowTimestamp()
	fields := map[string]interface{}{
		fieldTaskTitle:     title,
		fieldTaskStatus:    "pending",
		fieldTaskPriority:  priority,
		fieldTaskCreatedAt: now,
		fieldTaskUpdatedAt: now,
	}
	if description != "" {
		fields[fieldTaskDescription] = description
	}
	if assignee != "" {
		fields[fieldTaskAssignee] = assignee
	}
	// Merge extra fields into the record.
	for k, v := range extraFields {
		fields[k] = v
	}

	data, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", tasksTableID, "records"), nil, fields)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{"record": data, "created": true, "title": title}, nil)
	return nil
}

// ── ProjectTaskNext ─────────────────────────────────────────────────────

var ProjectTaskNext = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-next",
	Description: "Claim the highest-priority pending task and mark it in_progress",
	Risk:        "write",
	Scopes:      []string{"base:app:read", "base:record:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "filter", Type: "string_array", Desc: "filter by field value, e.g. type=脚本"},
	},
	DryRun: dryRunProjectTaskNext,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskNext(runtime)
	},
}

func dryRunProjectTaskNext(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectTasksTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/search").
		Desc("Search for pending tasks").
		Set("base_token", bt).
		PATCH("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/:record_id").
		Desc("Update task status to in_progress").
		Set("base_token", bt)
}

func executeProjectTaskNext(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	filters := runtime.StrArray("filter")

	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	// Search for pending tasks.
	records, err := searchRecordsByField(runtime, baseToken, tasksTableID, fieldTaskStatus, "pending")
	if err != nil {
		return err
	}

	// Apply --filter before priority sorting.
	records = applyRecordFilters(records, filters)

	if len(records) == 0 {
		runtime.Out(map[string]interface{}{"message": "no pending tasks"}, nil)
		return nil
	}

	// Find the highest priority task. Lower numeric value = higher priority.
	// Also supports string priorities via the taskPriorityOrder map.
	best := records[0]
	bestPriority := taskPriorityRank(best)
	for _, r := range records[1:] {
		p := taskPriorityRank(r)
		if p < bestPriority {
			best = r
			bestPriority = p
		}
	}

	// Update status to in_progress.
	recordID := recordIDFromMap(best)
	updateBody := map[string]interface{}{
		"fields": map[string]interface{}{
			fieldTaskStatus:    "in_progress",
			fieldTaskUpdatedAt: nowTimestamp(),
		},
	}
	_, err = baseV3Call(runtime, "PATCH", baseV3Path("bases", baseToken, "tables", tasksTableID, "records", recordID), nil, updateBody)
	if err != nil {
		return err
	}

	fields, _ := best["fields"].(map[string]interface{})
	result := map[string]interface{}{
		"claimed":     true,
		"record_id":   recordID,
		"title":       fields[fieldTaskTitle],
		"description": fields[fieldTaskDescription],
		"priority":    fields[fieldTaskPriority],
		"assignee":    fields[fieldTaskAssignee],
		"status":      "in_progress",
	}
	if subtasks, ok := fields[fieldTaskSubtasks]; ok && subtasks != nil {
		result["subtasks"] = subtasks
	}
	runtime.Out(result, nil)
	return nil
}

// taskPriorityRank returns a numeric rank for a task's priority.
// Lower values mean higher priority. Supports both numeric (0, 1, 2)
// and string ("high", "medium", "low") priorities.
func taskPriorityRank(record map[string]interface{}) int {
	fields, _ := record["fields"].(map[string]interface{})
	if fields == nil {
		return 999
	}
	raw := fields[fieldTaskPriority]
	// Try numeric value first.
	n := toInt(raw)
	if n > 0 || canonicalValue(raw) == "0" {
		return n
	}
	// Try string mapping.
	if rank, ok := taskPriorityOrder[canonicalValue(raw)]; ok {
		return rank
	}
	return 999
}

// ── ProjectTaskList ─────────────────────────────────────────────────────

var ProjectTaskList = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-list",
	Description: "List project tasks, optionally filtered by status",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "status", Desc: "filter by status", Enum: []string{"pending", "in_progress", "done", "blocked"}},
		{Name: "filter", Type: "string_array", Desc: "filter by field value, e.g. type=脚本"},
	},
	DryRun: dryRunProjectTaskList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskList(runtime)
	},
}

func dryRunProjectTaskList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	d := common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectTasksTable).
		Set("base_token", bt)
	if status := strings.TrimSpace(runtime.Str("status")); status != "" {
		d.POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/search").
			Desc("Search tasks by status").
			Set("base_token", bt)
	} else {
		d.GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
			Desc("List all task records").
			Set("base_token", bt)
	}
	return d
}

func executeProjectTaskList(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	status := strings.TrimSpace(runtime.Str("status"))
	filters := runtime.StrArray("filter")

	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	var records []map[string]interface{}
	if status != "" {
		records, err = searchRecordsByField(runtime, baseToken, tasksTableID, fieldTaskStatus, status)
	} else {
		records, err = listAllRecords(runtime, baseToken, tasksTableID)
	}
	if err != nil {
		return err
	}

	// Apply --filter after status filter, before output.
	records = applyRecordFilters(records, filters)

	items := make([]interface{}, 0, len(records))
	for _, r := range records {
		fields, _ := r["fields"].(map[string]interface{})
		items = append(items, map[string]interface{}{
			"record_id":   recordIDFromMap(r),
			"title":       fields[fieldTaskTitle],
			"description": fields[fieldTaskDescription],
			"status":      fields[fieldTaskStatus],
			"priority":    fields[fieldTaskPriority],
			"assignee":    fields[fieldTaskAssignee],
			"created_at":  fields[fieldTaskCreatedAt],
			"updated_at":  fields[fieldTaskUpdatedAt],
		})
	}

	runtime.Out(map[string]interface{}{"items": items, "count": len(items)}, nil)
	return nil
}

// ── ProjectTaskUpdate ───────────────────────────────────────────────────

var ProjectTaskUpdate = common.Shortcut{
	Service:     "base",
	Command:     "+project-task-update",
	Description: "Update a project task's status, summary, or subtask progress",
	Risk:        "write",
	Scopes:      []string{"base:record:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "task-id", Desc: "record ID of the task", Required: true},
		{Name: "status", Desc: "new status", Enum: []string{"pending", "in_progress", "done", "blocked"}},
		{Name: "summary", Desc: "task summary or result"},
		{Name: "subtask-progress", Desc: "subtask progress description"},
	},
	DryRun: dryRunProjectTaskUpdate,
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return validateProjectTaskUpdate(runtime)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectTaskUpdate(runtime)
	},
}

func dryRunProjectTaskUpdate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		PATCH("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/:record_id").
		Desc("Update task record").
		Set("base_token", bt).
		Set("record_id", runtime.Str("task-id"))
}

func validateProjectTaskUpdate(runtime *common.RuntimeContext) error {
	status := strings.TrimSpace(runtime.Str("status"))
	summary := strings.TrimSpace(runtime.Str("summary"))
	subtaskProgress := strings.TrimSpace(runtime.Str("subtask-progress"))
	if status == "" && summary == "" && subtaskProgress == "" {
		return common.FlagErrorf("at least one of --status, --summary, or --subtask-progress is required")
	}
	return nil
}

func executeProjectTaskUpdate(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	taskID := strings.TrimSpace(runtime.Str("task-id"))
	status := strings.TrimSpace(runtime.Str("status"))
	summary := strings.TrimSpace(runtime.Str("summary"))
	subtaskProgress := strings.TrimSpace(runtime.Str("subtask-progress"))

	tasksTableID, err := findProjectTableID(runtime, baseToken, projectTasksTable)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{
		fieldTaskUpdatedAt: nowTimestamp(),
	}
	if status != "" {
		fields[fieldTaskStatus] = status
	}
	if summary != "" {
		fields[fieldTaskSummary] = summary
	}
	if subtaskProgress != "" {
		fields[fieldTaskSubtaskProgress] = subtaskProgress
	}

	data, err := baseV3Call(runtime, "PATCH", baseV3Path("bases", baseToken, "tables", tasksTableID, "records", taskID), nil, fields)
	if err != nil {
		return fmt.Errorf("update task %s: %w", taskID, err)
	}

	runtime.Out(map[string]interface{}{"record": data, "updated": true, "task_id": taskID}, nil)
	return nil
}

// applyRecordFilters filters records by "key=value" expressions.
// Each filter string is split on the first "=" to get the field name and value.
// Only records matching ALL filters are retained.
func applyRecordFilters(records []map[string]interface{}, filters []string) []map[string]interface{} {
	if len(filters) == 0 {
		return records
	}
	var result []map[string]interface{}
	for _, r := range records {
		match := true
		for _, f := range filters {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key, val := parts[0], parts[1]
			if recordFieldValue(r, key) != val {
				match = false
				break
			}
		}
		if match {
			result = append(result, r)
		}
	}
	return result
}
