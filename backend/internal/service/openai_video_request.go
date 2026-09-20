package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/tidwall/sjson"
)

// OpenAIVideoRequestParameters reads scalar fields for routing, audit and billing.
// The original body remains the upstream payload, including every file part.
func OpenAIVideoRequestParameters(body []byte, contentType string) ([]byte, error) {
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid video JSON request")
		}
		return body, nil
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	fields := make(map[string]any)
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.FileName() != "" {
			_ = part.Close()
			continue
		}
		value, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		name := part.FormName()
		var field any = string(value)
		if name == "metadata" && json.Valid(value) {
			field = json.RawMessage(value)
		}
		if previous, exists := fields[name]; exists {
			if values, ok := previous.([]any); ok {
				fields[name] = append(values, field)
			} else {
				fields[name] = []any{previous, field}
			}
		} else {
			fields[name] = field
		}
	}
	return json.Marshal(fields)
}

// ReplaceOpenAIVideoRequestModel applies an explicit model mapping without
// changing the remaining fields or the bytes and headers of reference files.
func ReplaceOpenAIVideoRequestModel(body []byte, contentType, model string) ([]byte, string, error) {
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		updated, err := sjson.SetBytes(body, "model", model)
		return updated, contentType, err
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", err
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	if err := writer.SetBoundary(params["boundary"]); err != nil {
		return nil, "", err
	}
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		target, err := writer.CreatePart(part.Header)
		if err != nil {
			return nil, "", err
		}
		if part.FormName() == "model" && part.FileName() == "" {
			_, err = io.WriteString(target, model)
		} else {
			_, err = io.Copy(target, part)
		}
		if err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return output.Bytes(), writer.FormDataContentType(), nil
}
