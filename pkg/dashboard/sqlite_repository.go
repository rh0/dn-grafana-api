package dashboard

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

var (
	ErrDuplicate = errors.New("record already exists")
	ErrNotExist  = errors.New("row does not exist")
)

type SQLiteRepository struct {
	db    *sql.DB
	inits []func() string
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
		inits: []func() string{
			initDash,
			initOwner,
			initFolder,
			initTag,
		},
	}
}

// TabelInit runs through the table create queries to setup the tables.
func (r *SQLiteRepository) TabelInit() error {
	for _, f := range r.inits {
		query := f()
		_, err := r.db.Exec(query)
		if err != nil {
			return fmt.Errorf("There was an error executing query: %s \n %w", query, err)
		}
	}

	return nil
}

// InsetTag inserts a single tag record into the database.
func (r *SQLiteRepository) InsetTag(t Tag) (*Tag, error) {
	res, err := r.db.Exec("INSERT INTO tag(name, slug, definition, level) values(?,?,?,?)", t.Name, t.Slug, t.Definition, t.Level)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return nil, ErrDuplicate
			}

		}
		return nil, err
	}

	// Success! Grab the id from the record we just inserted.
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	// Add the id to the tag we were given.
	t.ID = id

	return &t, nil
}

// InsetTagBulk inserts multiple tags into the databas.
func (r *SQLiteRepository) InsetTagBulk(tags []Tag) error {
	// BEGIN TRANSACTION
	tx, err := r.db.Begin()

	for _, t := range tags {
		_, err = r.db.Exec("INSERT INTO tag(name, slug, definition, level) values(?,?,?,?)", t.Name, t.Slug, t.Definition, t.Level)
	}

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("InsetTagBulk: %w", err)
	}

	// COMMIT
	tx.Commit()

	return nil
}

// InsertDashboard inserts a single dashboard record into the databse.
func (r *SQLiteRepository) InsertDashboard(d Dashboard) (*Dashboard, error) {
	res, err := r.db.Exec(`
        INSERT INTO dashboard(
            grafana_id,
            uid,
            title,
            url,
            slug,
            description
        ) values(?,?,?,?,?,?)
        `, d.GrafanaID, d.UID, d.Title, d.Url, d.Slug, d.Description)

	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return nil, ErrDuplicate
			}

		}
		return nil, err
	}

	// Success! Grab the id from the record we just inserted.
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	// Add the id to the dashboard we were given.
	d.ID = id

	return &d, nil
}

// InsertDashboardBulk inserts multiple dashboards into the database.
func (r *SQLiteRepository) InsertDashboardBulk(dashboards []Dashboard) error {
    // BEGIN TRANSACTION
    tx, err := r.db.Begin()

    for _, d := range dashboards {
        _, err = r.db.Exec(`
            INSERT INTO dashboard(
                grafana_id,
                uid,
                title,
                url,
                slug,
                description
            ) values(?,?,?,?,?,?)
            `, d.GrafanaID, d.UID, d.Title, d.Url, d.Slug, d.Description)
    }

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("InsertDashboardBulkBulk: %w", err)
	}

	// COMMIT
	tx.Commit()

	return nil
}
