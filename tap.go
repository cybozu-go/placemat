package placemat

import (
	"fmt"
	"sync"
)

type tapNameGenerator struct {
	mu sync.Mutex
	id int
}

func (g *tapNameGenerator) New () string {
	g.mu.Lock()
	defer g.mu.Unlock()

	name := fmt.Sprintf("pmtap%d", g.id)
	g.id++
	return name
}

func (g *tapNameGenerator) GeneratedNames () []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	rv := make([]string, g.id)
	for i := 0; i < g.id; i++ {
		rv[i] = fmt.Sprintf("pmtap%d", i)
	}
	return rv
}
