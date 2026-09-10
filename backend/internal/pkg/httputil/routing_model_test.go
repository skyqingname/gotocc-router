package httputil

import (
	"bytes"
	"io"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayRoutingModelJSON(t *testing.T) {
	body := []byte(`{"model":"gpt-test","input":[{"type":"future_item","value":0}],"n":9007199254740993}`)
	original := bytes.Clone(body)
	model, err := ExtractGatewayRoutingModel("application/json", body, false)
	require.NoError(t, err)
	require.Equal(t, "gpt-test", model)
	rewritten, err := RewriteGatewayRoutingModel("application/json", body, false, "gpt-upstream")
	require.NoError(t, err)
	require.Contains(t, string(rewritten), `"model":"gpt-upstream"`)
	require.Contains(t, string(rewritten), `9007199254740993`)
	require.Contains(t, string(rewritten), `{"type":"future_item","value":0}`)
	require.Equal(t, original, body)
}

func TestGatewayRoutingModelRejectsAmbiguousModel(t *testing.T) {
	for _, body := range []string{`{"model":"a","model":"b"}`, `{"model":5}`, `{"model":"a"`, `{"model":""}`} {
		_, err := ExtractGatewayRoutingModel("application/json", []byte(body), false)
		require.Error(t, err)
	}
}

func TestGatewayPreviousResponseIDRejectsAmbiguousOwnership(t *testing.T) {
	for _, payload := range []string{`{"previous_response_id":"a","previous_response_id":"b"}`, `{"previous_response_id":12}`} {
		_, err := ExtractGatewayPreviousResponseID([]byte(payload))
		require.Error(t, err)
	}
	id, err := ExtractGatewayPreviousResponseID([]byte(`{"previous_response_id":"resp_a","future":{}}`))
	require.NoError(t, err)
	require.Equal(t, "resp_a", id)
}

func TestGatewayRoutingModelLiveSession(t *testing.T) {
	body := []byte(`{"model":"ignored","session":{"model":"gpt-live","type":"realtime"}}`)
	model, err := ExtractGatewayRoutingModel("application/json", body, true)
	require.NoError(t, err)
	require.Equal(t, "gpt-live", model)
	rewritten, err := RewriteGatewayRoutingModel("application/json", body, true, "gpt-upstream")
	require.NoError(t, err)
	require.Contains(t, string(rewritten), `"model":"ignored"`)
	require.Contains(t, string(rewritten), `"session":{"model":"gpt-upstream"`)
}

func TestGatewayRoutingModelMultipartPreservesFile(t *testing.T) {
	var input bytes.Buffer
	w := multipart.NewWriter(&input)
	require.NoError(t, w.WriteField("model", "gpt-image-test"))
	file, err := w.CreateFormFile("image", "image.png")
	require.NoError(t, err)
	content := []byte{0x89, 'P', 'N', 'G', 0, 1, 2, 255}
	_, err = file.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	original := bytes.Clone(input.Bytes())
	model, err := ExtractGatewayRoutingModel(w.FormDataContentType(), input.Bytes(), false)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-test", model)
	rewritten, err := RewriteGatewayRoutingModel(w.FormDataContentType(), input.Bytes(), false, "gpt-image-upstream")
	require.NoError(t, err)
	r := multipart.NewReader(bytes.NewReader(rewritten), w.Boundary())
	modelPart, err := r.NextPart()
	require.NoError(t, err)
	modelBytes, err := io.ReadAll(modelPart)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-upstream", string(modelBytes))
	imagePart, err := r.NextPart()
	require.NoError(t, err)
	require.Equal(t, "image.png", imagePart.FileName())
	imageBytes, err := io.ReadAll(imagePart)
	require.NoError(t, err)
	require.Equal(t, content, imageBytes)
	require.Equal(t, original, input.Bytes())
}
