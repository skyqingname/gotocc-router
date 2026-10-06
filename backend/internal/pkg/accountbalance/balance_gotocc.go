package accountbalance

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Candidate contains scheduling facts only, never account credentials.
type Candidate struct {
	ID         int64
	Priority   int
	Capability int
	Available  bool
	QuotaUsed  float64
	QuotaKnown bool
	LastUsedAt time.Time
}

// Rotation shares new-session assignments across groups within the single core.
// Existing sticky sessions return before requesting another rotation turn.
type Rotation struct {
	mu   sync.Mutex
	turn uint64
	last map[int64]uint64
}

type peerGroup struct {
	priority   int
	capability int
	available  bool
}

func (r *Rotation) Order(candidates []Candidate) []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.last == nil {
		r.last = make(map[int64]uint64)
	}

	minimumQuota := make(map[peerGroup]float64)
	groupOf := func(c Candidate) peerGroup {
		return peerGroup{c.Priority, c.Capability, c.Available}
	}
	for _, candidate := range candidates {
		if !candidate.QuotaKnown {
			continue
		}
		group := groupOf(candidate)
		minimum, found := minimumQuota[group]
		if !found || candidate.QuotaUsed < minimum {
			minimumQuota[group] = candidate.QuotaUsed
		}
	}
	quotaBand := func(c Candidate) int {
		// Unknown quota participates in rotation without inventing a percentage.
		if !c.QuotaKnown {
			return 0
		}
		return int(math.Floor((c.QuotaUsed - minimumQuota[groupOf(c)]) / QuotaGapPercent))
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Available != b.Available {
			return a.Available
		}
		if a.Capability != b.Capability {
			return a.Capability > b.Capability
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if aBand, bBand := quotaBand(a), quotaBand(b); aBand != bBand {
			return aBand < bBand
		}
		if r.last[a.ID] != r.last[b.ID] {
			return r.last[a.ID] < r.last[b.ID]
		}
		// Before this core has assigned a turn, start with the less recently used
		// account. This retains useful history across a core restart.
		if !a.LastUsedAt.Equal(b.LastUsedAt) {
			return a.LastUsedAt.Before(b.LastUsedAt)
		}
		return a.ID < b.ID
	})
	order := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		order = append(order, candidate.ID)
	}
	if len(order) > 0 {
		r.turn++
		r.last[order[0]] = r.turn
	}
	return order
}

// Observe seeds accounts already serving sticky sessions when this core starts.
// Repeated messages in the same session do not consume new-session turns.
func (r *Rotation) Observe(accountID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.last == nil {
		r.last = make(map[int64]uint64)
	}
	if _, known := r.last[accountID]; known {
		return
	}
	r.turn++
	r.last[accountID] = r.turn
}

// Selected records a replacement when the first reservation lost its slot or
// failed the existing fresh-account checks. Normal reservations are unchanged.
func (r *Rotation) Selected(firstID, selectedID int64) {
	if firstID == selectedID {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.turn++
	r.last[selectedID] = r.turn
}
