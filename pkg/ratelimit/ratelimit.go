package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

type Limit struct {
	runs []time.Time
}

var (
	limits    = make(map[int64]Limit)
	timeRange = 1 * time.Hour
	maxRuns   = 5
	mutex     = &sync.Mutex{}
)

func Request(userID int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	l, ok := limits[userID]
	if !ok {
		l = Limit{}
	}
	x := time.Now().Add(-timeRange)
	var newRuns []time.Time
	for _, r := range l.runs {
		if r.After(x) {
			newRuns = append(newRuns, r)
		}
	}
	if len(newRuns) >= maxRuns {
		return fmt.Errorf("limit of %d runs in %.0fh exceeded", maxRuns, timeRange.Hours())
	}
	l.runs = append(newRuns, time.Now())
	limits[userID] = l
	return nil
}
