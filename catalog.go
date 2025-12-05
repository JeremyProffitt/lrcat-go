// Package lrcat provides functionality for creating and manipulating
// Adobe Lightroom Classic catalog files (.lrcat).
//
// Lightroom catalogs are SQLite databases containing references to images,
// their metadata, develop settings, collections, and more.
package lrcat

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// LightroomEpoch is the reference date for Lightroom timestamps (2001-01-01 00:00:00 UTC)
var LightroomEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// Catalog represents a Lightroom catalog database
type Catalog struct {
	db       *sql.DB
	path     string
	readOnly bool
}

// CatalogOptions contains options for creating or opening a catalog
type CatalogOptions struct {
	// ReadOnly opens the catalog in read-only mode
	ReadOnly bool
}

// NewCatalog creates a new Lightroom catalog at the specified path.
// If the file already exists, it will be overwritten.
func NewCatalog(path string) (*Catalog, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Remove existing file if present
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("failed to remove existing catalog: %w", err)
		}
	}

	// Create new database
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog: %w", err)
	}

	catalog := &Catalog{
		db:   db,
		path: path,
	}

	// Initialize schema
	if err := catalog.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return catalog, nil
}

// OpenCatalog opens an existing Lightroom catalog
func OpenCatalog(path string, opts *CatalogOptions) (*Catalog, error) {
	if opts == nil {
		opts = &CatalogOptions{}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("catalog does not exist: %s", path)
	}

	dsn := path
	if opts.ReadOnly {
		dsn = fmt.Sprintf("file:%s?mode=ro&cache=private&immutable=1", path)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open catalog: %w", err)
	}

	catalog := &Catalog{
		db:       db,
		path:     path,
		readOnly: opts.ReadOnly,
	}

	return catalog, nil
}

// Close closes the catalog database connection
func (c *Catalog) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Path returns the file path of the catalog
func (c *Catalog) Path() string {
	return c.path
}

// DB returns the underlying database connection for advanced operations
func (c *Catalog) DB() *sql.DB {
	return c.db
}

// initSchema creates all required tables and initializes variables
func (c *Catalog) initSchema() error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create all tables
	for _, stmt := range schemaSQL {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w\nStatement: %s", err, stmt)
		}
	}

	// Insert required variables
	for name, value := range requiredVariables {
		_, err := tx.Exec(
			`INSERT INTO Adobe_variablesTable (id_global, name, type, value) VALUES (?, ?, ?, ?)`,
			NewUUID(), name, "string", value,
		)
		if err != nil {
			return fmt.Errorf("failed to insert variable %s: %w", name, err)
		}
	}

	return tx.Commit()
}

// GetDBVersion returns the Adobe database version from the catalog
func (c *Catalog) GetDBVersion() (string, error) {
	var version string
	err := c.db.QueryRow(
		`SELECT value FROM Adobe_variablesTable WHERE name = 'Adobe_DBVersion'`,
	).Scan(&version)
	if err != nil {
		return "", fmt.Errorf("failed to get DB version: %w", err)
	}
	return version, nil
}

// NewUUID generates a new UUID suitable for Lightroom's id_global fields
func NewUUID() string {
	return strings.ToUpper(uuid.New().String())
}

// ToLightroomTimestamp converts a time.Time to Lightroom's timestamp format
// (seconds since 2001-01-01 00:00:00 UTC)
func ToLightroomTimestamp(t time.Time) float64 {
	return t.Sub(LightroomEpoch).Seconds()
}

// FromLightroomTimestamp converts a Lightroom timestamp to time.Time
func FromLightroomTimestamp(ts float64) time.Time {
	return LightroomEpoch.Add(time.Duration(ts * float64(time.Second)))
}

// FormatCaptureTime formats a time for Lightroom's captureTime field
func FormatCaptureTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

// ImageCount returns the total number of images in the catalog
func (c *Catalog) ImageCount() (int, error) {
	var count int
	err := c.db.QueryRow(`SELECT COUNT(*) FROM Adobe_images`).Scan(&count)
	return count, err
}

// FolderCount returns the total number of folders in the catalog
func (c *Catalog) FolderCount() (int, error) {
	var count int
	err := c.db.QueryRow(`SELECT COUNT(*) FROM AgLibraryFolder`).Scan(&count)
	return count, err
}

// RootFolderCount returns the total number of root folders in the catalog
func (c *Catalog) RootFolderCount() (int, error) {
	var count int
	err := c.db.QueryRow(`SELECT COUNT(*) FROM AgLibraryRootFolder`).Scan(&count)
	return count, err
}
