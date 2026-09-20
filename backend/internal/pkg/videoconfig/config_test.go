//go:build unit

package videoconfig

import (
	"encoding/json"
	"testing"
)

func raw(value string) json.RawMessage { return json.RawMessage(value) }
func integer(value int) *int { return &value }
func fixture() Config {
	return Config{Bindings: []Binding{{
		ID: "channel-a", Name: "Test channel", Enabled: true, AccountID: 7,
		PublicModel: "video-demo", UpstreamModel: "grok-imagine-video", Protocol: XAIVideo,
		Parameters: map[string]Parameter{
			"seconds": {Type: "integer", Required: true, Default: raw("3"), Minimum: integer(1), Maximum: integer(15)},
			"resolution": {Type: "string", Required: true, Default: raw(`"720p"`), Locked: true, Enum: []json.RawMessage{raw(`"480p"`), raw(`"720p"`)}},
		},
		Prices: []Price{{Unit: "per_second", USD: "0.10"}},
	}}}
}

func TestDecimalQuoteAndImmutableConfiguration(t *testing.T) {
	config := fixture()
	compiled, err := Compile(config)
	if err != nil { t.Fatal(err) }
	config.Bindings[0].Prices[0].USD = "99"
	config.Bindings[0].Parameters["seconds"] = Parameter{Type: "integer", Default: raw("10")}
	quote, err := compiled.Resolve("channel-a", nil)
	if err != nil || quote.BaseCostUSD != "0.3" || quote.AccountID != 7 || len(quote.ConfigHash) != 64 { t.Fatalf("quote: %+v, %v", quote, err) }
	quote.Parameters["seconds"] = raw("15")
	fresh, err := compiled.Resolve("channel-a", nil)
	if err != nil || fresh.BaseCostUSD != "0.3" { t.Fatal("returned snapshot mutated compiled configuration") }
	if _, err := compiled.Resolve("channel-a", map[string]json.RawMessage{"resolution": raw(`"480p"`)}); err == nil { t.Fatal("locked value overridden") }
	for _, key := range []string{"authorization", "base_url", "account_id", "price", "model", "headers", "task_id"} {
		if _, err := compiled.Resolve("channel-a", map[string]json.RawMessage{key: raw(`"injected"`)}); err == nil { t.Fatalf("control field accepted: %s", key) }
	}
}

func TestInvalidConfigurationAndUnpricedVariant(t *testing.T) {
	cases := []func(*Config){
		func(c *Config) { c.Bindings[0].Protocol = "not-installed" },
		func(c *Config) { c.Bindings[0].Prices[0].USD = "-1" },
		func(c *Config) { c.Bindings[0].Prices[0].USD = "NaN" },
		func(c *Config) { c.Bindings[0].Prices[0].USD = "1e9999" },
		func(c *Config) { c.Bindings[0].Prices[0].USD = "0.12345678901" },
		func(c *Config) { c.Bindings = append(c.Bindings, c.Bindings[0]) },
		func(c *Config) { c.Bindings[0].Parameters["authorization"] = Parameter{Type: "string"} },
		func(c *Config) { c.Bindings[0].Parameters["seconds"] = Parameter{Type: "integer", Locked: true} },
		func(c *Config) { c.Bindings[0].Prices = append(c.Bindings[0].Prices, Price{Unit: "per_job", USD: "1", Resolution: "720p"}) },
	}
	for index, mutate := range cases {
		c := fixture(); mutate(&c)
		if _, err := Compile(c); err == nil { t.Fatalf("invalid case %d accepted", index) }
	}
	c := fixture()
	c.Bindings[0].Prices[0].Resolution = "480p"
	compiled, err := Compile(c)
	if err != nil { t.Fatal(err) }
	if _, err := compiled.Resolve("channel-a", nil); err == nil { t.Fatal("unpriced 720p variant quoted as free") }
	if _, err := Decode([]byte(`{"bindings":[],"headers":{"Authorization":"secret"}}`)); err == nil { t.Fatal("unknown configuration control accepted") }
}

func TestNativeProtocolRequestsAndResults(t *testing.T) {
	compiled, err := Compile(fixture())
	if err != nil { t.Fatal(err) }
	snapshot, err := compiled.Resolve("channel-a", nil)
	if err != nil { t.Fatal(err) }
	request, err := BuildCreate(snapshot, "A sailboat on a quiet lake")
	if err != nil || request.Path != "/v1/videos/generations" || request.Method != "POST" { t.Fatalf("xAI create: %+v, %v", request, err) }
	var body map[string]any
	if json.Unmarshal(request.Body, &body) != nil || body["duration"] != float64(3) || body["seconds"] != nil { t.Fatalf("incorrect xAI mapping: %s", request.Body) }
	created, err := ParseCreate(XAIVideo, []byte(`{"request_id":"task_1"}`))
	if err != nil || created.State != TaskPending { t.Fatalf("create parse: %+v, %v", created, err) }
	result, err := ParsePoll(XAIVideo, created.ID, []byte(`{"status":"done","video":{"url":"https://media.example/video.mp4","duration":3,"respect_moderation":true},"model":"grok-imagine-video"}`))
	if err != nil || result.State != TaskSucceeded || result.DurationSeconds != 3 || result.ContentPath != "" { t.Fatalf("xAI result: %+v, %v", result, err) }

	snapshot.Protocol = OpenAIJSON
	snapshot.UpstreamModel = "compatible-video-model"
	snapshot.Parameters = map[string]json.RawMessage{"seconds": raw("4"), "size": raw(`"1280x720"`)}
	request, err = BuildCreate(snapshot, "A sailboat on a quiet lake")
	if err != nil || request.Path != "/v1/videos" { t.Fatalf("OpenAI-compatible create: %+v, %v", request, err) }
	body = nil
	if json.Unmarshal(request.Body, &body) != nil || body["seconds"] != "4" || body["duration"] != nil { t.Fatalf("incorrect OpenAI-compatible mapping: %s", request.Body) }
	result, err = ParsePoll(OpenAIJSON, "task_2", []byte(`{"id":"task_2","status":"completed","seconds":"4"}`))
	if err != nil || result.ContentPath != "/v1/videos/task_2/content" || result.ResultURL != "" { t.Fatalf("OpenAI-compatible result: %+v, %v", result, err) }
}

func TestUnknownAndMalformedResultsAreNotTerminalSuccess(t *testing.T) {
	for _, payload := range []string{
		`{"status":"unknown"}`, `{"status":"done","video":null}`,
		`{"status":"done","status":"pending"}`, `{"status":"pending"} {}`,
		`{"status":"done","video":{"url":"http://host/result","duration":3}}`,
	} {
		if _, err := ParsePoll(XAIVideo, "task_1", []byte(payload)); err == nil { t.Fatalf("invalid response accepted: %s", payload) }
	}
	if _, err := ParsePoll(OpenAIJSON, "task_1", []byte(`{"id":"other","status":"completed"}`)); err == nil { t.Fatal("task identity mismatch accepted") }
	if _, err := BuildPoll(XAIVideo, "../another/task"); err == nil { t.Fatal("path injection accepted") }
	poll, err := BuildPoll(XAIVideo, "task_1")
	if err != nil || poll.Method != "GET" || len(poll.Body) != 0 { t.Fatal("poll created a paid request") }
}
