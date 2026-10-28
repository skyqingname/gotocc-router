package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/tidwall/gjson"
)

// applyUpstreamRequestAdaptations is the single entry for the proven rewrites
// in upstream-request-adaptations.json. Run before identity/signing/compression.
func (s *OpenAIGatewayService) applyUpstreamRequestAdaptations(request *http.Request, owner *Account) error {
	if request.Method != http.MethodPost ||
		!strings.HasSuffix(strings.TrimRight(request.URL.Path, "/"), "/images/edits") ||
		!upstreamImageEditsMultipartHosts[strings.ToLower(request.URL.Hostname())] {
		return nil
	}
	mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if strings.EqualFold(mediaType, "multipart/form-data") {
		return nil
	}
	body, err := io.ReadAll(request.Body)
	_ = request.Body.Close()
	if err != nil {
		return fmt.Errorf("read image edit request: %w", err)
	}
	parsed := &OpenAIImagesRequest{Endpoint: openAIImagesEditsEndpoint}
	if err := parseOpenAIImagesJSONRequest(body, parsed); err != nil {
		return err
	}
	converted, contentType, err := s.buildImageEditMultipart(request.Context(), owner, parsed, body)
	if err != nil {
		return err
	}
	request.Body = io.NopCloser(bytes.NewReader(converted))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(converted)), nil
	}
	request.ContentLength = int64(len(converted))
	request.Header.Del("Content-Length")
	request.Header.Set("Content-Type", contentType)
	return nil
}

func (s *OpenAIGatewayService) buildImageEditMultipart(
	ctx context.Context,
	owner *Account,
	parsed *OpenAIImagesRequest,
	body []byte,
) ([]byte, string, error) {
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	var fieldErr error
	gjson.ParseBytes(body).ForEach(func(key, value gjson.Result) bool {
		if key.String() == "images" || (key.String() == "mask" && parsed.MaskImageURL != "") {
			return true
		}
		text := value.Raw
		if value.Type == gjson.String {
			text = value.String()
		}
		fieldErr = writer.WriteField(key.String(), text)
		return fieldErr == nil
	})
	if fieldErr != nil {
		return nil, "", fmt.Errorf("write image edit fields: %w", fieldErr)
	}

	imageField := "image"
	if len(parsed.InputImageURLs) > 1 {
		imageField = "image[]"
	}
	for index, imageURL := range parsed.InputImageURLs {
		if err := s.writeOpenAIImageEditFile(ctx, owner, writer, imageField, fmt.Sprintf("image-%d", index+1), imageURL); err != nil {
			return nil, "", fmt.Errorf("prepare image edit input %d: %w", index+1, err)
		}
	}
	if parsed.MaskImageURL != "" {
		if err := s.writeOpenAIImageEditFile(ctx, owner, writer, "mask", "mask", parsed.MaskImageURL); err != nil {
			return nil, "", fmt.Errorf("prepare image edit mask: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finish image edit multipart: %w", err)
	}
	return output.Bytes(), writer.FormDataContentType(), nil
}

func (s *OpenAIGatewayService) writeOpenAIImageEditFile(
	ctx context.Context,
	owner *Account,
	writer *multipart.Writer,
	field string,
	filename string,
	imageURL string,
) error {
	// Reuse the existing account-bound image fetcher, including data URLs,
	// public-host download policy, proxy, identity and per-image download limit.
	encoded, err := s.fetchOpenAIImageURLBase64(ctx, owner, imageURL)
	if err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(data)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     field,
		"filename": filename + extensionForContentType(contentType),
	}))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}
