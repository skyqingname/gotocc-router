package service

import (
	"context"
	"fmt"
)

// RequestLimitScopeID reserves negative IDs for teams in the existing request
// capacity caches. It must never replace an actor, payer or persisted user ID.
func RequestLimitScopeID(userID int64, teamID *int64) int64 {
	if teamID != nil {
		return -*teamID
	}
	return userID
}

func (u *User) RequestLimitScopeID() int64 {
	return RequestLimitScopeID(u.ID, u.TeamWalletID)
}

func (s *ConcurrencyService) SetTeamRepository(repo TeamRepository) {
	s.teamRepo = repo
}

func (s *ConcurrencyService) resolveTeamConcurrency(ctx context.Context, teamID int64) (int, error) {
	if s.teamRepo == nil {
		return 0, fmt.Errorf("team concurrency repository unavailable")
	}
	readCtx, cancel := context.WithTimeout(ctx, regularLeaseOperationTimeout)
	defer cancel()
	team, err := s.teamRepo.GetContextByTeamID(readCtx, teamID)
	if err != nil {
		return 0, fmt.Errorf("current team concurrency unavailable: %w", err)
	}
	return team.Team.Concurrency, nil
}
