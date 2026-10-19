// Package modelcatalog 提供随版本打包的 models.dev 模型目录快照，供模型广场展示官方信息。
// 快照由 tools/generate_model_catalog.py 按产品根目录 model-plaza-catalog.json 生成，运行时不访问外网。
package modelcatalog

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
)

//go:embed catalog_gen.json
var catalogJSON []byte

// Cost 是官方单价，单位为美元 / 百万 token；缺失的项为 nil。
type Cost struct {
	Input      *float64 `json:"input"`
	Output     *float64 `json:"output"`
	CacheRead  *float64 `json:"cache_read"`
	CacheWrite *float64 `json:"cache_write"`
	Tiers      []Tier   `json:"tiers"`
}

// Tier 是上下文超过 ContextOver token 后改用的单价。
type Tier struct {
	Input       *float64 `json:"input"`
	Output      *float64 `json:"output"`
	CacheRead   *float64 `json:"cache_read"`
	CacheWrite  *float64 `json:"cache_write"`
	ContextOver int64    `json:"context_over"`
}

type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type Limit struct {
	Context int64 `json:"context"`
	Input   int64 `json:"input"`
	Output  int64 `json:"output"`
}

type Model struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Family           string     `json:"family"`
	Reasoning        bool       `json:"reasoning"`
	ToolCall         bool       `json:"tool_call"`
	StructuredOutput bool       `json:"structured_output"`
	Attachment       bool       `json:"attachment"`
	OpenWeights      bool       `json:"open_weights"`
	Knowledge        string     `json:"knowledge"`
	ReleaseDate      string     `json:"release_date"`
	Modalities       Modalities `json:"modalities"`
	Limit            Limit      `json:"limit"`
	Cost             *Cost      `json:"cost"`
	// Status 为 deprecated 时表示厂商已弃用该 ID；CanonicalModelID 形如 "厂商/规范 ID"。
	Status           string `json:"status"`
	CanonicalModelID string `json:"canonical_model_id"`
}

type Provider struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Models []Model `json:"models"`
}

type Catalog struct {
	Source    string `json:"source"`
	FetchedAt string `json:"fetched_at"`
	// VendorAliases 列出每个厂商还以第一方身份发布哪些规范厂商的模型（同一公司的不同目录名）。
	VendorAliases map[string][]string `json:"vendor_aliases"`
	Providers     []Provider          `json:"providers"`
}

// Entry 是查到的模型及其所属厂商。
type Entry struct {
	ProviderID   string
	ProviderName string
	Model        Model
}

// 官方信息只取模型厂商自己发布的条目：云平台转售的别家模型（规范 ID 属于其他厂商）
// 带的是转售价，不作为官方价。三个索引都只收第一方条目，均按小写键：
// byID 为厂商目录中的模型 ID；byCanonical 为规范 ID；byCanonicalStem 为去掉日期后缀的规范 ID。
// 同一键有多条时，未弃用的优先，其次取参数文件中排在前面的厂商。
var (
	catalog         Catalog
	byID            = map[string]Entry{}
	byCanonical     = map[string]Entry{}
	byCanonicalStem = map[string]Entry{}
	// dateSuffix 匹配 -0731、-260731、-20260731、-2026-07-31 这类版本日期后缀。
	dateSuffix = regexp.MustCompile(`-(\d{4}|\d{6}|\d{8}|\d{4}-\d{2}-\d{2})$`)
)

func init() {
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
	for _, provider := range catalog.Providers {
		for _, model := range provider.Models {
			vendor, canonical, _ := strings.Cut(strings.ToLower(model.CanonicalModelID), "/")
			if vendor != "" && vendor != provider.ID && !contains(catalog.VendorAliases[provider.ID], vendor) {
				continue
			}
			entry := Entry{ProviderID: provider.ID, ProviderName: provider.Name, Model: model}
			index(byID, strings.ToLower(model.ID), entry)
			if canonical != "" {
				index(byCanonical, canonical, entry)
				index(byCanonicalStem, dateSuffix.ReplaceAllString(canonical, ""), entry)
			}
		}
	}
}

func index(entries map[string]Entry, key string, entry Entry) {
	existing, exists := entries[key]
	if !exists || (existing.Model.Status == "deprecated" && entry.Model.Status != "deprecated") {
		entries[key] = entry
	}
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// Lookup 按模型名查找厂商第一方条目，不区分大小写：原名与去掉日期后缀的名字依次
// 匹配目录 ID、规范 ID 和去掉日期后缀的规范 ID。
func Lookup(modelID string) (Entry, bool) {
	name := strings.ToLower(strings.TrimSpace(modelID))
	for _, candidate := range []string{name, dateSuffix.ReplaceAllString(name, "")} {
		for _, entries := range []map[string]Entry{byID, byCanonical, byCanonicalStem} {
			if entry, ok := entries[candidate]; ok {
				return entry, true
			}
		}
	}
	return Entry{}, false
}

// FetchedAt 返回快照的抓取时间。
func FetchedAt() string { return catalog.FetchedAt }
