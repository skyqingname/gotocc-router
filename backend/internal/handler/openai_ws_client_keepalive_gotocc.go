package handler

import (
	"context"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// startOpenAIWSClientKeepalive 按 gateway.stream_keepalive_interval 向下游 WebSocket 发送 ping，
// 与 HTTP/SSE 下游 keepalive 使用同一参数。连接级读循环持续处理 pong 和断连，
// 不依赖当前轮次是否正在排队或等待上游。
func startOpenAIWSClientKeepalive(ctx context.Context, conn service.OpenAIWSIngressConn, interval time.Duration) context.CancelFunc {
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
