package lrcat

import (
	"testing"
	"time"
)

func TestAddKeyword(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	kw, err := catalog.AddKeyword("vacation", nil)
	if err != nil {
		t.Fatalf("Failed to add keyword: %v", err)
	}

	if kw.ID == 0 {
		t.Error("Keyword ID should not be 0")
	}
	if kw.Name != "vacation" {
		t.Errorf("Expected name 'vacation', got '%s'", kw.Name)
	}
	if kw.LCName != "vacation" {
		t.Errorf("Expected lowercase name 'vacation', got '%s'", kw.LCName)
	}
	if kw.ParentID != nil {
		t.Error("Parent ID should be nil for root keyword")
	}
}

func TestAddKeywordWithParent(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	parent, _ := catalog.AddKeyword("People", nil)
	child, err := catalog.AddKeyword("Family", &parent.ID)
	if err != nil {
		t.Fatalf("Failed to add child keyword: %v", err)
	}

	if child.ParentID == nil {
		t.Error("Child should have parent ID")
	}
	if *child.ParentID != parent.ID {
		t.Errorf("Expected parent ID %d, got %d", parent.ID, *child.ParentID)
	}
}

func TestGetKeyword(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, _ := catalog.AddKeyword("travel", nil)

	retrieved, err := catalog.GetKeyword(created.ID)
	if err != nil {
		t.Fatalf("Failed to get keyword: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Name != created.Name {
		t.Errorf("Name mismatch: expected %s, got %s", created.Name, retrieved.Name)
	}
}

func TestGetKeywordByName(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, _ := catalog.AddKeyword("Beach", nil)

	// Test case-insensitive search
	retrieved, err := catalog.GetKeywordByName("beach")
	if err != nil {
		t.Fatalf("Failed to get keyword by name: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find keyword")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
}

func TestGetOrCreateKeyword(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	// Create new
	kw1, err := catalog.GetOrCreateKeyword("nature", nil)
	if err != nil {
		t.Fatalf("Failed to get or create keyword: %v", err)
	}

	// Get existing
	kw2, err := catalog.GetOrCreateKeyword("Nature", nil)
	if err != nil {
		t.Fatalf("Failed to get existing keyword: %v", err)
	}

	if kw1.ID != kw2.ID {
		t.Error("Should return same keyword")
	}
}

func TestListKeywords(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	catalog.AddKeyword("alpha", nil)
	catalog.AddKeyword("beta", nil)
	catalog.AddKeyword("gamma", nil)

	keywords, err := catalog.ListKeywords()
	if err != nil {
		t.Fatalf("Failed to list keywords: %v", err)
	}

	if len(keywords) != 3 {
		t.Errorf("Expected 3 keywords, got %d", len(keywords))
	}
}

func TestAddKeywordToImage(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	// Add image
	input := &ImageInput{
		FilePath:    "/photos/tagged.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)

	// Add keyword
	kw, _ := catalog.AddKeyword("sunset", nil)

	// Associate
	err := catalog.AddKeywordToImage(image.ID, kw.ID)
	if err != nil {
		t.Fatalf("Failed to add keyword to image: %v", err)
	}

	// Verify
	keywords, _ := catalog.GetImageKeywords(image.ID)
	if len(keywords) != 1 {
		t.Errorf("Expected 1 keyword, got %d", len(keywords))
	}
	if keywords[0].Name != "sunset" {
		t.Errorf("Expected keyword 'sunset', got '%s'", keywords[0].Name)
	}
}

func TestRemoveKeywordFromImage(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/test.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)
	kw, _ := catalog.AddKeyword("temporary", nil)

	catalog.AddKeywordToImage(image.ID, kw.ID)
	err := catalog.RemoveKeywordFromImage(image.ID, kw.ID)
	if err != nil {
		t.Fatalf("Failed to remove keyword: %v", err)
	}

	keywords, _ := catalog.GetImageKeywords(image.ID)
	if len(keywords) != 0 {
		t.Errorf("Expected 0 keywords, got %d", len(keywords))
	}
}

func TestGetKeywordImages(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	kw, _ := catalog.AddKeyword("flowers", nil)

	for i := 0; i < 3; i++ {
		input := &ImageInput{
			FilePath:    "/photos/flower" + string(rune('0'+i)) + ".jpg",
			CaptureTime: time.Now(),
		}
		image, _ := catalog.AddImage(input)
		catalog.AddKeywordToImage(image.ID, kw.ID)
	}

	images, err := catalog.GetKeywordImages(kw.ID)
	if err != nil {
		t.Fatalf("Failed to get keyword images: %v", err)
	}

	if len(images) != 3 {
		t.Errorf("Expected 3 images, got %d", len(images))
	}
}

func TestCreateHierarchicalKeywords(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	kw, err := catalog.CreateHierarchicalKeywords("Animals/Dogs/Labrador")
	if err != nil {
		t.Fatalf("Failed to create hierarchical keywords: %v", err)
	}

	if kw.Name != "Labrador" {
		t.Errorf("Expected final keyword 'Labrador', got '%s'", kw.Name)
	}

	// Should have created 3 keywords
	keywords, _ := catalog.ListKeywords()
	if len(keywords) != 3 {
		t.Errorf("Expected 3 keywords, got %d", len(keywords))
	}

	// Verify hierarchy
	labrador, _ := catalog.GetKeywordByName("Labrador")
	if labrador.ParentID == nil {
		t.Error("Labrador should have a parent")
	}

	dogs, _ := catalog.GetKeyword(*labrador.ParentID)
	if dogs.Name != "Dogs" {
		t.Errorf("Expected parent 'Dogs', got '%s'", dogs.Name)
	}
}

func TestMultipleKeywordsOnImage(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/multikey.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)

	keywords := []string{"travel", "summer", "beach", "family"}
	for _, name := range keywords {
		kw, _ := catalog.AddKeyword(name, nil)
		catalog.AddKeywordToImage(image.ID, kw.ID)
	}

	imageKeywords, _ := catalog.GetImageKeywords(image.ID)
	if len(imageKeywords) != 4 {
		t.Errorf("Expected 4 keywords, got %d", len(imageKeywords))
	}
}
