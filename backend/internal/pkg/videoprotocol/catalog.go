package videoprotocol

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	yp "github.com/LuckyKuang/sub2api-plus/internal/pkg/yingceprotocol"
	"strings"
)

//go:embed catalog_gen.json
var catalogJSON []byte

type Models map[string]Config

func (m Models) Clone() Models {
	if m == nil {
		return nil
	}
	out := Models{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

type CatalogEntry struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Vendor   string          `json:"vendor"`
	Version  string          `json:"version"`
	Package  string          `json:"package"`
	Manifest json.RawMessage `json:"manifest"`
}
type CatalogData struct {
	SourceRepository string         `json:"source_repository"`
	SourceCommit     string         `json:"source_commit"`
	Protocols        []CatalogEntry `json:"protocols"`
}

var catalog CatalogData

func init() {
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
}
func Catalog() CatalogData { return catalog }
func Lookup(id string) (CatalogEntry, error) {
	for _, p := range catalog.Protocols {
		if p.ID == id {
			return p, nil
		}
	}
	return CatalogEntry{}, fmt.Errorf("unknown Yingce video protocol %s", id)
}
func (c *Config) FreezeCatalog() error {
	if c.Protocol != "yingce" {
		return nil
	}
	entry, err := Lookup(c.ProviderID)
	if err != nil {
		return err
	}
	var m yp.Manifest
	if err = json.Unmarshal(entry.Manifest, &m); err != nil {
		return err
	}
	m.Metadata.Documentation = ""
	if c.ProviderDefinition != nil {
		definition := *c.ProviderDefinition
		definition.ID = c.ProviderID
		m.Contributes.Providers = []yp.ManifestProvider{definition}
	}
	m.Contributes = yp.ManifestContributions{Providers: m.Contributes.Providers}
	c.ProviderManifest, err = json.Marshal(m)
	return err
}
func (c Config) Adapter() (yp.Adapter, error) {
	if len(c.ProviderManifest) == 0 {
		return nil, fmt.Errorf("video protocol snapshot is missing")
	}
	return yp.LoadManifest(c.ProviderManifest)
}
func (c Config) normalizeYingceResponse(body []byte, taskID string, create bool) ([]byte, error) {
	a, err := c.Adapter()
	if err != nil {
		return nil, err
	}
	var id, message string
	var status yp.Status
	var result *yp.Result
	if create {
		r, e := a.ParseCreate(context.Background(), body)
		if e != nil {
			return nil, e
		}
		id, message, status, result = r.TaskID, r.Message, r.Status, r.Result
	} else {
		p := yp.PollContext{TaskID: taskID, Model: c.UpstreamModel}
		if c.PollRequest != nil {
			p.Request = *c.PollRequest
		}
		r, e := a.ParsePoll(context.Background(), p, body)
		if e != nil {
			return nil, e
		}
		id, message, status, result = r.TaskID, r.Message, r.Status, r.Result
	}
	if id == "" {
		id = taskID
	}
	if id == "" {
		return nil, fmt.Errorf("video provider response has no task id")
	}
	state := string(status)
	if status == yp.StatusSucceeded {
		state = "completed"
	}
	out := map[string]any{"id": id, "object": "video", "status": state, "provider_status": string(status)}
	if result != nil && len(result.Videos) > 0 {
		out["url"] = result.Videos[0].URL
	}
	if message != "" && status == yp.StatusFailed {
		out["error"] = map[string]any{"message": message}
	}
	return json.Marshal(out)
}
func CanonicalParameter(name string) string {
	switch name {
	case "duration":
		return "seconds"
	case "aspectRatio":
		return "aspect_ratio"
	case "generateAudio":
		return "generate_audio"
	case "providerOptions":
		return "provider_options"
	}
	return strings.ReplaceAll(name, "providerOptions.", "provider_options.")
}

type CatalogView struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Vendor        string                   `json:"vendor"`
	Version       string                   `json:"version"`
	Definition    yp.ManifestProvider      `json:"definition"`
	Configuration yp.ManifestConfiguration `json:"configuration"`
	Workflows     []yp.ManifestWorkflow    `json:"workflows"`
	Documentation string                   `json:"documentation"`
	Template      Config                   `json:"template"`
}

func CatalogViews() ([]CatalogView, error) {
	out := []CatalogView{}
	for _, entry := range catalog.Protocols {
		var m yp.Manifest
		if err := json.Unmarshal(entry.Manifest, &m); err != nil {
			return nil, err
		}
		p := m.Contributes.Providers[0]
		template := Config{Protocol: "yingce", ProviderID: entry.ID, Enabled: true, Defaults: map[string]any{}, Parameters: []Parameter{}, RequestFields: map[string]string{}, Headers: map[string]string{}, Statuses: map[string]string{}, ProviderOptions: map[string]map[string]any{}}
		for _, param := range p.Parameters {
			if param.Name == "model" || param.Name == "prompt" || param.Name == "providerOptions" {
				continue
			}
			kind := param.Type
			if kind == "media[]" {
				kind = "array"
			}
			template.Parameters = append(template.Parameters, Parameter{Name: CanonicalParameter(param.Name), Label: param.Name, Type: kind, Required: param.Required, Values: append([]string{}, param.Values...)})
		}
		doc := strings.Split(m.Metadata.Documentation, "<!-- YINGCE_MANIFEST_CONTRACT_START -->")[0]
		workflows := []yp.ManifestWorkflow{}
		for _, w := range m.Contributes.Workflows {
			if w.ProviderID == entry.ID {
				workflows = append(workflows, w)
			}
		}
		out = append(out, CatalogView{ID: entry.ID, Name: entry.Name, Vendor: entry.Vendor, Version: entry.Version, Definition: p, Configuration: m.Configuration, Workflows: workflows, Documentation: doc, Template: template})
	}
	return out, nil
}
