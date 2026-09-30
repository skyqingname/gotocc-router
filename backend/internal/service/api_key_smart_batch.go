package service

import (
	"context"
	"errors"
	"sort"
	"strings"
)

func (s *AutoGroupResolver) ListBatchModels(ctx context.Context, key *APIKey) ([]BatchImagePublicModel, error) {
	state, err := s.loadCatalogState(ctx, key, AutoRouteRequest{Endpoint: AutoRouteEndpointBatchImages})
	if err != nil {
		return nil, err
	}
	result := make([]BatchImagePublicModel, 0)
	for _, provider := range batchImageProviderSelectionOrder("") {
		models, err := s.listModelsFromState(ctx, state, AutoRouteRequest{Endpoint: AutoRouteEndpointBatchImages, Provider: provider})
		if err != nil {
			return nil, err
		}
		for _, model := range models {
			result = append(result, BatchImagePublicModel{ID: model.ID, Object: "image.batch.model", Provider: provider})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID == result[j].ID {
			return result[i].Provider < result[j].Provider
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// FindIdempotentSubmission is an owner-scoped read. It must precede a new
// automatic route and never re-enqueues, reserves funds, or contacts upstream.
func (s *BatchImagePublicService) FindIdempotentSubmission(ctx context.Context, owner BatchImageOwner, req BatchImageSubmitRequest, idempotencyKey string) (*BatchImagePublicBatch, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, nil
	}
	if s == nil || s.Repo == nil {
		return nil, ErrAutoRouteUnavailable
	}
	job, err := s.Repo.GetBatchImageJobByIdempotencyKey(ctx, owner.UserID, owner.APIKeyID, idempotencyKey)
	if errors.Is(err, ErrBatchImageJobNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrAutoRouteUnavailable
	}
	if strings.TrimSpace(req.TaskName) == "" {
		req.TaskName = job.TaskName
	}
	normalized, err := s.validateSubmitRequest(req)
	if err != nil {
		return nil, err
	}
	if batchImageDerefString(job.RequestHash) != HashBatchImageSubmitRequest(normalized) {
		return nil, ErrBatchImageIdempotencyConflict
	}
	return BatchImageJobToPublic(job), nil
}
