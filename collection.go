package lrcat

import (
	"database/sql"
	"fmt"
	"time"
)

// CollectionType represents the type of collection
type CollectionType string

const (
	// CollectionTypeStandard is a regular collection
	CollectionTypeStandard CollectionType = "com.adobe.ag.library.collection"
	// CollectionTypeSmart is a smart collection
	CollectionTypeSmart CollectionType = "com.adobe.ag.library.smart_collection"
	// CollectionTypeGroup is a collection set/group
	CollectionTypeGroup CollectionType = "com.adobe.ag.library.group"
)

// Collection represents a collection in the Lightroom catalog
type Collection struct {
	ID         int64
	Name       string
	CreationID CollectionType
	ParentID   *int64
	Genealogy  string
	ImageCount *int
}

// AddCollection adds a new collection to the catalog
func (c *Catalog) AddCollection(name string, collectionType CollectionType, parentID *int64) (*Collection, error) {
	// Build genealogy
	genealogy := ""
	if parentID != nil {
		parent, err := c.GetCollection(*parentID)
		if err != nil {
			return nil, fmt.Errorf("parent collection not found: %w", err)
		}
		genealogy = parent.Genealogy
	}

	result, err := c.db.Exec(
		`INSERT INTO AgLibraryCollection (creationId, name, parent, genealogy, systemOnly)
		 VALUES (?, ?, ?, ?, ?)`,
		string(collectionType), name, parentID, genealogy, "",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add collection: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection ID: %w", err)
	}

	// Update genealogy to include the new ID
	newGenealogy := genealogy
	if newGenealogy != "" {
		newGenealogy += "/"
	}
	newGenealogy += fmt.Sprintf("%d", id)

	_, err = c.db.Exec(`UPDATE AgLibraryCollection SET genealogy = ? WHERE id_local = ?`, newGenealogy, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update genealogy: %w", err)
	}

	return &Collection{
		ID:         id,
		Name:       name,
		CreationID: collectionType,
		ParentID:   parentID,
		Genealogy:  newGenealogy,
	}, nil
}

// GetCollection retrieves a collection by its ID
func (c *Catalog) GetCollection(id int64) (*Collection, error) {
	coll := &Collection{}
	var parentID sql.NullInt64
	var imageCount sql.NullInt64
	var creationID string

	err := c.db.QueryRow(
		`SELECT id_local, name, creationId, parent, genealogy, imageCount
		 FROM AgLibraryCollection WHERE id_local = ?`,
		id,
	).Scan(&coll.ID, &coll.Name, &creationID, &parentID, &coll.Genealogy, &imageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("collection not found: %d", id)
		}
		return nil, err
	}

	coll.CreationID = CollectionType(creationID)
	if parentID.Valid {
		coll.ParentID = &parentID.Int64
	}
	if imageCount.Valid {
		count := int(imageCount.Int64)
		coll.ImageCount = &count
	}

	return coll, nil
}

// GetCollectionByName retrieves a collection by its name
func (c *Catalog) GetCollectionByName(name string) (*Collection, error) {
	coll := &Collection{}
	var parentID sql.NullInt64
	var imageCount sql.NullInt64
	var creationID string

	err := c.db.QueryRow(
		`SELECT id_local, name, creationId, parent, genealogy, imageCount
		 FROM AgLibraryCollection WHERE name = ?`,
		name,
	).Scan(&coll.ID, &coll.Name, &creationID, &parentID, &coll.Genealogy, &imageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	coll.CreationID = CollectionType(creationID)
	if parentID.Valid {
		coll.ParentID = &parentID.Int64
	}
	if imageCount.Valid {
		count := int(imageCount.Int64)
		coll.ImageCount = &count
	}

	return coll, nil
}

// ListCollections returns all collections in the catalog
func (c *Catalog) ListCollections() ([]*Collection, error) {
	rows, err := c.db.Query(
		`SELECT id_local, name, creationId, parent, genealogy, imageCount
		 FROM AgLibraryCollection WHERE systemOnly = '' ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*Collection
	for rows.Next() {
		coll := &Collection{}
		var parentID sql.NullInt64
		var imageCount sql.NullInt64
		var creationID string

		if err := rows.Scan(&coll.ID, &coll.Name, &creationID, &parentID, &coll.Genealogy, &imageCount); err != nil {
			return nil, err
		}

		coll.CreationID = CollectionType(creationID)
		if parentID.Valid {
			coll.ParentID = &parentID.Int64
		}
		if imageCount.Valid {
			count := int(imageCount.Int64)
			coll.ImageCount = &count
		}

		collections = append(collections, coll)
	}
	return collections, rows.Err()
}

// AddImageToCollection adds an image to a collection
func (c *Catalog) AddImageToCollection(imageID, collectionID int64) error {
	// Get current max position
	var maxPos sql.NullFloat64
	err := c.db.QueryRow(
		`SELECT MAX(positionInCollection) FROM AgLibraryCollectionImage WHERE collection = ?`,
		collectionID,
	).Scan(&maxPos)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	position := 1.0
	if maxPos.Valid {
		position = maxPos.Float64 + 1.0
	}

	_, err = c.db.Exec(
		`INSERT OR IGNORE INTO AgLibraryCollectionImage (collection, image, pick, positionInCollection)
		 VALUES (?, ?, 0, ?)`,
		collectionID, imageID, position,
	)
	if err != nil {
		return fmt.Errorf("failed to add image to collection: %w", err)
	}

	// Update image count
	return c.updateCollectionImageCount(collectionID)
}

// RemoveImageFromCollection removes an image from a collection
func (c *Catalog) RemoveImageFromCollection(imageID, collectionID int64) error {
	_, err := c.db.Exec(
		`DELETE FROM AgLibraryCollectionImage WHERE image = ? AND collection = ?`,
		imageID, collectionID,
	)
	if err != nil {
		return err
	}

	return c.updateCollectionImageCount(collectionID)
}

// updateCollectionImageCount updates the imageCount field for a collection
func (c *Catalog) updateCollectionImageCount(collectionID int64) error {
	_, err := c.db.Exec(
		`UPDATE AgLibraryCollection SET imageCount = (
			SELECT COUNT(*) FROM AgLibraryCollectionImage WHERE collection = ?
		) WHERE id_local = ?`,
		collectionID, collectionID,
	)
	return err
}

// GetCollectionImages returns all images in a collection
func (c *Catalog) GetCollectionImages(collectionID int64) ([]*Image, error) {
	rows, err := c.db.Query(
		`SELECT i.id_local, i.id_global, i.rootFile, i.captureTime, i.rating, i.colorLabels, i.pick,
		        i.fileFormat, i.fileWidth, i.fileHeight, i.orientation
		 FROM Adobe_images i
		 JOIN AgLibraryCollectionImage ci ON i.id_local = ci.image
		 WHERE ci.collection = ?
		 ORDER BY ci.positionInCollection`,
		collectionID,
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
			img.CaptureTime, _ = parseTime(captureTimeStr.String)
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

// GetImageCollections returns all collections that contain an image
func (c *Catalog) GetImageCollections(imageID int64) ([]*Collection, error) {
	rows, err := c.db.Query(
		`SELECT c.id_local, c.name, c.creationId, c.parent, c.genealogy, c.imageCount
		 FROM AgLibraryCollection c
		 JOIN AgLibraryCollectionImage ci ON c.id_local = ci.collection
		 WHERE ci.image = ?
		 ORDER BY c.name`,
		imageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*Collection
	for rows.Next() {
		coll := &Collection{}
		var parentID sql.NullInt64
		var imageCount sql.NullInt64
		var creationID string

		if err := rows.Scan(&coll.ID, &coll.Name, &creationID, &parentID, &coll.Genealogy, &imageCount); err != nil {
			return nil, err
		}

		coll.CreationID = CollectionType(creationID)
		if parentID.Valid {
			coll.ParentID = &parentID.Int64
		}
		if imageCount.Valid {
			count := int(imageCount.Int64)
			coll.ImageCount = &count
		}

		collections = append(collections, coll)
	}
	return collections, rows.Err()
}

// DeleteCollection deletes a collection (but not the images in it)
func (c *Catalog) DeleteCollection(collectionID int64) error {
	// First delete all image associations
	_, err := c.db.Exec(`DELETE FROM AgLibraryCollectionImage WHERE collection = ?`, collectionID)
	if err != nil {
		return err
	}

	// Delete collection content
	_, err = c.db.Exec(`DELETE FROM AgLibraryCollectionContent WHERE collection = ?`, collectionID)
	if err != nil {
		return err
	}

	// Delete the collection
	_, err = c.db.Exec(`DELETE FROM AgLibraryCollection WHERE id_local = ?`, collectionID)
	return err
}

// parseTime is a helper to parse Lightroom time format
func parseTime(s string) (t time.Time, err error) {
	layouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05-07:00",
	}
	for _, layout := range layouts {
		t, err = time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
