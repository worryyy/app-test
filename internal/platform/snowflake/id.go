package snowflake

import (
	"fmt"
	"sync"

	sf "github.com/bwmarrin/snowflake"
)

var (
	mu   sync.RWMutex
	node *sf.Node
)

func Init(machineID int64) error {
	n, err := sf.NewNode(machineID)
	if err != nil {
		return fmt.Errorf("init snowflake node: %w", err)
	}

	mu.Lock()
	node = n
	mu.Unlock()
	return nil
}

func Generate() sf.ID {
	mu.RLock()
	n := node
	mu.RUnlock()

	if n == nil {
		_ = Init(1)
		mu.RLock()
		n = node
		mu.RUnlock()
	}

	if n == nil {
		return 0
	}

	return n.Generate()
}
