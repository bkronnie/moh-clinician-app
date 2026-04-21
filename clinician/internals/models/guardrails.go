package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

var canonicalRights = []string{"National Admin", "Facility Admin", "Staff"}

// CheckCanonicalRightsDrift verifies that the rights table matches the expected
// application role baseline and returns non-fatal drift details.
func CheckCanonicalRightsDrift(ctx context.Context, db DB) (bool, []string, error) {
	const sqlstr = `
		SELECT COALESCE(rights, '')
		FROM clinician_app.rights
		ORDER BY rights ASC
	`

	rows, err := db.QueryContext(ctx, sqlstr)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	actual := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, nil, err
		}
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		actual = append(actual, trimmed)
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}

	details := evaluateCanonicalRightsDrift(actual)
	return len(details) > 0, details, nil
}

func evaluateCanonicalRightsDrift(actual []string) []string {
	canonicalSet := map[string]struct{}{}
	for _, name := range canonicalRights {
		canonicalSet[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	missing := []string{}
	unexpected := []string{}
	actualSet := map[string]struct{}{}
	actualLookup := map[string]string{}

	for _, name := range actual {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		actualSet[normalized] = struct{}{}
		if _, exists := actualLookup[normalized]; !exists {
			actualLookup[normalized] = strings.TrimSpace(name)
		}
	}

	for _, name := range canonicalRights {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if _, found := actualSet[normalized]; !found {
			missing = append(missing, name)
		}
	}

	for normalized, raw := range actualLookup {
		if _, ok := canonicalSet[normalized]; !ok {
			unexpected = append(unexpected, raw)
		}
	}

	sort.Strings(missing)
	sort.Strings(unexpected)

	details := []string{}
	if len(missing) > 0 {
		details = append(details, fmt.Sprintf("missing canonical roles: %s", strings.Join(missing, ", ")))
	}
	if len(unexpected) > 0 {
		details = append(details, fmt.Sprintf("unexpected roles present: %s", strings.Join(unexpected, ", ")))
	}
	if len(actualLookup) != len(canonicalRights) {
		details = append(details, fmt.Sprintf("role count mismatch: expected %d, found %d", len(canonicalRights), len(actualLookup)))
	}

	return details
}
