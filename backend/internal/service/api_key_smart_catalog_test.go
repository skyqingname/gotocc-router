//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func autoRouteCatalogModelIDs(models []AutoRouteModel) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

type concurrentAutoRouteCatalogRepo struct {
	catalog *AutoRouteCatalog
	calls   atomic.Int64
}

func (r *concurrentAutoRouteCatalogRepo) LoadAutoRouteCatalog(context.Context, []int64) (*AutoRouteCatalog, error) {
	r.calls.Add(1)
	return r.catalog, nil
}

func TestAutoRouteListModelsReturnsOnlyAuthorizedConfiguredUnion(t *testing.T) {
	resolver, key, users, groups, catalog := newAutoRouteFixture()
	groups.groups[1].IsExclusive = true
	users.user.AllowedGroups = []int64{10, 20}

	groups.groups = append(groups.groups, Group{
		ID: 30, Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 3, IsExclusive: true,
	})
	catalog.catalog.Accounts[30] = []Account{{
		ID: 300, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{"model_mapping": map[string]any{"private-model": "private-model"}},
	}}

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"claude-test", "gpt-test"}, autoRouteCatalogModelIDs(models))
	require.Equal(t, 1, catalog.calls, "the catalog must be loaded once for the whole model union")
}

func TestAutoRouteListModelsUsesActualFirstGroupWhenLaterGroupExposesModel(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[1].Platform = PlatformOpenAI
	groups.groups[0].ModelsListConfig = GroupModelsListConfig{Enabled: true, Models: []string{"other-model"}}
	groups.groups[1].ModelsListConfig = GroupModelsListConfig{Enabled: true, Models: []string{"shared-model"}}

	first := catalog.catalog.Accounts[10][0]
	first.Credentials = map[string]any{"model_mapping": map[string]any{"shared-model": "gpt-5.6-sol"}}
	catalog.catalog.Accounts[10] = []Account{first}
	second := first
	second.ID = 200
	catalog.catalog.Accounts[20] = []Account{second}

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "shared-model", models[0].ID)
	require.Equal(t, int64(10), models[0].selectedGroupID,
		"the descriptor must describe the group that will actually receive the request")
}

func TestAutoRouteListModelsDoesNotEnumerateWildcardClaims(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	account := catalog.catalog.Accounts[10][0]
	account.Credentials = map[string]any{"model_mapping": map[string]any{"custom-*": "gpt-5.6-sol"}}
	catalog.catalog.Accounts[10] = []Account{account}
	catalog.catalog.Accounts[20] = nil

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.NoError(t, err)
	require.Empty(t, models, "a wildcard is routing configuration, not a finite public model identifier")
}

func TestAutoRouteListModelsIncludesConfiguredChannelAliases(t *testing.T) {
	resolver, key, _, _, _ := newAutoRouteFixture()
	resolver.channels.cache.Store(populateChannelCache([]Channel{{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{10},
		ModelMapping: map[string]map[string]string{
			PlatformOpenAI: {"public-channel-alias": "gpt-test"},
		},
	}}, map[int64]string{10: PlatformOpenAI}))

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.NoError(t, err)
	require.Contains(t, autoRouteCatalogModelIDs(models), "public-channel-alias")
}

func TestAutoRouteListModelsIncludesFiniteCompositeRouteAliases(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups = groups.groups[:1]
	groups.groups[0].Platform = PlatformComposite
	catalog.catalog.Routes = make(map[int64][]CompositeModelRoute)
	catalog.catalog.Routes[10] = []CompositeModelRoute{{
		ID: 1, GroupID: 10, PublicModel: "public-composite-alias",
		MatchType: CompositeRouteMatchExact, TargetPlatform: PlatformOpenAI,
		UpstreamModel: "gpt-test", Endpoint: CompositeRouteEndpointResponses, Enabled: true,
	}}

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.NoError(t, err)
	require.Contains(t, autoRouteCatalogModelIDs(models), "public-composite-alias")
}

func TestAutoRouteListModelsFailsClosedWhenCatalogCannotBeRead(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	catalog.err = errors.New("catalog read failed")

	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointResponses,
	})

	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
	require.Nil(t, models)
}

func TestAutoRouteListModelsKeepsConcurrentRequestsIndependent(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	sharedCatalog := &concurrentAutoRouteCatalogRepo{catalog: catalog.catalog}
	resolver.catalog = sharedCatalog

	const workers = 16
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{
				Endpoint: CompositeRouteEndpointResponses,
			})
			if err != nil {
				errs <- err
				return
			}
			if got := autoRouteCatalogModelIDs(models); len(got) != 2 || got[0] != "claude-test" || got[1] != "gpt-test" {
				errs <- errors.New("concurrent catalog returned a cross-request model set")
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int64(workers), sharedCatalog.calls.Load())
	require.Nil(t, key.GroupID)
}

func TestAutoRouteBuildCodexModelsManifestUsesAuthorizedCatalogOnly(t *testing.T) {
	resolver, key, users, groups, catalog := newAutoRouteFixture()
	groups.groups[1].IsExclusive = true
	users.user.AllowedGroups = []int64{10, 20}

	groups.groups = append(groups.groups, Group{
		ID: 30, Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 3, IsExclusive: true,
	})
	catalog.catalog.Accounts[30] = []Account{{
		ID: 300, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{"model_mapping": map[string]any{"private-codex-model": "gpt-5.6-sol"}},
	}}

	body, err := resolver.BuildCodexModelsManifest(context.Background(), key)

	require.NoError(t, err)
	var manifest struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	require.NoError(t, json.Unmarshal(body, &manifest))
	got := make([]string, 0, len(manifest.Models))
	for _, model := range manifest.Models {
		got = append(got, model.Slug)
	}
	require.Equal(t, []string{"claude-test", "gpt-test"}, got)
	require.Equal(t, 1, catalog.calls, "the manifest must reuse a single strict catalog snapshot")
}

func TestAutoRouteRoutingCapabilitiesRequireConfiguredAuthorizedEndpoints(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[0].AllowImageGeneration = true
	openAI := catalog.catalog.Accounts[10][0]
	openAI.Credentials = map[string]any{"model_mapping": map[string]any{
		"gpt-test": "gpt-test", "gpt-image-2": "gpt-image-2",
	}}
	catalog.catalog.Accounts[10] = []Account{openAI}

	groups.groups = append(groups.groups, Group{
		ID: 30, Platform: PlatformGemini, Status: StatusActive, SortOrder: 3, AllowBatchImageGeneration: true,
	})
	catalog.catalog.Accounts[30] = []Account{{
		ID: 300, Platform: PlatformGemini, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Credentials: map[string]any{
			"api_key":       "test-api-key",
			"model_mapping": map[string]any{"gemini-2.5-flash-image": "gemini-2.5-flash-image"},
		},
	}}

	capabilities, err := resolver.GetRoutingCapabilities(context.Background(), key)

	require.NoError(t, err)
	require.Equal(t, string(APIKeyRoutingAuto), capabilities.RoutingMode)
	require.Equal(t, []string{"anthropic", "gemini", "openai"}, capabilities.Protocols)
	require.True(t, capabilities.AsyncImageSubmit)
	require.True(t, capabilities.BatchImageSubmit)
	require.Equal(t, 1, catalog.calls, "capabilities must share one strict catalog snapshot")
}
