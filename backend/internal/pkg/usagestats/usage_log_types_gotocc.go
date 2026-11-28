package usagestats

// UserDashboardFocus 是用户仪表盘重点卡片用的只读统计，嵌入 UserDashboardStats 后与其他字段平级输出。
//
// 近 24 小时只算本人：成功是记了用量的请求；失败取运维错误记录里状态码 ≥ 400 的请求（不含
// count_tokens），与使用记录页「错误请求」同口径；其中平台原因失败只算上游或本网关造成的，用户
// 自己的参数、鉴权问题和余额、额度、并发等业务限制不算。成功率 = 成功 ÷（成功 + 平台原因失败）。
type UserDashboardFocus struct {
	Last24hSuccess        int64    `json:"last24h_success"`
	Last24hFailed         int64    `json:"last24h_failed"`
	Last24hPlatformFailed int64    `json:"last24h_platform_failed"`
	Last24hTTFTMedianMs   *float64 `json:"last24h_ttft_median_ms"`
	// 近 7 天实际扣费合计，余额卡用它算日均消费和预计可用天数。
	Last7dActualCost float64 `json:"last7d_actual_cost"`
}
