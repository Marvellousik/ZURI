package gaps

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// CodeownerRule represents a path pattern mapped to assigned owners/usernames (§10.7).
type CodeownerRule struct {
	Pattern string
	Owners  []string
}

// ParseCodeowners parses a repository CODEOWNERS file into a list of ordered rules.
func ParseCodeowners(content string) []CodeownerRule {
	var rules []CodeownerRule
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pattern := parts[0]
			var owners []string
			for _, owner := range parts[1:] {
				cleanOwner := strings.TrimPrefix(owner, "@")
				if cleanOwner != "" {
					owners = append(owners, cleanOwner)
				}
			}
			if len(owners) > 0 {
				rules = append(rules, CodeownerRule{Pattern: pattern, Owners: owners})
			}
		}
	}

	return rules
}

// ResolveOwnersForFile matches a file path against CODEOWNERS rules (last matching rule wins per spec).
func ResolveOwnersForFile(filePath string, rules []CodeownerRule) []string {
	var matchedOwners []string
	cleanPath := strings.TrimPrefix(filePath, "/")

	for _, rule := range rules {
		p := strings.TrimPrefix(rule.Pattern, "/")
		if p == "*" || strings.HasPrefix(cleanPath, strings.TrimSuffix(p, "*")) || strings.Contains(cleanPath, p) {
			matchedOwners = rule.Owners
		}
	}

	return matchedOwners
}

// RouteGapToOwners routes a knowledge gap to responsible engineers based on affected files and CODEOWNERS rules.
func RouteGapToOwners(ctx context.Context, db *sql.DB, gapID string, codeownersContent string) ([]string, error) {
	rules := ParseCodeowners(codeownersContent)
	if len(rules) == 0 {
		return nil, nil
	}

	// Query touched file paths for memory records associated with this gap
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT f.file_path
		FROM memory_touches_file f
		JOIN knowledge_gap g ON f.memory_id = ANY(g.affected_memory_ids)
		WHERE g.gap_id = $1;
	`, gapID)

	if err != nil {
		return nil, fmt.Errorf("failed querying touched files for gap: %w", err)
	}
	defer rows.Close()

	ownerMap := make(map[string]bool)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil {
			owners := ResolveOwnersForFile(path, rules)
			for _, o := range owners {
				ownerMap[o] = true
			}
		}
	}

	// Fallback to wildcards if no specific file matched
	if len(ownerMap) == 0 {
		for _, o := range ResolveOwnersForFile("*", rules) {
			ownerMap[o] = true
		}
	}

	var assigned []string
	for o := range ownerMap {
		assigned = append(assigned, o)
	}

	if len(assigned) > 0 {
		_, err = db.ExecContext(ctx, `
			UPDATE knowledge_gap
			SET routed_to = $1
			WHERE gap_id = $2;
		`, pq.Array(assigned), gapID)
		if err != nil {
			return assigned, fmt.Errorf("failed updating knowledge_gap routed_to: %w", err)
		}
	}

	return assigned, nil
}
