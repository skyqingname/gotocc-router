package handler

import (
	"context"
	"time"

	coderws "github.com/coder/websocket"
)

// startOpenAIWSClientKeepalive 按 gateway.stream_keepalive_interval 向下游 WebSocket 发送 ping，
// 与 HTTP/SSE 下游 keepalive 使用同一参数。Cloudflare 对约 125 秒无流量的 WebSocket
// 直接断开，上游长时间推理或回合间隔都会触发。回合进行中核心不读取客户端帧，pong
// 留到下次读取时处理；Ping 在一个间隔内等不到 pong 只返回错误，不关闭连接。
func startOpenAIWSClientKeepalive(ctx context.Context, conn *coderws.Conn, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)
	if interval <= 0 {
		return cancel
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, pingCancel := context.WithTimeout(ctx, interval)
				_ = conn.Ping(pingCtx)
				pingCancel()
			}
		}
	}()
	return cancel
}
