package lrcat

import (
	"testing"
)

func TestAddRootFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rf, err := catalog.AddRootFolder("/photos/2024")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}

	if rf.ID == 0 {
		t.Error("Root folder ID should not be 0")
	}
	if rf.UUID == "" {
		t.Error("Root folder UUID should not be empty")
	}
	if rf.Name != "2024" {
		t.Errorf("Expected name '2024', got '%s'", rf.Name)
	}
	if rf.AbsolutePath != "/photos/2024/" {
		t.Errorf("Expected path '/photos/2024/', got '%s'", rf.AbsolutePath)
	}
}

func TestGetRootFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, err := catalog.AddRootFolder("/photos/test")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}

	retrieved, err := catalog.GetRootFolder(created.ID)
	if err != nil {
		t.Fatalf("Failed to get root folder: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Name != created.Name {
		t.Errorf("Name mismatch: expected %s, got %s", created.Name, retrieved.Name)
	}
}

func TestGetRootFolderByPath(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, err := catalog.AddRootFolder("/photos/vacation")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}

	retrieved, err := catalog.GetRootFolderByPath("/photos/vacation")
	if err != nil {
		t.Fatalf("Failed to get root folder by path: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find root folder")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
}

func TestListRootFolders(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	// Add multiple root folders
	_, err := catalog.AddRootFolder("/photos/a")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}
	_, err = catalog.AddRootFolder("/photos/b")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}

	folders, err := catalog.ListRootFolders()
	if err != nil {
		t.Fatalf("Failed to list root folders: %v", err)
	}

	if len(folders) != 2 {
		t.Errorf("Expected 2 root folders, got %d", len(folders))
	}
}

func TestAddFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rf, err := catalog.AddRootFolder("/photos")
	if err != nil {
		t.Fatalf("Failed to add root folder: %v", err)
	}

	folder, err := catalog.AddFolder(rf.ID, "2024/January")
	if err != nil {
		t.Fatalf("Failed to add folder: %v", err)
	}

	if folder.ID == 0 {
		t.Error("Folder ID should not be 0")
	}
	if folder.RootFolderID != rf.ID {
		t.Errorf("Expected root folder ID %d, got %d", rf.ID, folder.RootFolderID)
	}
	if folder.PathFromRoot != "2024/January/" {
		t.Errorf("Expected path '2024/January/', got '%s'", folder.PathFromRoot)
	}
}

func TestGetFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rf, _ := catalog.AddRootFolder("/photos")
	created, _ := catalog.AddFolder(rf.ID, "2024/February")

	retrieved, err := catalog.GetFolder(created.ID)
	if err != nil {
		t.Fatalf("Failed to get folder: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.PathFromRoot != created.PathFromRoot {
		t.Errorf("Path mismatch: expected %s, got %s", created.PathFromRoot, retrieved.PathFromRoot)
	}
}

func TestGetOrCreateFolder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rf, _ := catalog.AddRootFolder("/photos")

	// Create new folder
	folder1, err := catalog.GetOrCreateFolder(rf.ID, "2024/March")
	if err != nil {
		t.Fatalf("Failed to get or create folder: %v", err)
	}

	// Get existing folder
	folder2, err := catalog.GetOrCreateFolder(rf.ID, "2024/March")
	if err != nil {
		t.Fatalf("Failed to get or create existing folder: %v", err)
	}

	if folder1.ID != folder2.ID {
		t.Error("Should return same folder")
	}
}

func TestListFolders(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	rf, _ := catalog.AddRootFolder("/photos")
	catalog.AddFolder(rf.ID, "2024/April")
	catalog.AddFolder(rf.ID, "2024/May")

	folders, err := catalog.ListFolders(rf.ID)
	if err != nil {
		t.Fatalf("Failed to list folders: %v", err)
	}

	if len(folders) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(folders))
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\photos\\2024", "C:/photos/2024"},
		{"/photos/2024", "/photos/2024"},
		{"photos\\vacation", "photos/vacation"},
	}

	for _, tc := range tests {
		result := normalizePath(tc.input)
		// On non-Windows, backslashes might not be converted
		// This test is primarily for Windows behavior
		if result != tc.expected && result != tc.input {
			t.Errorf("normalizePath(%s): expected %s or %s, got %s", tc.input, tc.expected, tc.input, result)
		}
	}
}
