package httputil

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const maxRoutingModelBytes = 1024

var errInvalidRoutingBody = errors.New("request must contain one unambiguous model string")

func ExtractGatewayPreviousResponseID(body []byte) (string, error) {
	if !gjson.ValidBytes(body) {
		return "", errInvalidRoutingBody
	}
	object := gjson.ParseBytes(body)
	if !object.IsObject() || routingFieldCount(object, "previous_response_id") > 1 {
		return "", errInvalidRoutingBody
	}
	value := object.Get("previous_response_id")
	if !value.Exists() || value.Type == gjson.Null {
		return "", nil
	}
	if value.Type != gjson.String || len(value.String()) > 1024 {
		return "", errInvalidRoutingBody
	}
	return strings.TrimSpace(value.String()), nil
}

func routingModelJSON(body []byte, session bool) (string, error) {
	if !gjson.ValidBytes(body) {
		return "", errInvalidRoutingBody
	}
	object := gjson.ParseBytes(body)
	if session {
		if routingFieldCount(object, "session") != 1 {
			return "", errInvalidRoutingBody
		}
		object = object.Get("session")
	}
	if routingFieldCount(object, "model") != 1 {
		return "", errInvalidRoutingBody
	}
	value := object.Get("model")
	if value.Type != gjson.String {
		return "", errInvalidRoutingBody
	}
	return validateRoutingModel(value.String())
}

func routingFieldCount(object gjson.Result, field string) int {
	if !object.IsObject() {
		return 0
	}
	count := 0
	object.ForEach(func(key, _ gjson.Result) bool {
		if key.String() == field {
			count++
		}
		return true
	})
	return count
}

func validateRoutingModel(model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" || len(model) > maxRoutingModelBytes {
		return "", errInvalidRoutingBody
	}
	return model, nil
}

func routingMultipartBoundary(contentType string) (string, error) {
	if strings.TrimSpace(contentType) == "" {
		return "", nil
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", errInvalidRoutingBody
	}
	if !strings.EqualFold(mediaType, "multipart/form-data") {
		return "", nil
	}
	boundary := params["boundary"]
	if err := multipart.NewWriter(io.Discard).SetBoundary(boundary); err != nil {
		return "", errInvalidRoutingBody
	}
	return boundary, nil
}

// ExtractGatewayRoutingModel only reads routing metadata. Unknown sibling
// fields and uploaded bytes remain untouched for the endpoint's own parser.
func ExtractGatewayRoutingModel(contentType string, body []byte, session bool) (string, error) {
	boundary, err := routingMultipartBoundary(contentType)
	if err != nil {
		return "", err
	}
	if boundary == "" {
		return routingModelJSON(body, session)
	}
	field := "model"
	if session {
		field = "session"
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	model, count := "", 0
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", errInvalidRoutingBody
		}
		if part.FormName() != field || part.FileName() != "" {
			if _, err := io.Copy(io.Discard, part); err != nil {
				return "", errInvalidRoutingBody
			}
			continue
		}
		count++
		if count > 1 {
			return "", errInvalidRoutingBody
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return "", errInvalidRoutingBody
		}
		if session {
			model, err = routingModelJSON(data, false)
		} else {
			model, err = validateRoutingModel(string(data))
		}
		if err != nil {
			return "", err
		}
	}
	if count != 1 {
		return "", errInvalidRoutingBody
	}
	return model, nil
}

func RewriteGatewayRoutingModel(contentType string, body []byte, session bool, model string) ([]byte, error) {
	model, err := validateRoutingModel(model)
	if err != nil {
		return nil, err
	}
	if _, err := ExtractGatewayRoutingModel(contentType, body, session); err != nil {
		return nil, err
	}
	boundary, err := routingMultipartBoundary(contentType)
	if err != nil {
		return nil, err
	}
	if boundary == "" {
		path := "model"
		if session {
			path = "session.model"
		}
		return sjson.SetBytes(body, path, model)
	}
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, err
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		target, err := writer.CreatePart(part.Header)
		if err != nil {
			return nil, err
		}
		if part.FileName() == "" && ((!session && part.FormName() == "model") || (session && part.FormName() == "session")) {
			data := []byte(model)
			if session {
				data, err = io.ReadAll(part)
				if err == nil {
					data, err = sjson.SetBytes(data, "model", model)
				}
			}
			if err != nil {
				return nil, err
			}
			if _, err := target.Write(data); err != nil {
				return nil, err
			}
		} else if _, err := io.Copy(target, part); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
