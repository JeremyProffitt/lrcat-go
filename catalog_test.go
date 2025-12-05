package lrcat

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer catalog.Close()

	// Verify file exists
	if _, err := os.Stat(catalogPath); os.IsNotExist(err) {
		t.Error("Catalog file was not created")
	}

	// Verify DB version
	version, err := catalog.GetDBVersion()
	if err != nil {
		t.Fatalf("Failed to get DB version: %v", err)
	}
	if version != schemaVersion {
		t.Errorf("Expected version %s, got %s", schemaVersion, version)
	}
}

func TestOpenCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	// Create catalog
	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	catalog.Close()

	// Open catalog
	catalog, err = OpenCatalog(catalogPath, nil)
	if err != nil {
		t.Fatalf("Failed to open catalog: %v", err)
	}
	defer catalog.Close()

	version, err := catalog.GetDBVersion()
	if err != nil {
		t.Fatalf("Failed to get DB version: %v", err)
	}
	if version != schemaVersion {
		t.Errorf("Expected version %s, got %s", schemaVersion, version)
	}
}

func TestOpenCatalogReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	// Create catalog
	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	catalog.Close()

	// Open catalog in read-only mode
	catalog, err = OpenCatalog(catalogPath, &CatalogOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("Failed to open catalog read-only: %v", err)
	}
	defer catalog.Close()

	if catalog.Path() != catalogPath {
		t.Errorf("Expected path %s, got %s", catalogPath, catalog.Path())
	}
}

func TestOpenNonExistentCatalog(t *testing.T) {
	_, err := OpenCatalog("/nonexistent/path/catalog.lrcat", nil)
	if err == nil {
		t.Error("Expected error when opening non-existent catalog")
	}
}

func TestToLightroomTimestamp(t *testing.T) {
	// Test the Lightroom epoch itself
	ts := ToLightroomTimestamp(LightroomEpoch)
	if ts != 0 {
		t.Errorf("Expected 0 for epoch, got %f", ts)
	}

	// Test a known date
	testTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)
	ts = ToLightroomTimestamp(testTime)
	if ts <= 0 {
		t.Errorf("Expected positive timestamp, got %f", ts)
	}
}

func TestFromLightroomTimestamp(t *testing.T) {
	// Round-trip test
	original := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	ts := ToLightroomTimestamp(original)
	recovered := FromLightroomTimestamp(ts)

	if !original.Equal(recovered) {
		t.Errorf("Round-trip failed: original %v, recovered %v", original, recovered)
	}
}

func TestFormatCaptureTime(t *testing.T) {
	testTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	formatted := FormatCaptureTime(testTime)
	expected := "2024-06-15T14:30:45"
	if formatted != expected {
		t.Errorf("Expected %s, got %s", expected, formatted)
	}
}

func TestNewUUID(t *testing.T) {
	uuid1 := NewUUID()
	uuid2 := NewUUID()

	if uuid1 == uuid2 {
		t.Error("UUIDs should be unique")
	}

	if len(uuid1) != 36 {
		t.Errorf("Expected UUID length 36, got %d", len(uuid1))
	}
}

func TestImageCount(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer catalog.Close()

	count, err := catalog.ImageCount()
	if err != nil {
		t.Fatalf("Failed to get image count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 images, got %d", count)
	}
}

func TestFolderCount(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	defer catalog.Close()

	count, err := catalog.FolderCount()
	if err != nil {
		t.Fatalf("Failed to get folder count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 folders, got %d", count)
	}
}

// Helper function to create a test catalog
func createTestCatalog(t *testing.T) *Catalog {
	t.Helper()
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "test.lrcat")

	catalog, err := NewCatalog(catalogPath)
	if err != nil {
		t.Fatalf("Failed to create catalog: %v", err)
	}
	return catalog
}
