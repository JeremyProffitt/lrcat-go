package lrcat

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// XMPMetadata represents XMP metadata for an image
type XMPMetadata struct {
	ImageID int64
	XMP     string
}

// CompressXMP compresses XMP data in Lightroom's format:
// 4-byte big-endian length of uncompressed data followed by zlib-compressed data
func CompressXMP(xmp string) ([]byte, error) {
	if xmp == "" {
		return nil, nil
	}

	// Create buffer for compressed data
	var buf bytes.Buffer

	// Write uncompressed length as 4-byte big-endian
	uncompressedLen := uint32(len(xmp))
	if err := binary.Write(&buf, binary.BigEndian, uncompressedLen); err != nil {
		return nil, fmt.Errorf("failed to write length header: %w", err)
	}

	// Create zlib writer and compress the data
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte(xmp)); err != nil {
		zw.Close()
		return nil, fmt.Errorf("failed to compress XMP: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zlib writer: %w", err)
	}

	return buf.Bytes(), nil
}

// DecompressXMP decompresses XMP data from Lightroom's format
func DecompressXMP(data []byte) (string, error) {
	if len(data) < 4 {
		return "", nil
	}

	// Read uncompressed length from first 4 bytes (big-endian)
	// This is informational; we don't strictly need it for decompression
	_ = binary.BigEndian.Uint32(data[:4])

	// Decompress the rest using zlib
	reader := bytes.NewReader(data[4:])
	zr, err := zlib.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer zr.Close()

	decompressed, err := io.ReadAll(zr)
	if err != nil {
		return "", fmt.Errorf("failed to decompress XMP: %w", err)
	}

	return string(decompressed), nil
}

// SetXMP sets the XMP metadata for an image
func (c *Catalog) SetXMP(imageID int64, xmp string) error {
	compressed, err := CompressXMP(xmp)
	if err != nil {
		return fmt.Errorf("failed to compress XMP: %w", err)
	}

	_, err = c.db.Exec(
		`UPDATE Adobe_AdditionalMetadata SET xmp = ? WHERE image = ?`,
		compressed, imageID,
	)
	if err != nil {
		return fmt.Errorf("failed to update XMP: %w", err)
	}

	return nil
}

// GetXMP retrieves the XMP metadata for an image
func (c *Catalog) GetXMP(imageID int64) (string, error) {
	var data []byte
	err := c.db.QueryRow(
		`SELECT xmp FROM Adobe_AdditionalMetadata WHERE image = ?`,
		imageID,
	).Scan(&data)
	if err != nil {
		return "", fmt.Errorf("failed to get XMP: %w", err)
	}

	if len(data) == 0 {
		return "", nil
	}

	return DecompressXMP(data)
}

// GenerateBasicXMP generates a basic XMP sidecar content for an image
func GenerateBasicXMP(rating *int, colorLabel string, captureTime string) string {
	ratingStr := ""
	if rating != nil {
		ratingStr = fmt.Sprintf(`   xmp:Rating="%d"`, *rating)
	}

	labelStr := ""
	if colorLabel != "" {
		labelStr = fmt.Sprintf(`   xmp:Label="%s"`, colorLabel)
	}

	dateStr := ""
	if captureTime != "" {
		dateStr = fmt.Sprintf(`   exif:DateTimeOriginal="%s"`, captureTime)
	}

	return fmt.Sprintf(`<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 7.0-c000 1.000000, 0000/00/00-00:00:00">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
   xmlns:xmp="http://ns.adobe.com/xap/1.0/"
   xmlns:exif="http://ns.adobe.com/exif/1.0/"
   xmlns:crs="http://ns.adobe.com/camera-raw-settings/1.0/"
%s%s%s
   crs:Version="15.0"
   crs:ProcessVersion="11.0">
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>`, ratingStr, labelStr, dateStr)
}

// ExtractXMPValue extracts a value from XMP content by key (e.g., "exif:DateTimeOriginal")
func ExtractXMPValue(xmp string, key string) string {
	// Look for key="value" pattern
	searchStr := key + `="`
	startIdx := bytes.Index([]byte(xmp), []byte(searchStr))
	if startIdx == -1 {
		return ""
	}

	startIdx += len(searchStr)
	endIdx := bytes.Index([]byte(xmp[startIdx:]), []byte(`"`))
	if endIdx == -1 {
		return ""
	}

	return xmp[startIdx : startIdx+endIdx]
}
