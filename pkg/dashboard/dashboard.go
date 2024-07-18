package dashboard

// Dashboard represents the structure of the dashboard table.
type Dashboard struct {
	ID          int64
	GrafanaID   int64
	UID         string
	Title       string
	Url         string
	Slug        string
	Description string
}

// initDash returns the CREATE query string for the dashboard table.
func initDash() string {
	return `
    CREATE TABLE IF NOT EXISTS dashboard(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        grafana_id INTEGER NOT NULL UNIQUE,
        uid STRING NOT NULL,
        title STRING NOT NULL,
        url STRING NOT NULL,
        slug STRING,
        description STRING
    );
    `
}

// Owner represents the owner of the dashboard.
type Owner struct {
	ID   int64
	Name string
}

// initOwner returns the CREATE query string for the owner table.
func initOwner() string {
	return `
    CREATE TABLE IF NOT EXISTS owner(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name STRING NOT NULL UNIQUE
    );
    `
}

// Folder holds information about the dashboard folder.
type Folder struct {
	ID    int64
	Title string
	Url   string
}

// initFolder returns the CREATE query string for the folder table.
func initFolder() string {
	return `
    CREATE TABLE IF NOT EXISTS folder(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title STRING NOT NULL,
        url STRING NOT NULL
    );
    `
}

// Tag represents a tag of any kind.
type Tag struct {
	ID         int64
	Name       string
	Prefix     string
	Slug       string
	Definition string
	Level      int64
}

// initTag returns the CREATE query string for the tag table.
func initTag() string {
	return `
    CREATE TABLE IF NOT EXISTS tag(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name STRING NOT NULL,
        prefix STRING,
        slug STRING,
        definition STRING,
        level INTEGER DEFAULT 0
    );
    `
}
