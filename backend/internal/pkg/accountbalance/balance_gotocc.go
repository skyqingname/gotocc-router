package accountbalance

import (
	"math"
	"sort"
	"sync"
)

// Candidate contains scheduling facts only, never account credentials.
type Candidate struct {
	ID         int64
	Priority   int
	Capability int
	Available  bool
	QuotaUsed  float64
	QuotaKnown bool
}

// Rotation shares turns across requests and groups within the single core.
// Reserving a turn before slot acquisition also spreads concurrent arrivals.
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
