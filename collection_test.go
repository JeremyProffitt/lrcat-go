package lrcat

import (
	"testing"
	"time"
)

func TestAddCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	coll, err := catalog.AddCollection("Vacation 2024", CollectionTypeStandard, nil)
	if err != nil {
		t.Fatalf("Failed to add collection: %v", err)
	}

	if coll.ID == 0 {
		t.Error("Collection ID should not be 0")
	}
	if coll.Name != "Vacation 2024" {
		t.Errorf("Expected name 'Vacation 2024', got '%s'", coll.Name)
	}
	if coll.CreationID != CollectionTypeStandard {
		t.Errorf("Expected type standard, got '%s'", coll.CreationID)
	}
}

func TestAddSmartCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	coll, err := catalog.AddCollection("5 Stars", CollectionTypeSmart, nil)
	if err != nil {
		t.Fatalf("Failed to add smart collection: %v", err)
	}

	if coll.CreationID != CollectionTypeSmart {
		t.Errorf("Expected type smart, got '%s'", coll.CreationID)
	}
}

func TestAddCollectionGroup(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	group, err := catalog.AddCollection("Travel", CollectionTypeGroup, nil)
	if err != nil {
		t.Fatalf("Failed to add collection group: %v", err)
	}

	// Add collection inside group
	coll, err := catalog.AddCollection("Italy 2024", CollectionTypeStandard, &group.ID)
	if err != nil {
		t.Fatalf("Failed to add collection in group: %v", err)
	}

	if coll.ParentID == nil {
		t.Error("Collection should have parent")
	}
	if *coll.ParentID != group.ID {
		t.Errorf("Expected parent ID %d, got %d", group.ID, *coll.ParentID)
	}
}

func TestGetCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, _ := catalog.AddCollection("Test Collection", CollectionTypeStandard, nil)

	retrieved, err := catalog.GetCollection(created.ID)
	if err != nil {
		t.Fatalf("Failed to get collection: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Name != created.Name {
		t.Errorf("Name mismatch: expected %s, got %s", created.Name, retrieved.Name)
	}
}

func TestGetCollectionByName(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	created, _ := catalog.AddCollection("My Photos", CollectionTypeStandard, nil)

	retrieved, err := catalog.GetCollectionByName("My Photos")
	if err != nil {
		t.Fatalf("Failed to get collection by name: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find collection")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: expected %d, got %d", created.ID, retrieved.ID)
	}
}

func TestListCollections(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	catalog.AddCollection("Collection A", CollectionTypeStandard, nil)
	catalog.AddCollection("Collection B", CollectionTypeStandard, nil)
	catalog.AddCollection("Collection C", CollectionTypeSmart, nil)

	collections, err := catalog.ListCollections()
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}
}

func TestAddImageToCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	// Add image
	input := &ImageInput{
		FilePath:    "/photos/collection_test.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)

	// Add collection
	coll, _ := catalog.AddCollection("Test", CollectionTypeStandard, nil)

	// Add image to collection
	err := catalog.AddImageToCollection(image.ID, coll.ID)
	if err != nil {
		t.Fatalf("Failed to add image to collection: %v", err)
	}

	// Verify
	images, _ := catalog.GetCollectionImages(coll.ID)
	if len(images) != 1 {
		t.Errorf("Expected 1 image, got %d", len(images))
	}
}

func TestRemoveImageFromCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/remove_test.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)
	coll, _ := catalog.AddCollection("RemoveTest", CollectionTypeStandard, nil)

	catalog.AddImageToCollection(image.ID, coll.ID)
	err := catalog.RemoveImageFromCollection(image.ID, coll.ID)
	if err != nil {
		t.Fatalf("Failed to remove image from collection: %v", err)
	}

	images, _ := catalog.GetCollectionImages(coll.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images, got %d", len(images))
	}
}

func TestGetImageCollections(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/multi_collection.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)

	// Add to multiple collections
	coll1, _ := catalog.AddCollection("Collection 1", CollectionTypeStandard, nil)
	coll2, _ := catalog.AddCollection("Collection 2", CollectionTypeStandard, nil)
	coll3, _ := catalog.AddCollection("Collection 3", CollectionTypeStandard, nil)

	catalog.AddImageToCollection(image.ID, coll1.ID)
	catalog.AddImageToCollection(image.ID, coll2.ID)
	catalog.AddImageToCollection(image.ID, coll3.ID)

	collections, err := catalog.GetImageCollections(image.ID)
	if err != nil {
		t.Fatalf("Failed to get image collections: %v", err)
	}

	if len(collections) != 3 {
		t.Errorf("Expected 3 collections, got %d", len(collections))
	}
}

func TestDeleteCollection(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	coll, _ := catalog.AddCollection("To Delete", CollectionTypeStandard, nil)

	// Add an image
	input := &ImageInput{
		FilePath:    "/photos/delete_test.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)
	catalog.AddImageToCollection(image.ID, coll.ID)

	// Delete collection
	err := catalog.DeleteCollection(coll.ID)
	if err != nil {
		t.Fatalf("Failed to delete collection: %v", err)
	}

	// Verify deleted
	_, err = catalog.GetCollection(coll.ID)
	if err == nil {
		t.Error("Expected error when getting deleted collection")
	}

	// Image should still exist
	_, err = catalog.GetImage(image.ID)
	if err != nil {
		t.Error("Image should still exist after collection deletion")
	}
}

func TestCollectionImageCount(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	coll, _ := catalog.AddCollection("Count Test", CollectionTypeStandard, nil)

	for i := 0; i < 5; i++ {
		input := &ImageInput{
			FilePath:    "/photos/count" + string(rune('0'+i)) + ".jpg",
			CaptureTime: time.Now(),
		}
		image, _ := catalog.AddImage(input)
		catalog.AddImageToCollection(image.ID, coll.ID)
	}

	// Get fresh collection data
	updated, _ := catalog.GetCollection(coll.ID)
	if updated.ImageCount == nil || *updated.ImageCount != 5 {
		count := 0
		if updated.ImageCount != nil {
			count = *updated.ImageCount
		}
		t.Errorf("Expected image count 5, got %d", count)
	}
}

func TestCollectionImageOrder(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	coll, _ := catalog.AddCollection("Order Test", CollectionTypeStandard, nil)

	var imageIDs []int64
	for i := 0; i < 3; i++ {
		input := &ImageInput{
			FilePath:    "/photos/order" + string(rune('0'+i)) + ".jpg",
			CaptureTime: time.Now(),
		}
		image, _ := catalog.AddImage(input)
		catalog.AddImageToCollection(image.ID, coll.ID)
		imageIDs = append(imageIDs, image.ID)
	}

	images, _ := catalog.GetCollectionImages(coll.ID)

	// Images should be in order added
	for i, img := range images {
		if img.ID != imageIDs[i] {
			t.Errorf("Expected image ID %d at position %d, got %d", imageIDs[i], i, img.ID)
		}
	}
}
