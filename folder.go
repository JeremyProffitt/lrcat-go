package lrcat

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// RootFolder represents a root folder in the Lightroom catalog
type RootFolder struct {
	ID           int64
	UUID         string
	AbsolutePath string
	Name         string
}

// Folder represents a folder within a root folder
type Folder struct {
	ID           int64
	UUID         string
	RootFolderID int64
	ParentID     *int64
	PathFromRoot string
}

// AddRootFolder adds a new root folder to the catalog.
// The path should be an absolute path to the folder.
func (c *Catalog) AddRootFolder(absolutePath string) (*RootFolder, error) {
	// Normalize path separators
	absolutePath = normalizePath(absolutePath)

	// Ensure path ends with separator
	if !strings.HasSuffix(absolutePath, "/") {
		absolutePath += "/"
	}

	// Extract folder name
	name := filepath.Base(strings.TrimSuffix(absolutePath, "/"))

	uuid := NewUUID()
	result, err := c.db.Exec(
		`INSERT INTO AgLibraryRootFolder (id_global, absolutePath, name, relativePathFromCatalog)
		 VALUES (?, ?, ?, ?)`,
		uuid, absolutePath, name, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add root folder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get root folder ID: %w", err)
	}

	return &RootFolder{
		ID:           id,
		UUID:         uuid,
		AbsolutePath: absolutePath,
		Name:         name,
	}, nil
}

// GetRootFolder retrieves a root folder by its ID
func (c *Catalog) GetRootFolder(id int64) (*RootFolder, error) {
	rf := &RootFolder{}
	err := c.db.QueryRow(
		`SELECT id_local, id_global, absolutePath, name FROM AgLibraryRootFolder WHERE id_local = ?`,
		id,
	).Scan(&rf.ID, &rf.UUID, &rf.AbsolutePath, &rf.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("root folder not found: %d", id)
		}
		return nil, err
	}
	return rf, nil
}

// GetRootFolderByPath retrieves a root folder by its absolute path
func (c *Catalog) GetRootFolderByPath(absolutePath string) (*RootFolder, error) {
	absolutePath = normalizePath(absolutePath)
	if !strings.HasSuffix(absolutePath, "/") {
		absolutePath += "/"
	}

	rf := &RootFolder{}
	err := c.db.QueryRow(
		`SELECT id_local, id_global, absolutePath, name FROM AgLibraryRootFolder WHERE absolutePath = ?`,
		absolutePath,
	).Scan(&rf.ID, &rf.UUID, &rf.AbsolutePath, &rf.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rf, nil
}

// ListRootFolders returns all root folders in the catalog
func (c *Catalog) ListRootFolders() ([]*RootFolder, error) {
	rows, err := c.db.Query(
		`SELECT id_local, id_global, absolutePath, name FROM AgLibraryRootFolder ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*RootFolder
	for rows.Next() {
		rf := &RootFolder{}
		if err := rows.Scan(&rf.ID, &rf.UUID, &rf.AbsolutePath, &rf.Name); err != nil {
			return nil, err
		}
		folders = append(folders, rf)
	}
	return folders, rows.Err()
}

// AddFolder adds a new folder within a root folder.
// pathFromRoot is the relative path from the root folder (e.g., "2024/January/")
func (c *Catalog) AddFolder(rootFolderID int64, pathFromRoot string) (*Folder, error) {
	// Normalize path
	pathFromRoot = normalizePath(pathFromRoot)

	// Ensure path ends with separator if not empty
	if pathFromRoot != "" && !strings.HasSuffix(pathFromRoot, "/") {
		pathFromRoot += "/"
	}

	uuid := NewUUID()
	result, err := c.db.Exec(
		`INSERT INTO AgLibraryFolder (id_global, rootFolder, pathFromRoot, parentId, visibility)
		 VALUES (?, ?, ?, ?, ?)`,
		uuid, rootFolderID, pathFromRoot, nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add folder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %w", err)
	}

	return &Folder{
		ID:           id,
		UUID:         uuid,
		RootFolderID: rootFolderID,
		PathFromRoot: pathFromRoot,
	}, nil
}

// GetFolder retrieves a folder by its ID
func (c *Catalog) GetFolder(id int64) (*Folder, error) {
	f := &Folder{}
	var parentID sql.NullInt64
	err := c.db.QueryRow(
		`SELECT id_local, id_global, rootFolder, pathFromRoot, parentId FROM AgLibraryFolder WHERE id_local = ?`,
		id,
	).Scan(&f.ID, &f.UUID, &f.RootFolderID, &f.PathFromRoot, &parentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("folder not found: %d", id)
		}
		return nil, err
	}
	if parentID.Valid {
		f.ParentID = &parentID.Int64
	}
	return f, nil
}

// GetOrCreateFolder gets an existing folder or creates it if it doesn't exist
func (c *Catalog) GetOrCreateFolder(rootFolderID int64, pathFromRoot string) (*Folder, error) {
	pathFromRoot = normalizePath(pathFromRoot)
	if pathFromRoot != "" && !strings.HasSuffix(pathFromRoot, "/") {
		pathFromRoot += "/"
	}

	// Try to find existing folder
	f := &Folder{}
	var parentID sql.NullInt64
	err := c.db.QueryRow(
		`SELECT id_local, id_global, rootFolder, pathFromRoot, parentId FROM AgLibraryFolder
		 WHERE rootFolder = ? AND pathFromRoot = ?`,
		rootFolderID, pathFromRoot,
	).Scan(&f.ID, &f.UUID, &f.RootFolderID, &f.PathFromRoot, &parentID)

	if err == nil {
		if parentID.Valid {
			f.ParentID = &parentID.Int64
		}
		return f, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new folder
	return c.AddFolder(rootFolderID, pathFromRoot)
}

// ListFolders returns all folders under a root folder
func (c *Catalog) ListFolders(rootFolderID int64) ([]*Folder, error) {
	rows, err := c.db.Query(
		`SELECT id_local, id_global, rootFolder, pathFromRoot, parentId FROM AgLibraryFolder
		 WHERE rootFolder = ? ORDER BY pathFromRoot`,
		rootFolderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*Folder
	for rows.Next() {
		f := &Folder{}
		var parentID sql.NullInt64
		if err := rows.Scan(&f.ID, &f.UUID, &f.RootFolderID, &f.PathFromRoot, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			f.ParentID = &parentID.Int64
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// normalizePath converts Windows backslashes to forward slashes
func normalizePath(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	return path
}
