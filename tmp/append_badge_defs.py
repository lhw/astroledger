#!/usr/bin/env python3
"""Append badge definition CRUD functions to extra_queries.go"""

target = '/Users/lhw/src/astroledger/backend/internal/db/extra_queries.go'

addition = r"""

// ─── Badge Definitions ───────────────────────────────────────────────────────

// BadgeDefinitionRow represents one row from the badge_definitions table.
type BadgeDefinitionRow struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tier        int    `json:"tier"`
	Icon        string `json:"icon"`
	IsHardcoded bool   `json:"is_hardcoded"`
	CreatedAt   string `json:"created_at"`
}

func scanBadgeDefinition(row func(dest ...any) error) (BadgeDefinitionRow, error) {
	var d BadgeDefinitionRow
	var hardcodedInt int64
	err := row(&d.ID, &d.Key, &d.Title, &d.Description, &d.Tier, &d.Icon, &hardcodedInt, &d.CreatedAt)
	if err != nil {
		return d, err
	}
	d.IsHardcoded = hardcodedInt == 1
	return d, nil
}

const badgeDefCols = `id, key, title, description, tier, icon, is_hardcoded, created_at`

// GetAllBadgeDefinitions returns all badge definitions ordered by tier then title.
func (q *Queries) GetAllBadgeDefinitions(ctx context.Context) ([]BadgeDefinitionRow, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT `+badgeDefCols+` FROM badge_definitions ORDER BY tier ASC, title ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BadgeDefinitionRow, 0)
	for rows.Next() {
		d, err := scanBadgeDefinition(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// GetBadgeDefinitionByKey returns a single badge definition by its unique key.
func (q *Queries) GetBadgeDefinitionByKey(ctx context.Context, key string) (BadgeDefinitionRow, error) {
	return scanBadgeDefinition(q.db.QueryRowContext(ctx,
		`SELECT `+badgeDefCols+` FROM badge_definitions WHERE key = ?`, key).Scan)
}

// CreateBadgeDefinitionParams holds fields for inserting a new custom badge definition.
type CreateBadgeDefinitionParams struct {
	Key         string
	Title       string
	Description string
	Tier        int
	Icon        string
}

// CreateBadgeDefinition inserts a new admin-created badge definition and returns its ID.
func (q *Queries) CreateBadgeDefinition(ctx context.Context, p CreateBadgeDefinitionParams) (int64, error) {
	res, err := q.db.ExecContext(ctx,
		`INSERT INTO badge_definitions (key, title, description, tier, icon, is_hardcoded) VALUES (?, ?, ?, ?, ?, 0)`,
		p.Key, p.Title, p.Description, p.Tier, p.Icon)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateBadgeDefinitionParams holds the mutable fields for updating a badge definition.
type UpdateBadgeDefinitionParams struct {
	Key         string
	Title       string
	Description string
	Tier        int
	Icon        string
}

// UpdateBadgeDefinition updates the editable fields of a badge definition.
func (q *Queries) UpdateBadgeDefinition(ctx context.Context, p UpdateBadgeDefinitionParams) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE badge_definitions SET title = ?, description = ?, tier = ?, icon = ? WHERE key = ?`,
		p.Title, p.Description, p.Tier, p.Icon, p.Key)
	return err
}
"""

with open(target, 'a') as f:
    f.write(addition)

print("Done")
