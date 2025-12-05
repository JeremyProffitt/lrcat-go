package lrcat

import (
	"strings"
	"testing"
	"time"
)

func TestCompressDecompressXMP(t *testing.T) {
	originalXMP := `<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
      xmlns:xmp="http://ns.adobe.com/xap/1.0/"
      xmp:Rating="5">
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>`

	compressed, err := CompressXMP(originalXMP)
	if err != nil {
		t.Fatalf("Failed to compress XMP: %v", err)
	}

	if len(compressed) >= len(originalXMP) {
		t.Log("Warning: Compressed data is not smaller than original (expected for small data)")
	}

	// First 4 bytes should be the uncompressed length
	if len(compressed) < 4 {
		t.Fatal("Compressed data too short")
	}

	decompressed, err := DecompressXMP(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress XMP: %v", err)
	}

	if decompressed != originalXMP {
		t.Errorf("Decompressed XMP doesn't match original.\nExpected: %s\nGot: %s", originalXMP, decompressed)
	}
}

func TestCompressEmptyXMP(t *testing.T) {
	compressed, err := CompressXMP("")
	if err != nil {
		t.Fatalf("Failed to compress empty XMP: %v", err)
	}
	if compressed != nil {
		t.Error("Expected nil for empty XMP")
	}
}

func TestDecompressEmptyData(t *testing.T) {
	result, err := DecompressXMP([]byte{})
	if err != nil {
		t.Fatalf("Failed to decompress empty data: %v", err)
	}
	if result != "" {
		t.Error("Expected empty string for empty data")
	}
}

func TestDecompressShortData(t *testing.T) {
	result, err := DecompressXMP([]byte{0x00, 0x01, 0x02})
	if err != nil {
		t.Fatalf("Failed to decompress short data: %v", err)
	}
	if result != "" {
		t.Error("Expected empty string for short data")
	}
}

func TestSetAndGetXMP(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	// Add an image
	input := &ImageInput{
		FilePath:    "/photos/test.jpg",
		CaptureTime: time.Now(),
	}
	image, err := catalog.AddImage(input)
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Set XMP
	xmpContent := `<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about="" xmp:Rating="5"/>
  </rdf:RDF>
</x:xmpmeta>`

	err = catalog.SetXMP(image.ID, xmpContent)
	if err != nil {
		t.Fatalf("Failed to set XMP: %v", err)
	}

	// Get XMP
	retrieved, err := catalog.GetXMP(image.ID)
	if err != nil {
		t.Fatalf("Failed to get XMP: %v", err)
	}

	if retrieved != xmpContent {
		t.Errorf("XMP mismatch.\nExpected: %s\nGot: %s", xmpContent, retrieved)
	}
}

func TestGenerateBasicXMP(t *testing.T) {
	rating := 4
	xmp := GenerateBasicXMP(&rating, "Red", "2024-06-15T14:30:00")

	if !strings.Contains(xmp, `xmp:Rating="4"`) {
		t.Error("XMP should contain rating")
	}
	if !strings.Contains(xmp, `xmp:Label="Red"`) {
		t.Error("XMP should contain label")
	}
	if !strings.Contains(xmp, `exif:DateTimeOriginal="2024-06-15T14:30:00"`) {
		t.Error("XMP should contain date")
	}
	if !strings.Contains(xmp, "x:xmpmeta") {
		t.Error("XMP should contain xmpmeta root")
	}
}

func TestGenerateBasicXMPNoRating(t *testing.T) {
	xmp := GenerateBasicXMP(nil, "", "")

	if strings.Contains(xmp, "xmp:Rating") {
		t.Error("XMP should not contain rating when nil")
	}
	if strings.Contains(xmp, "xmp:Label") {
		t.Error("XMP should not contain label when empty")
	}
}

func TestExtractXMPValue(t *testing.T) {
	xmp := `<rdf:Description exif:DateTimeOriginal="2024-06-15T14:30:00" xmp:Rating="5"/>`

	date := ExtractXMPValue(xmp, "exif:DateTimeOriginal")
	if date != "2024-06-15T14:30:00" {
		t.Errorf("Expected date '2024-06-15T14:30:00', got '%s'", date)
	}

	rating := ExtractXMPValue(xmp, "xmp:Rating")
	if rating != "5" {
		t.Errorf("Expected rating '5', got '%s'", rating)
	}

	missing := ExtractXMPValue(xmp, "xmp:Label")
	if missing != "" {
		t.Errorf("Expected empty string for missing key, got '%s'", missing)
	}
}

func TestXMPRoundTrip(t *testing.T) {
	catalog := createTestCatalog(t)
	defer catalog.Close()

	input := &ImageInput{
		FilePath:    "/photos/roundtrip.jpg",
		CaptureTime: time.Now(),
	}
	image, _ := catalog.AddImage(input)

	// Large XMP content to test compression
	rating := 5
	xmpContent := GenerateBasicXMP(&rating, "Green", "2024-01-15T10:00:00")

	err := catalog.SetXMP(image.ID, xmpContent)
	if err != nil {
		t.Fatalf("Failed to set XMP: %v", err)
	}

	retrieved, err := catalog.GetXMP(image.ID)
	if err != nil {
		t.Fatalf("Failed to get XMP: %v", err)
	}

	if retrieved != xmpContent {
		t.Error("XMP round-trip failed")
	}
}
