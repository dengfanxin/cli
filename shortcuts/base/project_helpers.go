// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"time"

	"github.com/larksuite/cli/shortcuts/common"
)

// Well-known table names for project shortcuts.
const (
	projectInfoTable      = "_project_info"
	projectKVTable        = "_kv"
	projectMembersTable   = "_members"
	projectTasksTable     = "_tasks"
	projectResourcesTable = "_resources"
)

// wellKnownProjectTables is the set of tables managed by project shortcuts.
var wellKnownProjectTables = map[string]bool{
	projectInfoTable:      true,
	projectKVTable:        true,
	projectMembersTable:   true,
	projectTasksTable:     true,
	projectResourcesTable: true,
}

// Field name constants for _project_info.
const (
	fieldInfoName        = "name"
	fieldInfoDescription = "description"
	fieldInfoStatus      = "status"
)

// Field name constants for _kv.
const (
	fieldKVKey       = "key"
	fieldKVValue     = "value"
	fieldKVSource    = "source"
	fieldKVUpdatedAt = "updated_at"
)

// Field name constants for _members.
const (
	fieldMemberName     = "name"
	fieldMemberRole     = "role"
	fieldMemberType     = "type"
	fieldMemberUserID   = "user_id"
	fieldMemberJoinedAt = "joined_at"
)

// Field name constants for _tasks.
const (
	fieldTaskTitle           = "title"
	fieldTaskDescription     = "description"
	fieldTaskStatus          = "status"
	fieldTaskPriority        = "priority"
	fieldTaskAssignee        = "assignee"
	fieldTaskSubtasks        = "subtasks"
	fieldTaskCreatedAt       = "created_at"
	fieldTaskUpdatedAt       = "updated_at"
	fieldTaskSubtaskProgress = "subtask_progress"
	fieldTaskSummary         = "summary"
)

// Field name constants for _resources.
const (
	fieldResourceName        = "name"
	fieldResourceType        = "type"
	fieldResourceURL         = "url"
	fieldResourceDescription = "description"
)

// projectFlag returns the standard --base-token flag used by project shortcuts.
func projectFlag() common.Flag {
	return baseTokenFlag(true)
}

// projectBaseToken reads the base token from the runtime context.
func projectBaseToken(runtime *common.RuntimeContext) string {
	return runtime.Str("base-token")
}

// listAllProjectTables lists every table in the base, paginating as needed.
func listAllProjectTables(runtime *common.RuntimeContext, baseToken string) ([]map[string]interface{}, error) {
	const pageLimit = 100
	offset := 0
	var all []map[string]interface{}
	for {
		batch, total, err := listAllTables(runtime, baseToken, offset, pageLimit)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) == 0 || len(batch) < pageLimit || (total > 0 && len(all) >= total) {
			break
		}
		offset += len(batch)
	}
	return all, nil
}

// findProjectTableID finds a table by name within the given base.
// Returns the table ID or an error if not found.
func findProjectTableID(runtime *common.RuntimeContext, baseToken, tableName string) (string, error) {
	tables, err := listAllProjectTables(runtime, baseToken)
	if err != nil {
		return "", err
	}
	for _, t := range tables {
		if tableNameFromMap(t) == tableName {
			return tableID(t), nil
		}
	}
	return "", fmt.Errorf("table %q not found in base %s", tableName, baseToken)
}

// ensureProjectTable checks if a well-known table exists; creates it if not.
// When creating a new table, it also creates one additional text field to
// complement the auto-created default field.
// Returns the table ID.
func ensureProjectTable(runtime *common.RuntimeContext, baseToken, tableName string) (string, error) {
	tables, err := listAllProjectTables(runtime, baseToken)
	if err != nil {
		return "", err
	}
	for _, t := range tables {
		if tableNameFromMap(t) == tableName {
			return tableID(t), nil
		}
	}

	// Table does not exist -- create it.
	created, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables"), nil, map[string]interface{}{"name": tableName})
	if err != nil {
		return "", fmt.Errorf("create table %q: %w", tableName, err)
	}
	newTableID := tableID(created)
	if newTableID == "" {
		return "", fmt.Errorf("create table %q: no table_id in response", tableName)
	}

	// Create fields for the table. Base API only auto-creates an "ID" field;
	// we need to add all the fields required by the project shortcuts.
	// A short delay between field creations avoids API rate limits.
	fields := projectTableFields(tableName)
	for i, field := range fields {
		_, err = baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", newTableID, "fields"), nil, field)
		if err != nil {
			return "", fmt.Errorf("create field %v in table %q: %w", field["name"], tableName, err)
		}
		if i < len(fields)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return newTableID, nil
}

// projectTableFields returns the field definitions to create for a well-known table.
func projectTableFields(tableName string) []map[string]interface{} {
	tf := func(name string) map[string]interface{} {
		return map[string]interface{}{"type": "text", "name": name}
	}
	switch tableName {
	case projectKVTable:
		return []map[string]interface{}{tf(fieldKVKey), tf(fieldKVValue), tf(fieldKVSource), tf(fieldKVUpdatedAt)}
	case projectMembersTable:
		return []map[string]interface{}{tf(fieldMemberName), tf(fieldMemberRole), tf(fieldMemberType), tf(fieldMemberUserID), tf(fieldMemberJoinedAt)}
	case projectTasksTable:
		return []map[string]interface{}{tf(fieldTaskTitle), tf(fieldTaskDescription), tf(fieldTaskStatus), tf(fieldTaskPriority), tf(fieldTaskAssignee), tf(fieldTaskSubtasks), tf(fieldTaskSummary), tf(fieldTaskCreatedAt), tf(fieldTaskUpdatedAt)}
	case projectResourcesTable:
		return []map[string]interface{}{tf(fieldResourceName), tf(fieldResourceType), tf(fieldResourceURL), tf(fieldResourceDescription)}
	default:
		return nil
	}
}

// searchRecordsByField searches records in a table where the given field equals the given value.
// Lists all records and filters client-side (Base v3 does not expose a search endpoint).
func searchRecordsByField(runtime *common.RuntimeContext, baseToken, tableIDValue, field, fieldValue string) ([]map[string]interface{}, error) {
	all, err := listAllRecords(runtime, baseToken, tableIDValue)
	if err != nil {
		return nil, err
	}
	var matched []map[string]interface{}
	for _, r := range all {
		if recordFieldValue(r, field) == fieldValue {
			matched = append(matched, r)
		}
	}
	return matched, nil
}

// listProjectRecords lists all records in a table and returns both the records
// and the total count from the API response. The total may exceed the number
// of parsed records (e.g. when responses use a compact format).
func listProjectRecords(runtime *common.RuntimeContext, baseToken, tableIDValue string) ([]map[string]interface{}, int, error) {
	const pageLimit = 200
	offset := 0
	var all []map[string]interface{}
	apiTotal := 0
	for {
		data, err := baseV3Call(runtime, "GET", baseV3Path("bases", baseToken, "tables", tableIDValue, "records"), map[string]interface{}{"offset": offset, "limit": pageLimit}, nil)
		if err != nil {
			return nil, 0, err
		}
		batch := parseRecordsFromData(data)
		all = append(all, batch...)
		t := toInt(data["total"])
		if t > apiTotal {
			apiTotal = t
		}
		if len(batch) == 0 || len(batch) < pageLimit || (apiTotal > 0 && len(all) >= apiTotal) {
			break
		}
		offset += len(batch)
	}
	if apiTotal < len(all) {
		apiTotal = len(all)
	}
	return all, apiTotal, nil
}

// listAllRecords is a convenience wrapper around listProjectRecords
// that discards the total count.
func listAllRecords(runtime *common.RuntimeContext, baseToken, tableIDValue string) ([]map[string]interface{}, error) {
	records, _, err := listProjectRecords(runtime, baseToken, tableIDValue)
	return records, err
}

// parseRecordsFromData extracts records from an API response data map.
// It supports two formats:
//  1. Standard: data["items"] is []interface{} of record objects
//  2. Compact: data["records"] has "schema" (field names), "record_ids", and "rows"
func parseRecordsFromData(data map[string]interface{}) []map[string]interface{} {
	// Try standard "items" format first.
	if rawItems, ok := data["items"].([]interface{}); ok && len(rawItems) > 0 {
		items := make([]map[string]interface{}, 0, len(rawItems))
		for _, item := range rawItems {
			if m, ok := item.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
		return items
	}

	// Try compact format: {data: [[...]], fields: ["key","value",...], record_id_list: [...]}
	schema := toStringSlice(data["fields"])
	rawIDs, _ := data["record_id_list"].([]interface{})
	rawRows, _ := data["data"].([]interface{})
	if len(schema) == 0 || len(rawRows) == 0 {
		return nil
	}
	items := make([]map[string]interface{}, 0, len(rawRows))
	for i, rawRow := range rawRows {
		row, ok := rawRow.([]interface{})
		if !ok {
			continue
		}
		fields := map[string]interface{}{}
		for j, col := range row {
			if j < len(schema) {
				fields[schema[j]] = col
			}
		}
		record := map[string]interface{}{"fields": fields}
		if i < len(rawIDs) {
			if id, ok := rawIDs[i].(string); ok {
				record["record_id"] = id
			}
		}
		items = append(items, record)
	}
	return items
}

// recordFieldValue extracts a text field value from a record.
func recordFieldValue(record map[string]interface{}, field string) string {
	fields, _ := record["fields"].(map[string]interface{})
	if fields == nil {
		return ""
	}
	v, _ := fields[field]
	return canonicalValue(v)
}

// recordIDFromMap extracts the record_id from a record map.
func recordIDFromMap(record map[string]interface{}) string {
	if v, _ := record["record_id"].(string); v != "" {
		return v
	}
	v, _ := record["id"].(string)
	return v
}

// nowTimestamp returns the current time as an ISO 8601 string.
func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// isWellKnownTable checks if a table name is a well-known project table.
func isWellKnownTable(name string) bool {
	return wellKnownProjectTables[name]
}

// listExistingFieldNames returns a set of field names that already exist in the table.
func listExistingFieldNames(runtime *common.RuntimeContext, baseToken, tableIDValue string) (map[string]bool, error) {
	data, err := baseV3Call(runtime, "GET", baseV3Path("bases", baseToken, "tables", tableIDValue, "fields"), nil, nil)
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool)
	// data may be the fields array directly (when "data" unwrapping yields []interface{})
	// or a map with "fields" key containing the array.
	var fieldList []interface{}
	if fl, ok := data["fields"].([]interface{}); ok {
		fieldList = fl
	}
	for _, f := range fieldList {
		switch v := f.(type) {
		case map[string]interface{}:
			if n, ok := v["name"].(string); ok {
				names[n] = true
			}
		case string:
			names[v] = true
		}
	}
	return names, nil
}
