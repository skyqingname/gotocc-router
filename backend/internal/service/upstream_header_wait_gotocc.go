package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

// UpstreamHeaderWaitObserver 观察"请求已发往上游、尚未收到上游响应头"的等待区间。
// 流式入口借它在等待期间向下游保活，见 docs/STREAM_KEEPALIVE.md。
type UpstreamHeaderWaitObserver interface {
	// BeginUpstreamHeaderWait 在上游请求发出时调用；返回的函数在响应头到达或请求失败时调用。
	BeginUpstreamHeaderWait() (end func())
}

type upstreamHeaderWaitObserverKey struct{}

// WithUpstreamHeaderWaitObserver 把观察者挂到请求 context；由它派生的上游请求都会通知观察者。
func WithUpstreamHeaderWaitObserver(ctx context.Context, observer UpstreamHeaderWaitObserver) context.Context {
	return context.WithValue(ctx, upstreamHeaderWaitObserverKey{}, observer)
}

// BeginUpstreamHeaderWait 由上游 HTTP 出口在发送前调用；context 未挂观察者时返回空操作。
func BeginUpstreamHeaderWait(ctx context.Context) func() {
	observer, ok := ctx.Value(upstreamHeaderWaitObserverKey{}).(UpstreamHeaderWaitObserver)
	if !ok {
		return func() {}
	}
	return observer.BeginUpstreamHeaderWait()
}

// RecordStreamKeepaliveBytes 记录网关自发的非语义心跳字节。
// OpenAICompactKeepaliveAdjustedWrittenSize 与 openAIStreamClientOutputStarted 扣除这部分字节，
// 只写过心跳的响应仍按"正文未开始"参与换号判定。
func RecordStreamKeepaliveBytes(c *gin.Context, written int) {
	recordOpenAIStreamKeepaliveBytes(c, written)
}
