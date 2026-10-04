// Package modelcatalog 提供随版本打包的 models.dev 模型目录快照，供模型广场展示官方信息。
// 快照由 tools/generate_model_catalog.py 按产品根目录 model-plaza-catalog.json 生成，运行时不访问外网。
package modelcatalog

import (
	_ "embed"
	"encoding/json"
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
}

type Provider struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Models []Model `json:"models"`
}

type Catalog struct {
	Source    string     `json:"source"`
	FetchedAt string     `json:"fetched_at"`
	Providers []Provider `json:"providers"`
}

// Entry 是查到的模型及其所属厂商。
type Entry struct {
	ProviderID   string
	ProviderName string
	Model        Model
}

var (
	catalog Catalog
	// byID 按小写模型 ID 索引；同一 ID 出现在多个厂商时取参数文件中排在前面的厂商。
	byID = map[string]Entry{}
)

func init() {
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
	for _, provider := range catalog.Providers {
		for _, model := range provider.Models {
			key := strings.ToLower(model.ID)
			if _, exists := byID[key]; exists {
				continue
			}
			byID[key] = Entry{ProviderID: provider.ID, ProviderName: provider.Name, Model: model}
		}
	}
}

// Lookup 按模型 ID 查找，不区分大小写。
func Lookup(modelID string) (Entry, bool) {
	entry, ok := byID[strings.ToLower(strings.TrimSpace(modelID))]
	return entry, ok
}

// FetchedAt 返回快照的抓取时间。
func FetchedAt() string { return catalog.FetchedAt }
