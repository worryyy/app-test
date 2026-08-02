package ciimpact

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ParseNameStatus(reader io.Reader) ([]Change, error) {
	scanner := bufio.NewScanner(reader)
	changes := make([]Change, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid git name-status line %q", line)
		}
		change := Change{Status: fields[0]}
		if strings.HasPrefix(change.Status, "R") || strings.HasPrefix(change.Status, "C") {
			if len(fields) != 3 {
				return nil, fmt.Errorf("invalid rename/copy line %q", line)
			}
			change.OldPath, change.Path = cleanPath(fields[1]), cleanPath(fields[2])
		} else {
			change.Path = cleanPath(fields[1])
		}
		changes = append(changes, change)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read git name-status: %w", err)
	}
	return changes, nil
}
