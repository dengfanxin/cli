// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

// ── ProjectKVSet ────────────────────────────────────────────────────────

var ProjectKVSet = common.Shortcut{
	Service:     "base",
	Command:     "+project-kv-set",
	Description: "Set a key-value pair in the project KV store",
	Risk:        "write",
	Scopes:      []string{"base:table:create", "base:field:create", "base:field:read", "base:field:update", "base:record:create", "base:record:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "key", Desc: "key name", Required: true},
		{Name: "value", Desc: "value to store", Required: true},
		{Name: "source", Desc: "source label (e.g. agent name)"},
	},
	DryRun: dryRunProjectKVSet,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectKVSet(runtime)
	},
}

func dryRunProjectKVSet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find or create " + projectKVTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/search").
		Desc("Search for existing key").
		Set("base_token", bt).
		Body(map[string]interface{}{
			"filter": map[string]interface{}{
				"conjunction": "and",
				"conditions": []interface{}{
					map[string]interface{}{"field_name": fieldKVKey, "operator": "is", "value": []interface{}{runtime.Str("key")}},
				},
			},
		}).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("Create or update KV record").
		Set("base_token", bt)
}

func executeProjectKVSet(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	key := strings.TrimSpace(runtime.Str("key"))
	value := runtime.Str("value")
	source := strings.TrimSpace(runtime.Str("source"))

	// Ensure KV table exists.
	kvTableID, err := ensureProjectTable(runtime, baseToken, projectKVTable)
	if err != nil {
		return err
	}

	// Search for existing key.
	existing, err := searchRecordsByField(runtime, baseToken, kvTableID, fieldKVKey, key)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{
		fieldKVKey:       key,
		fieldKVValue:     value,
		fieldKVUpdatedAt: nowTimestamp(),
	}
	if source != "" {
		fields[fieldKVSource] = source
	}
	body := fields

	if len(existing) > 0 {
		// Update existing record.
		recordID := recordIDFromMap(existing[0])
		data, err := baseV3Call(runtime, "PATCH", baseV3Path("bases", baseToken, "tables", kvTableID, "records", recordID), nil, body)
		if err != nil {
			return err
		}
		runtime.Out(map[string]interface{}{"record": data, "updated": true, "key": key}, nil)
		return nil
	}

	// Create new record.
	data, err := baseV3Call(runtime, "POST", baseV3Path("bases", baseToken, "tables", kvTableID, "records"), nil, body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"record": data, "created": true, "key": key}, nil)
	return nil
}

// ── ProjectKVGet ────────────────────────────────────────────────────────

var ProjectKVGet = common.Shortcut{
	Service:     "base",
	Command:     "+project-kv-get",
	Description: "Get a value by key from the project KV store",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "key", Desc: "key name", Required: true},
	},
	DryRun: dryRunProjectKVGet,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectKVGet(runtime)
	},
}

func dryRunProjectKVGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectKVTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/search").
		Desc("Search for key").
		Set("base_token", bt).
		Body(map[string]interface{}{
			"filter": map[string]interface{}{
				"conjunction": "and",
				"conditions": []interface{}{
					map[string]interface{}{"field_name": fieldKVKey, "operator": "is", "value": []interface{}{runtime.Str("key")}},
				},
			},
		})
}

func executeProjectKVGet(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	key := strings.TrimSpace(runtime.Str("key"))

	kvTableID, err := findProjectTableID(runtime, baseToken, projectKVTable)
	if err != nil {
		return err
	}

	records, err := searchRecordsByField(runtime, baseToken, kvTableID, fieldKVKey, key)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("key %q not found", key)
	}

	fields, _ := records[0]["fields"].(map[string]interface{})
	runtime.Out(map[string]interface{}{
		"key":        key,
		"value":      fields[fieldKVValue],
		"source":     fields[fieldKVSource],
		"updated_at": fields[fieldKVUpdatedAt],
	}, nil)
	return nil
}

// ── ProjectKVList ───────────────────────────────────────────────────────

var ProjectKVList = common.Shortcut{
	Service:     "base",
	Command:     "+project-kv-list",
	Description: "List all key-value pairs in the project KV store",
	Risk:        "read",
	Scopes:      []string{"base:app:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "prefix", Desc: "filter keys by prefix"},
	},
	DryRun: dryRunProjectKVList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectKVList(runtime)
	},
}

func dryRunProjectKVList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectKVTable).
		Set("base_token", bt).
		GET("/open-apis/base/v3/bases/:base_token/tables/:table_id/records").
		Desc("List all KV records").
		Set("base_token", bt)
}

func executeProjectKVList(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	prefix := strings.TrimSpace(runtime.Str("prefix"))

	kvTableID, err := findProjectTableID(runtime, baseToken, projectKVTable)
	if err != nil {
		return err
	}

	records, err := listAllRecords(runtime, baseToken, kvTableID)
	if err != nil {
		return err
	}

	items := make([]interface{}, 0, len(records))
	for _, r := range records {
		key := recordFieldValue(r, fieldKVKey)
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		fields, _ := r["fields"].(map[string]interface{})
		items = append(items, map[string]interface{}{
			"key":        key,
			"value":      fields[fieldKVValue],
			"source":     fields[fieldKVSource],
			"updated_at": fields[fieldKVUpdatedAt],
		})
	}

	runtime.Out(map[string]interface{}{"items": items, "count": len(items)}, nil)
	return nil
}

// ── ProjectKVDelete ─────────────────────────────────────────────────────

var ProjectKVDelete = common.Shortcut{
	Service:     "base",
	Command:     "+project-kv-delete",
	Description: "Delete a key-value pair from the project KV store",
	Risk:        "write",
	Scopes:      []string{"base:record:delete"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		projectFlag(),
		{Name: "key", Desc: "key to delete", Required: true},
	},
	DryRun: dryRunProjectKVDelete,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeProjectKVDelete(runtime)
	},
}

func dryRunProjectKVDelete(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	bt := projectBaseToken(runtime)
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/tables").
		Desc("List tables to find " + projectKVTable).
		Set("base_token", bt).
		POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/search").
		Desc("Search for key to delete").
		Set("base_token", bt).
		DELETE("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/:record_id").
		Desc("Delete the KV record").
		Set("base_token", bt)
}

func executeProjectKVDelete(runtime *common.RuntimeContext) error {
	baseToken := projectBaseToken(runtime)
	key := strings.TrimSpace(runtime.Str("key"))

	kvTableID, err := findProjectTableID(runtime, baseToken, projectKVTable)
	if err != nil {
		return err
	}

	records, err := searchRecordsByField(runtime, baseToken, kvTableID, fieldKVKey, key)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("key %q not found", key)
	}

	recordID := recordIDFromMap(records[0])
	_, err = baseV3Call(runtime, "DELETE", baseV3Path("bases", baseToken, "tables", kvTableID, "records", recordID), nil, nil)
	if err != nil {
		return err
	}

	runtime.Out(map[string]interface{}{"deleted": true, "key": key}, nil)
	return nil
}
