package store

import "context"

func (s *Store) ListProfiles(ctx context.Context) ([]Profile, error) {
	const q = `
		SELECT id::text, name, created_at, updated_at
		FROM profiles
		ORDER BY created_at ASC
	`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var profile Profile
		if err := rows.Scan(&profile.ID, &profile.Name, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (s *Store) CreateProfile(ctx context.Context, name string) (Profile, error) {
	const q = `
		INSERT INTO profiles (name)
		VALUES ($1)
		RETURNING id::text, name, created_at, updated_at
	`

	var profile Profile
	err := s.Pool.QueryRow(ctx, q, name).Scan(
		&profile.ID,
		&profile.Name,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	return profile, err
}
