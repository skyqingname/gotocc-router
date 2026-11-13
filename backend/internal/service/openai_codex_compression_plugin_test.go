//go:build unit

package service

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	pluginv1 "github.com/LuckyKuang/sub2api-plus/pkg/pluginapi/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type codexCompressionPluginCapture struct {
	pluginv1.UnimplementedTransportPluginServer
	requests chan struct {
		start *pluginv1.ForwardRequestStart
		body  []byte
	}
}

func (s *codexCompressionPluginCapture) Forward(stream pluginv1.TransportPlugin_ForwardServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	var body []byte
	for {
		frame, err := stream.Recv()
		if err != nil {
			return err
		}
		body = append(body, frame.GetBodyChunk()...)
		if frame.GetBodyEnd() {
			break
		}
	}
	s.requests <- struct {
		start *pluginv1.ForwardRequestStart
		body  []byte
	}{first.GetStart(), body}
	if err := stream.Send(&pluginv1.ForwardResponse{Frame: &pluginv1.ForwardResponse_Start{Start: &pluginv1.ForwardResponseStart{StatusCode: 200}}}); err != nil {
		return err
	}
	return stream.Send(&pluginv1.ForwardResponse{Frame: &pluginv1.ForwardResponse_End{End: &pluginv1.ForwardResponseEnd{}}})
}

// The plugin must forward JSON without compression: the Codex upstream rejects
// some locally valid Zstd frames, including the former 512 KiB-window encoding.
func TestCodexOAuthIdentityBodyPluginWire(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() { _ = listener.Close() })
	server := grpc.NewServer()
	capture := &codexCompressionPluginCapture{requests: make(chan struct {
		start *pluginv1.ForwardRequestStart
		body  []byte
	}, 1)}
	pluginv1.RegisterTransportPluginServer(server, capture)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	connection, err := grpc.NewClient("passthrough:///codex-compression-fixture", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close() })
	runtime := &pluginRuntime{api: pluginv1.NewTransportPluginClient(connection)}
	for _, kind := range []string{AccountTypeOAuth, AccountTypeSetupToken, AccountTypeAPIKey} {
		for _, size := range []int{64, 131073, 140132, 2497413} {
			t.Run(kind+"/bytes="+strconv.Itoa(size), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				prefix, suffix := `{"model":"gpt-6.1-sol","stream":true,"input":"`, `"}`
				body := prefix + strings.Repeat("a", size-len(prefix)-len(suffix)) + suffix
				request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://chatgpt.com/backend-api/codex/responses", strings.NewReader(body))
				require.NoError(t, err)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer fixture")
				request.Header.Set("User-Agent", "codex_cli_rs/0.163.0 (Ubuntu 24.04; x86_64) xterm-256color")
				request.Header.Set("Originator", "codex_cli_rs")
				request.Header.Set("Version", "0.163.0")
				require.True(t, runtime.beginRequest())
				response, err := runtime.roundTrip(ctx, request, "", &Account{ID: 91, Platform: PlatformOpenAI, Type: kind})
				require.NoError(t, err)
				_, err = io.Copy(io.Discard, response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				wire := <-capture.requests
				headers := headersFromPlugin(wire.start.Headers)
				require.Empty(t, headers.Get("Content-Encoding"))
				require.Equal(t, "0.163.0", headers.Get("Version"))
				require.Equal(t, "codex_cli_rs", headers.Get("Originator"))
				require.Equal(t, "Bearer fixture", headers.Get("Authorization"))
				require.Equal(t, int64(len(wire.body)), wire.start.ContentLength)
				require.Equal(t, body, string(wire.body))
			})
		}
	}
}
