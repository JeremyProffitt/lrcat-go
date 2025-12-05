package lrcat

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Keyword represents a keyword in the Lightroom catalog
type Keyword struct {
	ID             int64
	UUID           string
	Name           string
	LCName         string
	ParentID       *int64
	Genealogy      string
	IncludeOnExport bool
}

// AddKeyword adds a new keyword to the catalog
func (c *Catalog) AddKeyword(name string, parentID *int64) (*Keyword, error) {
	uuid := NewUUID()
	lcName := strings.ToLower(name)
	dateCreated := FormatCaptureTime(time.Now())

	// Build genealogy
	genealogy := ""
	if parentID != nil {
		parent, err := c.GetKeyword(*parentID)
		if err != nil {
			return nil, fmt.Errorf("parent keyword not found: %w", err)
		}
		genealogy = parent.Genealogy
	}

	result, err := c.db.Exec(
		`INSERT INTO AgLibraryKeyword (id_global, name, lc_name, parent, genealogy, dateCreated, includeOnExport, includeParents, includeSynonyms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid, name, lcName, parentID, genealogy, dateCreated, 1, 1, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add keyword: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get keyword ID: %w", err)
	}

	// Update genealogy to include the new ID
	newGenealogy := genealogy
	if newGenealogy != "" {
		newGenealogy += "/"
	}
	newGenealogy += fmt.Sprintf("%d", id)

	_, err = c.db.Exec(`UPDATE AgLibraryKeyword SET genealogy = ? WHERE id_local = ?`, newGenealogy, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update genealogy: %w", err)
	}

	return &Keyword{
		ID:             id,
		UUID:           uuid,
		Name:           name,
		LCName:         lcName,
		ParentID:       parentID,
		Genealogy:      newGenealogy,
		IncludeOnExport: true,
	}, nil
}

// GetKeyword retrieves a keyword by its ID
func (c *Catalog) GetKeyword(id int64) (*Keyword, error) {
	kw := &Keyword{}
	var parentID sql.NullInt64
	var includeOnExport int

	err := c.db.QueryRow(
		`SELECT id_local, id_global, name, lc_name, parent, genealogy, includeOnExport
		 FROM AgLibraryKeyword WHERE id_local = ?`,
		id,
	).Scan(&kw.ID, &kw.UUID, &kw.Name, &kw.LCName, &parentID, &kw.Genealogy, &includeOnExport)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("keyword not found: %d", id)
		}
		return nil, err
	}

	if parentID.Valid {
		kw.ParentID = &parentID.Int64
	}
	kw.IncludeOnExport = includeOnExport == 1

	return kw, nil
}

// GetKeywordByName retrieves a keyword by its name (case-insensitive)
func (c *Catalog) GetKeywordByName(name string) (*Keyword, error) {
	lcName := strings.ToLower(name)
	kw := &Keyword{}
	var parentID sql.NullInt64
	var includeOnExport int

	err := c.db.QueryRow(
		`SELECT id_local, id_global, name, lc_name, parent, genealogy, includeOnExport
		 FROM AgLibraryKeyword WHERE lc_name = ?`,
		lcName,
	).Scan(&kw.ID, &kw.UUID, &kw.Name, &kw.LCName, &parentID, &kw.Genealogy, &includeOnExport)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if parentID.Valid {
		kw.ParentID = &parentID.Int64
	}
	kw.IncludeOnExport = includeOnExport == 1

	return kw, nil
}

// GetOrCreateKeyword gets an existing keyword or creates it if it doesn't exist
func (c *Catalog) GetOrCreateKeyword(name string, parentID *int64) (*Keyword, error) {
	kw, err := c.GetKeywordByName(name)
	if err != nil {
		return nil, err
	}
	if kw != nil {
		return kw, nil
	}
	return c.AddKeyword(name, parentID)
}

// ListKeywords returns all keywords in the catalog
func (c *Catalog) ListKeywords() ([]*Keyword, error) {
	rows, err := c.db.Query(
		`SELECT id_local, id_global, name, lc_name, parent, genealogy, includeOnExport
		 FROM AgLibraryKeyword ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keywords []*Keyword
	for rows.Next() {
		kw := &Keyword{}
		var parentID sql.NullInt64
		var includeOnExport int

		if err := rows.Scan(&kw.ID, &kw.UUID, &kw.Name, &kw.LCName, &parentID, &kw.Genealogy, &includeOnExport); err != nil {
			return nil, err
		}

		if parentID.Valid {
			kw.ParentID = &parentID.Int64
		}
		kw.IncludeOnExport = includeOnExport == 1

		keywords = append(keywords, kw)
	}
	return keywords, rows.Err()
}

// AddKeywordToImage associates a keyword with an image
func (c *Catalog) AddKeywordToImage(imageID, keywordID int64) error {
	_, err := c.db.Exec(
		`INSERT OR IGNORE INTO AgLibraryKeywordImage (image, tag) VALUES (?, ?)`,
		imageID, keywordID,
	)
	if err != nil {
		return fmt.Errorf("failed to add keyword to image: %w", err)
	}

	// Update keyword last applied time
	_, err = c.db.Exec(
		`UPDATE AgLibraryKeyword SET lastApplied = ? WHERE id_local = ?`,
		ToLightroomTimestamp(time.Now()), keywordID,
	)

	return err
}

// RemoveKeywordFromImage removes a keyword association from an image
func (c *Catalog) RemoveKeywordFromImage(imageID, keywordID int64) error {
	_, err := c.db.Exec(
		`DELETE FROM AgLibraryKeywordImage WHERE image = ? AND tag = ?`,
		imageID, keywordID,
	)
	return err
}

// GetImageKeywords returns all keywords associated with an image
func (c *Catalog) GetImageKeywords(imageID int64) ([]*Keyword, error) {
	rows, err := c.db.Query(
		`SELECT k.id_local, k.id_global, k.name, k.lc_name, k.parent, k.genealogy, k.includeOnExport
		 FROM AgLibraryKeyword k
		 JOIN AgLibraryKeywordImage ki ON k.id_local = ki.tag
		 WHERE ki.image = ?
		 ORDER BY k.name`,
		imageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keywords []*Keyword
	for rows.Next() {
		kw := &Keyword{}
		var parentID sql.NullInt64
		var includeOnExport int

		if err := rows.Scan(&kw.ID, &kw.UUID, &kw.Name, &kw.LCName, &parentID, &kw.Genealogy, &includeOnExport); err != nil {
			return nil, err
		}

		if parentID.Valid {
			kw.ParentID = &parentID.Int64
		}
		kw.IncludeOnExport = includeOnExport == 1

		keywords = append(keywords, kw)
	}
	return keywords, rows.Err()
}

// GetKeywordImages returns all images associated with a keyword
func (c *Catalog) GetKeywordImages(keywordID int64) ([]*Image, error) {
	rows, err := c.db.Query(
		`SELECT i.id_local, i.id_global, i.rootFile, i.captureTime, i.rating, i.colorLabels, i.pick,
		        i.fileFormat, i.fileWidth, i.fileHeight, i.orientation
		 FROM Adobe_images i
		 JOIN AgLibraryKeywordImage ki ON i.id_local = ki.image
		 WHERE ki.tag = ?
		 ORDER BY i.captureTime`,
		keywordID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		img := &Image{}
		var captureTimeStr sql.NullString
		var rating sql.NullInt64
		var width, height, orientation sql.NullInt64

		if err := rows.Scan(&img.ID, &img.UUID, &img.FileID, &captureTimeStr, &rating, &img.ColorLabel, &img.Pick,
			&img.FileFormat, &width, &height, &orientation); err != nil {
			return nil, err
		}

		if captureTimeStr.Valid {
			img.CaptureTime, _ = time.Parse("2006-01-02T15:04:05", captureTimeStr.String)
		}
		if rating.Valid {
			r := int(rating.Int64)
			img.Rating = &r
		}
		if width.Valid {
			w := int(width.Int64)
			img.Width = &w
		}
		if height.Valid {
			h := int(height.Int64)
			img.Height = &h
		}
		if orientation.Valid {
			o := int(orientation.Int64)
			img.Orientation = &o
		}

		images = append(images, img)
	}
	return images, rows.Err()
}

// CreateHierarchicalKeywords creates a hierarchy of keywords from a path like "People/Family/John"
func (c *Catalog) CreateHierarchicalKeywords(path string) (*Keyword, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty keyword path")
	}

	var parentID *int64
	var lastKeyword *Keyword

	for _, name := range parts {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		kw, err := c.GetOrCreateKeyword(name, parentID)
		if err != nil {
			return nil, err
		}
		parentID = &kw.ID
		lastKeyword = kw
	}

	return lastKeyword, nil
}
