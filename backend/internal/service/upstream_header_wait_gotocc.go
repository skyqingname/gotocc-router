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

// OpenAIStreamKeepaliveBytes 返回上游响应头到达后、首字等待期间写给客户端的空闲保活字节。
// 这部分字节不构成语义正文，前一次尝试写过它们之后安全换号时，等待期保活仍应继续。
func OpenAIStreamKeepaliveBytes(c *gin.Context) int {
	value, _ := c.Get(openAIStreamKeepaliveBytesKey)
	bytes, _ := value.(int)
	return bytes
}
