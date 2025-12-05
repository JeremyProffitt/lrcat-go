package lrcat

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Image represents an image in the Lightroom catalog
type Image struct {
	ID          int64
	UUID        string
	FileID      int64
	FolderID    int64
	CaptureTime time.Time
	Rating      *int
	ColorLabel  string
	Pick        int
	FileFormat  string
	Width       *int
	Height      *int
	Orientation *int
}

// ImageFile represents a file record in AgLibraryFile
type ImageFile struct {
	ID               int64
	UUID             string
	FolderID         int64
	BaseName         string
	Extension        string
	OriginalFilename string
}

// ImportSession represents an import session
type ImportSession struct {
	ID         int64
	ImportDate time.Time
	ImageCount int
	Name       string
}

// ImageInput contains the input data for adding an image
type ImageInput struct {
	// FilePath is the absolute path to the image file
	FilePath string
	// CaptureTime is the capture date/time of the image
	CaptureTime time.Time
	// Rating is the star rating (0-5)
	Rating *int
	// ColorLabel is the color label (e.g., "Red", "Yellow", "Green", "Blue", "Purple")
	ColorLabel string
	// Pick status: 0 = unflagged, 1 = picked, -1 = rejected
	Pick int
	// FileFormat (e.g., "JPG", "RAW", "DNG", "TIFF", "PSD", "PNG")
	FileFormat string
	// Width is the image width in pixels
	Width *int
	// Height is the image height in pixels
	Height *int
	// Orientation is the EXIF orientation value
	Orientation *int
}

// AddImage adds a single image to the catalog.
// The image's folder will be created automatically if it doesn't exist.
func (c *Catalog) AddImage(input *ImageInput) (*Image, error) {
	// Normalize the file path
	absPath := normalizePath(input.FilePath)

	// Split path into directory and filename
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Get or create root folder and folder
	rootFolder, folder, err := c.ensureFolderPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure folder path: %w", err)
	}

	// Determine file format
	fileFormat := input.FileFormat
	if fileFormat == "" {
		fileFormat = detectFileFormat(ext)
	}

	// Create file record
	file, err := c.addFile(folder.ID, baseName, ext, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to add file record: %w", err)
	}

	// Create image record
	image, err := c.addImageRecord(file.ID, input, fileFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to add image record: %w", err)
	}

	// Add additional metadata placeholder
	if err := c.addAdditionalMetadata(image.ID); err != nil {
		return nil, fmt.Errorf("failed to add metadata: %w", err)
	}

	_ = rootFolder // Used for folder creation

	return image, nil
}

// AddImages adds multiple images to the catalog in a single transaction.
// Returns the import session and the list of added images.
func (c *Catalog) AddImages(inputs []*ImageInput) (*ImportSession, []*Image, error) {
	if len(inputs) == 0 {
		return nil, nil, fmt.Errorf("no images to add")
	}

	tx, err := c.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create import session
	importSession, err := c.createImportSession(tx, len(inputs))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create import session: %w", err)
	}

	var images []*Image
	for _, input := range inputs {
		image, err := c.addImageInTx(tx, input)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to add image %s: %w", input.FilePath, err)
		}

		// Link image to import
		if err := c.linkImageToImport(tx, image.ID, importSession.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to link image to import: %w", err)
		}

		images = append(images, image)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return importSession, images, nil
}

// ensureFolderPath ensures the folder path exists and returns the root folder and folder
func (c *Catalog) ensureFolderPath(dirPath string) (*RootFolder, *Folder, error) {
	dirPath = normalizePath(dirPath)
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	// Try to find an existing root folder that matches
	rootFolders, err := c.ListRootFolders()
	if err != nil {
		return nil, nil, err
	}

	var matchingRoot *RootFolder
	var pathFromRoot string

	for _, rf := range rootFolders {
		if strings.HasPrefix(dirPath, rf.AbsolutePath) {
			matchingRoot = rf
			pathFromRoot = strings.TrimPrefix(dirPath, rf.AbsolutePath)
			break
		}
	}

	// If no matching root folder, create one at the directory
	if matchingRoot == nil {
		matchingRoot, err = c.AddRootFolder(dirPath)
		if err != nil {
			return nil, nil, err
		}
		pathFromRoot = ""
	}

	// Get or create the folder
	folder, err := c.GetOrCreateFolder(matchingRoot.ID, pathFromRoot)
	if err != nil {
		return nil, nil, err
	}

	return matchingRoot, folder, nil
}

// addFile creates a file record in AgLibraryFile
func (c *Catalog) addFile(folderID int64, baseName, extension, originalFilename string) (*ImageFile, error) {
	uuid := NewUUID()
	idxFilename := baseName + "." + extension
	lcIdxFilename := strings.ToLower(idxFilename)
	lcIdxFilenameExt := strings.ToLower(extension)

	result, err := c.db.Exec(
		`INSERT INTO AgLibraryFile
		 (id_global, folder, baseName, extension, originalFilename, idx_filename, lc_idx_filename, lc_idx_filenameExtension)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid, folderID, baseName, extension, originalFilename, idxFilename, lcIdxFilename, lcIdxFilenameExt,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ImageFile{
		ID:               id,
		UUID:             uuid,
		FolderID:         folderID,
		BaseName:         baseName,
		Extension:        extension,
		OriginalFilename: originalFilename,
	}, nil
}

// addImageRecord creates an image record in Adobe_images
func (c *Catalog) addImageRecord(fileID int64, input *ImageInput, fileFormat string) (*Image, error) {
	uuid := NewUUID()
	captureTimeStr := FormatCaptureTime(input.CaptureTime)

	var rating interface{}
	if input.Rating != nil {
		rating = *input.Rating
	}

	var width, height, orientation interface{}
	if input.Width != nil {
		width = *input.Width
	}
	if input.Height != nil {
		height = *input.Height
	}
	if input.Orientation != nil {
		orientation = *input.Orientation
	}

	result, err := c.db.Exec(
		`INSERT INTO Adobe_images
		 (id_global, rootFile, captureTime, rating, colorLabels, pick, fileFormat, fileWidth, fileHeight, orientation, touchTime)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid, fileID, captureTimeStr, rating, input.ColorLabel, input.Pick, fileFormat, width, height, orientation,
		ToLightroomTimestamp(time.Now()),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Image{
		ID:          id,
		UUID:        uuid,
		FileID:      fileID,
		CaptureTime: input.CaptureTime,
		Rating:      input.Rating,
		ColorLabel:  input.ColorLabel,
		Pick:        input.Pick,
		FileFormat:  fileFormat,
		Width:       input.Width,
		Height:      input.Height,
		Orientation: input.Orientation,
	}, nil
}

// addAdditionalMetadata adds a metadata placeholder for an image
func (c *Catalog) addAdditionalMetadata(imageID int64) error {
	uuid := NewUUID()
	_, err := c.db.Exec(
		`INSERT INTO Adobe_AdditionalMetadata (id_global, image, xmp) VALUES (?, ?, ?)`,
		uuid, imageID, "",
	)
	return err
}

// createImportSession creates a new import session
func (c *Catalog) createImportSession(tx *sql.Tx, imageCount int) (*ImportSession, error) {
	now := time.Now()
	importDate := FormatCaptureTime(now)

	result, err := tx.Exec(
		`INSERT INTO AgLibraryImport (importDate, imageCount) VALUES (?, ?)`,
		importDate, imageCount,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ImportSession{
		ID:         id,
		ImportDate: now,
		ImageCount: imageCount,
	}, nil
}

// linkImageToImport links an image to an import session
func (c *Catalog) linkImageToImport(tx *sql.Tx, imageID, importID int64) error {
	_, err := tx.Exec(
		`INSERT INTO AgLibraryImportImage (image, import) VALUES (?, ?)`,
		imageID, importID,
	)
	return err
}

// addImageInTx adds an image within a transaction
func (c *Catalog) addImageInTx(tx *sql.Tx, input *ImageInput) (*Image, error) {
	absPath := normalizePath(input.FilePath)
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Ensure folder path
	rootFolder, folder, err := c.ensureFolderPath(dir)
	if err != nil {
		return nil, err
	}
	_ = rootFolder

	fileFormat := input.FileFormat
	if fileFormat == "" {
		fileFormat = detectFileFormat(ext)
	}

	// Create file record
	fileUUID := NewUUID()
	idxFilename := baseName + "." + ext
	lcIdxFilename := strings.ToLower(idxFilename)
	lcIdxFilenameExt := strings.ToLower(ext)

	fileResult, err := tx.Exec(
		`INSERT INTO AgLibraryFile
		 (id_global, folder, baseName, extension, originalFilename, idx_filename, lc_idx_filename, lc_idx_filenameExtension)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fileUUID, folder.ID, baseName, ext, filename, idxFilename, lcIdxFilename, lcIdxFilenameExt,
	)
	if err != nil {
		return nil, err
	}

	fileID, err := fileResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Create image record
	imageUUID := NewUUID()
	captureTimeStr := FormatCaptureTime(input.CaptureTime)

	var rating interface{}
	if input.Rating != nil {
		rating = *input.Rating
	}

	var width, height, orientation interface{}
	if input.Width != nil {
		width = *input.Width
	}
	if input.Height != nil {
		height = *input.Height
	}
	if input.Orientation != nil {
		orientation = *input.Orientation
	}

	imageResult, err := tx.Exec(
		`INSERT INTO Adobe_images
		 (id_global, rootFile, captureTime, rating, colorLabels, pick, fileFormat, fileWidth, fileHeight, orientation, touchTime)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		imageUUID, fileID, captureTimeStr, rating, input.ColorLabel, input.Pick, fileFormat, width, height, orientation,
		ToLightroomTimestamp(time.Now()),
	)
	if err != nil {
		return nil, err
	}

	imageID, err := imageResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Add additional metadata
	metaUUID := NewUUID()
	_, err = tx.Exec(
		`INSERT INTO Adobe_AdditionalMetadata (id_global, image, xmp) VALUES (?, ?, ?)`,
		metaUUID, imageID, "",
	)
	if err != nil {
		return nil, err
	}

	return &Image{
		ID:          imageID,
		UUID:        imageUUID,
		FileID:      fileID,
		FolderID:    folder.ID,
		CaptureTime: input.CaptureTime,
		Rating:      input.Rating,
		ColorLabel:  input.ColorLabel,
		Pick:        input.Pick,
		FileFormat:  fileFormat,
		Width:       input.Width,
		Height:      input.Height,
		Orientation: input.Orientation,
	}, nil
}

// GetImage retrieves an image by its ID
func (c *Catalog) GetImage(id int64) (*Image, error) {
	img := &Image{}
	var captureTimeStr sql.NullString
	var rating sql.NullInt64
	var width, height, orientation sql.NullInt64

	err := c.db.QueryRow(
		`SELECT i.id_local, i.id_global, i.rootFile, i.captureTime, i.rating, i.colorLabels, i.pick,
		        i.fileFormat, i.fileWidth, i.fileHeight, i.orientation
		 FROM Adobe_images i WHERE i.id_local = ?`,
		id,
	).Scan(&img.ID, &img.UUID, &img.FileID, &captureTimeStr, &rating, &img.ColorLabel, &img.Pick,
		&img.FileFormat, &width, &height, &orientation)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("image not found: %d", id)
		}
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

	return img, nil
}

// ListImages returns all images in the catalog
func (c *Catalog) ListImages() ([]*Image, error) {
	rows, err := c.db.Query(
		`SELECT i.id_local, i.id_global, i.rootFile, i.captureTime, i.rating, i.colorLabels, i.pick,
		        i.fileFormat, i.fileWidth, i.fileHeight, i.orientation
		 FROM Adobe_images i ORDER BY i.captureTime`,
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

// detectFileFormat determines the Lightroom file format from extension
func detectFileFormat(ext string) string {
	ext = strings.ToUpper(ext)
	switch ext {
	case "JPG", "JPEG":
		return "JPG"
	case "PNG":
		return "PNG"
	case "TIFF", "TIF":
		return "TIFF"
	case "PSD":
		return "PSD"
	case "DNG":
		return "DNG"
	case "CR2", "CR3", "NEF", "ARW", "ORF", "RAF", "RW2", "PEF", "SRW":
		return "RAW"
	case "MP4", "MOV", "AVI", "MKV":
		return "VIDEO"
	default:
		return "JPG"
	}
}

// ImageExists checks if an image file already exists in the catalog
func (c *Catalog) ImageExists(filePath string) (bool, error) {
	absPath := normalizePath(filePath)
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)

	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	var count int
	err := c.db.QueryRow(
		`SELECT COUNT(*) FROM AgLibraryFile f
		 JOIN AgLibraryFolder fo ON f.folder = fo.id_local
		 JOIN AgLibraryRootFolder rf ON fo.rootFolder = rf.id_local
		 WHERE rf.absolutePath || fo.pathFromRoot || f.baseName || '.' || f.extension = ?`,
		absPath,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	// Also check by original filename in the same folder structure
	if count == 0 {
		err = c.db.QueryRow(
			`SELECT COUNT(*) FROM AgLibraryFile f WHERE f.originalFilename = ?`,
			filename,
		).Scan(&count)
	}

	return count > 0, err
}

// ScanDirectory scans a directory for image files and returns ImageInputs
func ScanDirectory(dir string, recursive bool) ([]*ImageInput, error) {
	var inputs []*ImageInput

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if isImageExtension(ext) {
			info, err := d.Info()
			if err != nil {
				return nil // Skip files we can't read
			}

			inputs = append(inputs, &ImageInput{
				FilePath:    path,
				CaptureTime: info.ModTime(),
			})
		}

		return nil
	})

	return inputs, err
}

// isImageExtension checks if the extension is a supported image format
func isImageExtension(ext string) bool {
	supportedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".tiff": true, ".tif": true,
		".psd": true, ".dng": true, ".cr2": true, ".cr3": true, ".nef": true,
		".arw": true, ".orf": true, ".raf": true, ".rw2": true, ".pef": true,
		".srw": true, ".mp4": true, ".mov": true, ".avi": true,
	}
	return supportedExts[ext]
}
