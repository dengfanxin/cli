// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

var BaseRecordDownloadAttachment = common.Shortcut{
	Service:     "base",
	Command:     "+record-download-attachment",
	Description: "Download a Base attachment file by file_token to local",
	Risk:        "read",
	Scopes:      []string{"drive:file:download"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		baseTokenFlag(true),
		{Name: "file-token", Desc: "attachment file_token (from a Base attachment field)", Required: true},
		{Name: "output", Desc: "local save path (default: file-token)"},
		{Name: "overwrite", Type: "bool", Desc: "overwrite existing output file"},
	},
	DryRun: dryRunRecordDownloadAttachment,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeRecordDownloadAttachment(ctx, runtime)
	},
}

// buildBitableExtra constructs the "extra" query param required by the Base
// attachment download API. Format: {"bitablePerm":{"tableId":"","rev":99999}}
// The tableId can be empty when the base-token alone is sufficient.
func buildBitableExtra() string {
	extra := map[string]interface{}{
		"bitablePerm": map[string]interface{}{
			"tableId": "",
			"rev":     99999,
		},
	}
	b, _ := json.Marshal(extra)
	return string(b)
}

func dryRunRecordDownloadAttachment(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	fileToken := runtime.Str("file-token")
	outputPath := runtime.Str("output")
	if outputPath == "" {
		outputPath = fileToken
	}
	return common.NewDryRunAPI().
		GET("/open-apis/drive/v1/medias/:file_token/download").
		Desc("Download Base attachment with bitablePerm extra param").
		Set("file_token", fileToken).
		Set("output", outputPath).
		Params(map[string]interface{}{
			"extra": buildBitableExtra(),
		})
}

func executeRecordDownloadAttachment(ctx context.Context, runtime *common.RuntimeContext) error {
	fileToken := runtime.Str("file-token")
	outputPath := runtime.Str("output")
	overwrite := runtime.Bool("overwrite")

	if err := validate.ResourceName(fileToken, "--file-token"); err != nil {
		return output.ErrValidation("%s", err)
	}

	if outputPath == "" {
		outputPath = fileToken
	}

	// Early path validation + overwrite check.
	if _, resolveErr := runtime.ResolveSavePath(outputPath); resolveErr != nil {
		return output.ErrValidation("unsafe output path: %s", resolveErr)
	}
	if _, statErr := runtime.FileIO().Stat(outputPath); statErr == nil && !overwrite {
		return output.ErrValidation("output file already exists: %s (use --overwrite to replace)", outputPath)
	}

	fmt.Fprintf(runtime.IO().ErrOut, "Downloading attachment: %s\n", common.MaskToken(fileToken))

	queryParams := make(larkcore.QueryParams)
	queryParams.Set("extra", buildBitableExtra())

	resp, err := runtime.DoAPIStream(ctx, &larkcore.ApiReq{
		HttpMethod:  http.MethodGet,
		ApiPath:     fmt.Sprintf("/open-apis/drive/v1/medias/%s/download", validate.EncodePathSegment(fileToken)),
		QueryParams: queryParams,
	})
	if err != nil {
		return output.ErrNetwork("download failed: %s", err)
	}
	defer resp.Body.Close()

	result, err := runtime.FileIO().Save(outputPath, fileio.SaveOptions{
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
	}, resp.Body)
	if err != nil {
		return common.WrapSaveErrorByCategory(err, "io")
	}

	savedPath, _ := runtime.ResolveSavePath(outputPath)
	if savedPath == "" {
		savedPath = outputPath
	}
	runtime.Out(map[string]interface{}{
		"saved_path":   savedPath,
		"size_bytes":   result.Size(),
		"content_type": resp.Header.Get("Content-Type"),
	}, nil)
	return nil
}
