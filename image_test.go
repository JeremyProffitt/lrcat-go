package lrcat

import (
	"testing"
	"time"
)

func TestAddImage(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rating := 4
	width := 4000
	height := 3000

	input := &ImageInput{
		FilePath:    "/photos/2024/vacation/IMG_001.jpg",
		CaptureTime: time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
		Rating:      &rating,
		ColorLabel:  "Red",
		Pick:        1,
		Width:       &width,
		Height:      &height,
	}

	image, err := catalog.AddImage(input)
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	if image.ID == 0 {
		t.Error("Image ID should not be 0")
	}
	if image.UUID == "" {
		t.Error("Image UUID should not be empty")
	}
	if *image.Rating != rating {
		t.Errorf("Expected rating %d, got %d", rating, *image.Rating)
	}
	if image.ColorLabel != "Red" {
		t.Errorf("Expected color label 'Red', got '%s'", image.ColorLabel)
	}
	if image.Pick != 1 {
		t.Errorf("Expected pick 1, got %d", image.Pick)
	}
	if image.FileFormat != "JPG" {
		t.Errorf("Expected format 'JPG', got '%s'", image.FileFormat)
	}

	// Verify counts
	count, _ := catalog.ImageCount()
	if count != 1 {
		t.Errorf("Expected 1 image, got %d", count)
	}

	folderCount, _ := catalog.FolderCount()
	if folderCount != 1 {
		t.Errorf("Expected 1 folder, got %d", folderCount)
	}

	rootCount, _ := catalog.RootFolderCount()
	if rootCount != 1 {
		t.Errorf("Expected 1 root folder, got %d", rootCount)
	}
}

func TestAddImages(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	inputs := []*ImageInput{
		{
			FilePath:    "/photos/2024/IMG_001.jpg",
			CaptureTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			FilePath:    "/photos/2024/IMG_002.jpg",
			CaptureTime: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			FilePath:    "/photos/2024/IMG_003.png",
			CaptureTime: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC),
		},
	}

	importSession, images, err := catalog.AddImages(inputs)
	if err != nil {
		t.Fatalf("Failed to add images: %v", err)
	}

	if importSession == nil {
		t.Fatal("Import session should not be nil")
	}
	if importSession.ImageCount != 3 {
		t.Errorf("Expected import count 3, got %d", importSession.ImageCount)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}

	count, _ := catalog.ImageCount()
	if count != 3 {
		t.Errorf("Expected 3 images in catalog, got %d", count)
	}
}

func TestGetImage(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rating := 5
	input := &ImageInput{
		FilePath:    "/photos/test.jpg",
		CaptureTime: time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		Rating:      &rating,
	}

	created, _ := catalog.AddImage(input)

	retrieved, err := catalog.GetImage(created.ID)
	if err != nil {
		t.Fatalf("Failed to get image: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
	if *retrieved.Rating != rating {
		t.Errorf("Rating mismatch: expected %d, got %d", rating, *retrieved.Rating)
	}
}

func TestListImages(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	for i := 0; i < 5; i++ {
		input := &ImageInput{
			FilePath:    "/photos/IMG_00" + string(rune('0'+i)) + ".jpg",
			CaptureTime: time.Now(),
		}
		catalog.AddImage(input)
	}

	images, err := catalog.ListImages()
	if err != nil {
		t.Fatalf("Failed to list images: %v", err)
	}

	if len(images) != 5 {
		t.Errorf("Expected 5 images, got %d", len(images))
	}
}

func TestDetectFileFormat(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{"jpg", "JPG"},
		{"JPEG", "JPG"},
		{"png", "PNG"},
		{"tiff", "TIFF"},
		{"tif", "TIFF"},
		{"dng", "DNG"},
		{"cr2", "RAW"},
		{"nef", "RAW"},
		{"arw", "RAW"},
		{"mp4", "VIDEO"},
		{"mov", "VIDEO"},
		{"psd", "PSD"},
		{"unknown", "JPG"}, // default
	}

	for _, tc := range tests {
		result := detectFileFormat(tc.ext)
		if result != tc.expected {
			t.Errorf("detectFileFormat(%s): expected %s, got %s", tc.ext, tc.expected, result)
		}
	}
}

func TestImageExists(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/existing.jpg",
		CaptureTime: time.Now(),
	}
	catalog.AddImage(input)

	exists, err := catalog.ImageExists("/photos/existing.jpg")
	if err != nil {
		t.Fatalf("Failed to check image exists: %v", err)
	}
	if !exists {
		t.Error("Image should exist")
	}

	exists, _ = catalog.ImageExists("/photos/nonexistent.jpg")
	if exists {
		t.Error("Image should not exist")
	}
}

func TestIsImageExtension(t *testing.T) {
	validExts := []string{".jpg", ".jpeg", ".png", ".dng", ".cr2", ".nef", ".mp4"}
	for _, ext := range validExts {
		if !isImageExtension(ext) {
			t.Errorf("Expected %s to be valid image extension", ext)
		}
	}

	invalidExts := []string{".txt", ".pdf", ".doc", ".exe"}
	for _, ext := range invalidExts {
		if isImageExtension(ext) {
			t.Errorf("Expected %s to be invalid image extension", ext)
		}
	}
}

func TestAddImageWithDifferentFormats(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	formats := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "JPG"},
		{"photo.png", "PNG"},
		{"photo.dng", "DNG"},
		{"photo.cr2", "RAW"},
		{"video.mp4", "VIDEO"},
	}

	for _, f := range formats {
		input := &ImageInput{
			FilePath:    "/photos/" + f.filename,
			CaptureTime: time.Now(),
		}
		image, err := catalog.AddImage(input)
		if err != nil {
			t.Fatalf("Failed to add %s: %v", f.filename, err)
		}
		if image.FileFormat != f.expected {
			t.Errorf("For %s: expected format %s, got %s", f.filename, f.expected, image.FileFormat)
		}
	}
}

func TestAddImageCreatesFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/2024/vacation/beach/IMG_001.jpg",
		CaptureTime: time.Now(),
	}

	_, err := catalog.AddImage(input)
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Check that root folder was created
	rootFolders, _ := catalog.ListRootFolders()
	if len(rootFolders) != 1 {
		t.Errorf("Expected 1 root folder, got %d", len(rootFolders))
	}
}
