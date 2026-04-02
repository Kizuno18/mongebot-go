// Package account - SQLite repository adapter for profile persistence.
package account

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/storage"
)

// SQLiteRepo adapts storage.ProfileRepo to the Repository interface.
type SQLiteRepo struct {
	repo *storage.ProfileRepo
}

// NewSQLiteRepo creates a new SQLite repository adapter.
func NewSQLiteRepo(db *storage.DB) *SQLiteRepo {
	return &SQLiteRepo{
		repo: storage.NewProfileRepo(db),
	}
}

// Create inserts a new profile.
func (r *SQLiteRepo) Create(ctx context.Context, p *Profile) error {
	row := r.profileToRow(p)
	return r.repo.Create(ctx, row)
}

// GetByID retrieves a profile by its ID.
func (r *SQLiteRepo) GetByID(ctx context.Context, id string) (*Profile, error) {
	row, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToProfile(row)
}

// GetActive returns the currently active profile.
func (r *SQLiteRepo) GetActive(ctx context.Context) (*Profile, error) {
	row, err := r.repo.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return r.rowToProfile(row)
}

// List returns all profiles.
func (r *SQLiteRepo) List(ctx context.Context) ([]*Profile, error) {
	rows, err := r.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	profiles := make([]*Profile, 0, len(rows))
	for _, row := range rows {
		p, err := r.rowToProfile(&row)
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// Update modifies an existing profile.
func (r *SQLiteRepo) Update(ctx context.Context, p *Profile) error {
	row := r.profileToRow(p)
	return r.repo.Update(ctx, row)
}

// Delete removes a profile by ID.
func (r *SQLiteRepo) Delete(ctx context.Context, id string) error {
	return r.repo.Delete(ctx, id)
}

// SetActive deactivates all profiles then activates the specified one.
func (r *SQLiteRepo) SetActive(ctx context.Context, id string) error {
	return r.repo.SetActive(ctx, id)
}

// ExistsByChannel checks if a profile already exists for the given platform+channel.
func (r *SQLiteRepo) ExistsByChannel(ctx context.Context, platform, channel string) (bool, error) {
	return r.repo.ExistsByChannel(ctx, platform, channel)
}

// profileToRow converts a Profile to a ProfileRow.
func (r *SQLiteRepo) profileToRow(p *Profile) *storage.ProfileRow {
	row := &storage.ProfileRow{
		ID:        p.ID,
		Name:      p.Name,
		Platform:  p.Platform,
		Channel:   p.Channel,
		Active:    p.Active,
		MaxWorkers: p.MaxWorkers,
		ProxyTag:  &p.ProxyTag,
		Notes:     &p.Notes,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}

	// Serialize Features
	if p.Features != nil {
		featuresJSON, _ := json.Marshal(p.Features)
		s := string(featuresJSON)
		row.Features = &s
	}

	// Serialize TokenIDs
	if len(p.TokenIDs) > 0 {
		tokenIDsJSON, _ := json.Marshal(p.TokenIDs)
		s := string(tokenIDsJSON)
		row.TokenIDs = &s
	}

	return row
}

// rowToProfile converts a ProfileRow to a Profile.
func (r *SQLiteRepo) rowToProfile(row *storage.ProfileRow) (*Profile, error) {
	p := &Profile{
		ID:        row.ID,
		Name:      row.Name,
		Platform:  row.Platform,
		Channel:   row.Channel,
		Active:    row.Active,
		MaxWorkers: row.MaxWorkers,
	}

	if row.CreatedAt != "" {
		p.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	}
	if row.UpdatedAt != "" {
		p.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
	}

	if row.Features != nil && *row.Features != "" {
		var features FeatureOverride
		if err := json.Unmarshal([]byte(*row.Features), &features); err == nil {
			p.Features = &features
		}
	}

	if row.TokenIDs != nil && *row.TokenIDs != "" {
		json.Unmarshal([]byte(*row.TokenIDs), &p.TokenIDs)
	}

	if row.ProxyTag != nil {
		p.ProxyTag = *row.ProxyTag
	}
	if row.Notes != nil {
		p.Notes = *row.Notes
	}

	return p, nil
}
