package rules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Rule struct {
	ID               int64
	Name             string
	AlbumName        string
	CameraMake       string
	CameraModel      string
	LensModel        string
	MediaType        string
	City             string
	State            string
	Country          string
	TakenAfter       string
	TakenBefore      string
	OriginalFileName string
	Description      string
	Interval         int
	Enabled          bool
	LastSync         time.Time
	LastCount        int
	TotalCount       int
	CreatedAt        time.Time
}

func (r Rule) filterParts() []string {
	var parts []string
	if r.CameraMake != "" || r.CameraModel != "" {
		parts = append(parts, "camera:"+r.CameraMake+"/"+r.CameraModel)
	}
	if r.LensModel != "" {
		parts = append(parts, "lens:"+r.LensModel)
	}
	if r.MediaType != "" {
		parts = append(parts, "type:"+r.MediaType)
	}
	if r.City != "" {
		parts = append(parts, "city:"+r.City)
	}
	if r.State != "" {
		parts = append(parts, "state:"+r.State)
	}
	if r.Country != "" {
		parts = append(parts, "country:"+r.Country)
	}
	if r.TakenAfter != "" {
		parts = append(parts, "after:"+r.TakenAfter)
	}
	if r.TakenBefore != "" {
		parts = append(parts, "before:"+r.TakenBefore)
	}
	if r.OriginalFileName != "" {
		parts = append(parts, "file:"+r.OriginalFileName)
	}
	if r.Description != "" {
		parts = append(parts, "desc:"+r.Description)
	}
	return parts
}

func (r Rule) HasFilters() bool {
	return len(r.filterParts()) > 0
}

func (r Rule) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.AlbumName) == "" {
		return errors.New("album name is required")
	}
	if !r.HasFilters() {
		return errors.New("at least one filter is required")
	}
	return nil
}

func (r Rule) FilterSummary() string {
	parts := r.filterParts()
	if len(parts) == 0 {
		return "no filters"
	}
	return strings.Join(parts, ", ")
}

func (r Rule) LastSyncAgo() string {
	if r.LastSync.IsZero() {
		return "never"
	}
	d := time.Since(r.LastSync)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		album_name TEXT NOT NULL,
		camera_make TEXT DEFAULT '',
		camera_model TEXT DEFAULT '',
		lens_model TEXT DEFAULT '',
		media_type TEXT DEFAULT '',
		city TEXT DEFAULT '',
		state TEXT DEFAULT '',
		country TEXT DEFAULT '',
		taken_after TEXT DEFAULT '',
		taken_before TEXT DEFAULT '',
		original_file_name TEXT DEFAULT '',
		description TEXT DEFAULT '',
		interval_sec INTEGER DEFAULT 3600,
		enabled INTEGER DEFAULT 1,
		last_sync DATETIME,
		last_count INTEGER DEFAULT 0,
		total_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return nil, err
	}
	for _, col := range []string{"original_file_name", "description"} {
		db.Exec(`ALTER TABLE rules ADD COLUMN ` + col + ` TEXT DEFAULT ''`)
	}
	return &Store{db: db}, nil
}

func (s *Store) List() ([]Rule, error) {
	rows, err := s.db.Query(`SELECT id, name, album_name, camera_make, camera_model,
		lens_model, media_type, city, state, country, taken_after, taken_before,
		COALESCE(original_file_name,''), COALESCE(description,''),
		interval_sec, enabled, COALESCE(last_sync,''), last_count, total_count, created_at
		FROM rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var lastSync, createdAt string
		err := rows.Scan(&r.ID, &r.Name, &r.AlbumName, &r.CameraMake, &r.CameraModel,
			&r.LensModel, &r.MediaType, &r.City, &r.State, &r.Country,
			&r.TakenAfter, &r.TakenBefore, &r.OriginalFileName, &r.Description,
			&r.Interval, &r.Enabled,
			&lastSync, &r.LastCount, &r.TotalCount, &createdAt)
		if err != nil {
			return nil, err
		}
		r.LastSync, _ = time.Parse(time.RFC3339, lastSync)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Store) Get(id int64) (Rule, error) {
	var r Rule
	var lastSync, createdAt string
	err := s.db.QueryRow(`SELECT id, name, album_name, camera_make, camera_model,
		lens_model, media_type, city, state, country, taken_after, taken_before,
		COALESCE(original_file_name,''), COALESCE(description,''),
		interval_sec, enabled, COALESCE(last_sync,''), last_count, total_count, created_at
		FROM rules WHERE id = ?`, id).Scan(&r.ID, &r.Name, &r.AlbumName, &r.CameraMake,
		&r.CameraModel, &r.LensModel, &r.MediaType, &r.City, &r.State, &r.Country,
		&r.TakenAfter, &r.TakenBefore, &r.OriginalFileName, &r.Description,
		&r.Interval, &r.Enabled,
		&lastSync, &r.LastCount, &r.TotalCount, &createdAt)
	r.LastSync, _ = time.Parse(time.RFC3339, lastSync)
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return r, err
}

func (s *Store) Create(r *Rule) error {
	res, err := s.db.Exec(`INSERT INTO rules (name, album_name, camera_make, camera_model,
		lens_model, media_type, city, state, country, taken_after, taken_before,
		original_file_name, description, interval_sec, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Name, r.AlbumName, r.CameraMake, r.CameraModel, r.LensModel, r.MediaType,
		r.City, r.State, r.Country, r.TakenAfter, r.TakenBefore,
		r.OriginalFileName, r.Description, r.Interval, r.Enabled)
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) Update(r *Rule) error {
	_, err := s.db.Exec(`UPDATE rules SET name=?, album_name=?, camera_make=?, camera_model=?,
		lens_model=?, media_type=?, city=?, state=?, country=?, taken_after=?, taken_before=?,
		original_file_name=?, description=?, interval_sec=?, enabled=? WHERE id=?`,
		r.Name, r.AlbumName, r.CameraMake, r.CameraModel, r.LensModel, r.MediaType,
		r.City, r.State, r.Country, r.TakenAfter, r.TakenBefore,
		r.OriginalFileName, r.Description, r.Interval, r.Enabled, r.ID)
	return err
}

func (s *Store) UpdateSync(id int64, lastCount, totalCount int) error {
	_, err := s.db.Exec(`UPDATE rules SET last_sync=?, last_count=?, total_count=? WHERE id=?`,
		time.Now().UTC().Format(time.RFC3339), lastCount, totalCount, id)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM rules WHERE id=?`, id)
	return err
}
