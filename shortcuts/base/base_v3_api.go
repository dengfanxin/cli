// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/shortcuts/common"
)

type baseV3Error string

func (e baseV3Error) Error() string { return string(e) }

func baseV3Raw(runtime *common.RuntimeContext, method, path string, params map[string]interface{}, data interface{}, headers ...http.Header) (map[string]interface{}, error) {
	queryParams := make(larkcore.QueryParams)
	for k, v := range params {
		queryParams.Set(k, fmt.Sprintf("%v", v))
	}
	req := &larkcore.ApiReq{
		HttpMethod:  strings.ToUpper(method),
		ApiPath:     path,
		Body:        data,
		QueryParams: queryParams,
	}
	headersToSend := make(http.Header)
	headersToSend.Set("X-App-Id", runtime.Config.AppID)
	for _, extra := range headers {
		for key, values := range extra {
			for _, value := range values {
				headersToSend.Add(key, value)
			}
		}
	}
	resp, err := runtime.DoAPI(req, larkcore.WithHeaders(headersToSend))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		body := strings.TrimSpace(string(resp.RawBody))
		if body == "" {
			return nil, baseV3Error(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}
		return nil, baseV3Error(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, body))
	}
	var result map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(resp.RawBody))
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		return nil, baseV3Error(fmt.Sprintf("response parse error: %v", err))
	}
	return result, nil
}

func baseV3Delete(runtime *common.RuntimeContext, path string) (map[string]interface{}, error) {
	result, err := baseV3Raw(runtime, http.MethodDelete, path, nil, nil)
	return handleBaseAPIResult(result, err, "delete API call failed")
}

func baseV3Call(runtime *common.RuntimeContext, method, path string, params map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	result, err := baseV3Raw(runtime, method, path, params, data)
	return handleBaseAPIResult(result, err, "API call failed")
}

func baseV3CallAny(runtime *common.RuntimeContext, method, path string, params map[string]interface{}, data interface{}) (interface{}, error) {
	result, err := baseV3Raw(runtime, method, path, params, data)
	return handleBaseAPIResultAny(result, err, "API call failed")
}
