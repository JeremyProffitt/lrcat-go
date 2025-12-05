# lrcat-go

[![Test](https://github.com/JeremyProffitt/lrcat-go/actions/workflows/test.yml/badge.svg)](https://github.com/JeremyProffitt/lrcat-go/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/JeremyProffitt/lrcat-go.svg)](https://pkg.go.dev/github.com/JeremyProffitt/lrcat-go)

A Go library for creating and manipulating Adobe Lightroom Classic catalog files (`.lrcat`).

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [API Documentation](#api-documentation)
  - [Catalog Operations](#catalog-operations)
  - [Folder Management](#folder-management)
  - [Image Management](#image-management)
  - [Keywords](#keywords)
  - [Collections](#collections)
  - [XMP Metadata](#xmp-metadata)
- [Lightroom Catalog Format](#lightroom-catalog-format)
- [Examples](#examples)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)
- [References](#references)

---

## Overview

Adobe Lightroom Classic stores all photo metadata, develop settings, collections, and organizational data in a SQLite database with the `.lrcat` extension. This library provides a clean Go API to:

- **Create new catalogs** from scratch
- **Add images** with metadata (ratings, labels, flags)
- **Organize with folders** matching your filesystem structure
- **Tag with keywords** including hierarchical keyword trees
- **Group into collections** including smart collections and collection sets
- **Read and write XMP metadata** with proper Lightroom compression

This is useful for:
- Migrating photos from other applications to Lightroom
- Building tools that pre-populate Lightroom catalogs
- Automating catalog creation for batch workflows
- Reading metadata from existing Lightroom catalogs

---

## Installation

```bash
go get github.com/JeremyProffitt/lrcat-go
```

---

## Requirements

### Go Version
- Go 1.21 or later

### CGO and C Compiler
This library uses [go-sqlite3](https://github.com/mattn/go-sqlite3), which requires CGO and a C compiler:

| Platform | Installation |
|----------|--------------|
| **Ubuntu/Debian** | `sudo apt-get install build-essential` |
| **Fedora/RHEL** | `sudo dnf install gcc` |
| **macOS** | `xcode-select --install` |
| **Windows** | Install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or [MinGW-w64](https://www.mingw-w64.org/) |

Ensure CGO is enabled:
```bash
export CGO_ENABLED=1
```

---

## Quick Start

### Creating a New Catalog

```go
package main

import (
    "log"
    "time"

    lrcat "github.com/JeremyProffitt/lrcat-go"
)

func main() {
    // Create a new Lightroom catalog
    catalog, err := lrcat.NewCatalog("MyPhotos.lrcat")
    if err != nil {
        log.Fatal(err)
    }
    defer catalog.Close()

    // Add a photo
    rating := 4
    image, err := catalog.AddImage(&lrcat.ImageInput{
        FilePath:    "/Users/john/Photos/2024/vacation/IMG_001.jpg",
        CaptureTime: time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
        Rating:      &rating,
        ColorLabel:  "Green",
        Pick:        1,  // Flagged as pick
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Added image ID: %d", image.ID)
}
```

### Opening an Existing Catalog (Read-Only)

```go
// Open in read-only mode - safe while Lightroom is running
catalog, err := lrcat.OpenCatalog("MyPhotos.lrcat", &lrcat.CatalogOptions{
    ReadOnly: true,
})
if err != nil {
    log.Fatal(err)
}
defer catalog.Close()

// Query the catalog
count, _ := catalog.ImageCount()
fmt.Printf("Catalog contains %d images\n", count)

images, _ := catalog.ListImages()
for _, img := range images {
    fmt.Printf("- %s (Rating: %v)\n", img.UUID, img.Rating)
}
```

---

## Core Concepts

### Lightroom Catalog Structure

```
Catalog (.lrcat)
├── Root Folders (physical disk locations)
│   └── Folders (subdirectories)
│       └── Files (image references)
│           └── Images (metadata, ratings, etc.)
├── Keywords (hierarchical tags)
├── Collections (virtual groupings)
│   ├── Standard Collections
│   ├── Smart Collections
│   └── Collection Sets (folders)
└── XMP Metadata (develop settings, EXIF, IPTC)
```

### Key Types

| Type | Description |
|------|-------------|
| `Catalog` | Main database handle |
| `RootFolder` | Top-level folder (e.g., `/Users/john/Photos`) |
| `Folder` | Subfolder within a root folder |
| `Image` | Photo record with metadata |
| `ImageInput` | Input struct for adding images |
| `Keyword` | Tag that can be applied to images |
| `Collection` | Virtual grouping of images |

### Timestamps

Lightroom uses a custom epoch starting January 1, 2001:

```go
// Convert Go time to Lightroom timestamp
lrTimestamp := lrcat.ToLightroomTimestamp(time.Now())

// Convert back to Go time
goTime := lrcat.FromLightroomTimestamp(lrTimestamp)

// Format for captureTime field
formatted := lrcat.FormatCaptureTime(time.Now())  // "2024-06-15T14:30:00"
```

---

## API Documentation

### Catalog Operations

#### Creating and Opening Catalogs

```go
// Create a new catalog (overwrites if exists)
catalog, err := lrcat.NewCatalog("/path/to/catalog.lrcat")

// Open existing catalog
catalog, err := lrcat.OpenCatalog("/path/to/catalog.lrcat", nil)

// Open read-only (safe while Lightroom is open)
catalog, err := lrcat.OpenCatalog("/path/to/catalog.lrcat", &lrcat.CatalogOptions{
    ReadOnly: true,
})

// Always close when done
defer catalog.Close()
```

#### Catalog Information

```go
path := catalog.Path()              // File path
db := catalog.DB()                  // Underlying *sql.DB
version, _ := catalog.GetDBVersion() // Adobe DB version

imageCount, _ := catalog.ImageCount()
folderCount, _ := catalog.FolderCount()
rootCount, _ := catalog.RootFolderCount()
```

---

### Folder Management

Lightroom organizes images in a two-level folder hierarchy: **Root Folders** (absolute paths) and **Folders** (relative paths within roots).

#### Root Folders

```go
// Add a root folder
root, err := catalog.AddRootFolder("/Users/john/Photos")
// root.ID = 1
// root.Name = "Photos"
// root.AbsolutePath = "/Users/john/Photos/"

// Get by ID or path
root, err := catalog.GetRootFolder(1)
root, err := catalog.GetRootFolderByPath("/Users/john/Photos")

// List all root folders
roots, err := catalog.ListRootFolders()
```

#### Subfolders

```go
// Add a folder within a root
folder, err := catalog.AddFolder(root.ID, "2024/Vacation")
// folder.PathFromRoot = "2024/Vacation/"

// Get or create (idempotent)
folder, err := catalog.GetOrCreateFolder(root.ID, "2024/Vacation")

// List folders under a root
folders, err := catalog.ListFolders(root.ID)
```

---

### Image Management

#### Adding Images

```go
// Single image with full metadata
width, height := 4000, 3000
rating := 5
orientation := 1

image, err := catalog.AddImage(&lrcat.ImageInput{
    FilePath:    "/Users/john/Photos/2024/IMG_001.jpg",
    CaptureTime: time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
    Rating:      &rating,       // 0-5 stars (nil = no rating)
    ColorLabel:  "Red",         // Red, Yellow, Green, Blue, Purple
    Pick:        1,             // -1=rejected, 0=unflagged, 1=picked
    FileFormat:  "JPG",         // Auto-detected if empty
    Width:       &width,
    Height:      &height,
    Orientation: &orientation,  // EXIF orientation
})
```

#### Batch Import

```go
inputs := []*lrcat.ImageInput{
    {FilePath: "/photos/IMG_001.jpg", CaptureTime: time.Now()},
    {FilePath: "/photos/IMG_002.jpg", CaptureTime: time.Now()},
    {FilePath: "/photos/IMG_003.jpg", CaptureTime: time.Now()},
}

importSession, images, err := catalog.AddImages(inputs)
// importSession.ID = import identifier
// importSession.ImageCount = 3
// images = slice of created Image records
```

#### Querying Images

```go
// Get by ID
image, err := catalog.GetImage(123)

// List all images
images, err := catalog.ListImages()

// Check if image exists
exists, err := catalog.ImageExists("/photos/IMG_001.jpg")
```

#### Directory Scanning

```go
// Scan directory for image files
inputs, err := lrcat.ScanDirectory("/photos/2024", true)  // recursive=true

// Import all found images
_, images, err := catalog.AddImages(inputs)
```

#### Supported File Formats

| Extension | Format |
|-----------|--------|
| `.jpg`, `.jpeg` | JPG |
| `.png` | PNG |
| `.tiff`, `.tif` | TIFF |
| `.psd` | PSD |
| `.dng` | DNG |
| `.cr2`, `.cr3`, `.nef`, `.arw`, `.orf`, `.raf`, `.rw2`, `.pef`, `.srw` | RAW |
| `.mp4`, `.mov`, `.avi` | VIDEO |

---

### Keywords

Keywords are tags that can be organized hierarchically.

#### Creating Keywords

```go
// Simple keyword
keyword, err := catalog.AddKeyword("vacation", nil)

// Child keyword
parentID := keyword.ID
child, err := catalog.AddKeyword("beach", &parentID)

// Create entire hierarchy at once
// Creates: Animals -> Dogs -> Labrador
labrador, err := catalog.CreateHierarchicalKeywords("Animals/Dogs/Labrador")
```

#### Querying Keywords

```go
// Get by ID
kw, err := catalog.GetKeyword(123)

// Get by name (case-insensitive)
kw, err := catalog.GetKeywordByName("vacation")

// Get or create (idempotent)
kw, err := catalog.GetOrCreateKeyword("travel", nil)

// List all keywords
keywords, err := catalog.ListKeywords()
```

#### Applying Keywords to Images

```go
// Add keyword to image
err := catalog.AddKeywordToImage(imageID, keywordID)

// Remove keyword from image
err := catalog.RemoveKeywordFromImage(imageID, keywordID)

// Get all keywords for an image
keywords, err := catalog.GetImageKeywords(imageID)

// Get all images with a keyword
images, err := catalog.GetKeywordImages(keywordID)
```

---

### Collections

Collections are virtual groupings of images (images can be in multiple collections).

#### Collection Types

```go
lrcat.CollectionTypeStandard  // Regular collection
lrcat.CollectionTypeSmart     // Smart collection (dynamic)
lrcat.CollectionTypeGroup     // Collection set (folder)
```

#### Creating Collections

```go
// Standard collection
coll, err := catalog.AddCollection("Best of 2024", lrcat.CollectionTypeStandard, nil)

// Collection set (folder)
set, err := catalog.AddCollection("Travel", lrcat.CollectionTypeGroup, nil)

// Collection inside a set
sub, err := catalog.AddCollection("Italy", lrcat.CollectionTypeStandard, &set.ID)

// Smart collection
smart, err := catalog.AddCollection("5 Stars", lrcat.CollectionTypeSmart, nil)
```

#### Managing Collection Contents

```go
// Add image to collection
err := catalog.AddImageToCollection(imageID, collectionID)

// Remove image from collection
err := catalog.RemoveImageFromCollection(imageID, collectionID)

// Get images in collection
images, err := catalog.GetCollectionImages(collectionID)

// Get collections containing an image
collections, err := catalog.GetImageCollections(imageID)

// Delete collection (doesn't delete images)
err := catalog.DeleteCollection(collectionID)
```

#### Querying Collections

```go
// Get by ID
coll, err := catalog.GetCollection(123)

// Get by name
coll, err := catalog.GetCollectionByName("Best of 2024")

// List all collections
collections, err := catalog.ListCollections()
```

---

### XMP Metadata

XMP (Extensible Metadata Platform) stores develop settings, EXIF, and IPTC data. Lightroom compresses XMP using zlib with a 4-byte length header.

#### Reading and Writing XMP

```go
// Set XMP for an image (automatically compressed)
xmpContent := `<x:xmpmeta xmlns:x="adobe:ns:meta/">...</x:xmpmeta>`
err := catalog.SetXMP(imageID, xmpContent)

// Get XMP for an image (automatically decompressed)
xmp, err := catalog.GetXMP(imageID)
```

#### Generating Basic XMP

```go
rating := 5
xmp := lrcat.GenerateBasicXMP(&rating, "Red", "2024-06-15T14:30:00")
```

#### Manual Compression/Decompression

```go
// Compress XMP string to Lightroom format
compressed, err := lrcat.CompressXMP(xmpString)

// Decompress Lightroom XMP blob
decompressed, err := lrcat.DecompressXMP(blobData)
```

#### Extracting Values from XMP

```go
xmp := `<rdf:Description exif:DateTimeOriginal="2024-06-15T14:30:00"/>`
date := lrcat.ExtractXMPValue(xmp, "exif:DateTimeOriginal")
// date = "2024-06-15T14:30:00"
```

---

## Lightroom Catalog Format

### Database Schema

Lightroom catalogs are SQLite databases. Key tables:

| Table | Purpose |
|-------|---------|
| `Adobe_images` | Core image records |
| `AgLibraryFile` | File names and extensions |
| `AgLibraryFolder` | Folder structure |
| `AgLibraryRootFolder` | Root folder paths |
| `AgLibraryKeyword` | Keyword definitions |
| `AgLibraryKeywordImage` | Image-keyword associations |
| `AgLibraryCollection` | Collection definitions |
| `AgLibraryCollectionImage` | Image-collection associations |
| `Adobe_AdditionalMetadata` | XMP metadata (compressed) |
| `AgHarvestedExifMetadata` | EXIF data |

### XMP Compression Format

```
[4 bytes: uncompressed length (big-endian)] + [zlib compressed XML]
```

### Timestamp Format

Seconds since January 1, 2001 00:00:00 UTC (Cocoa/Core Foundation epoch).

---

## Examples

### Complete Workflow: Import Photos with Keywords and Collections

```go
package main

import (
    "log"
    "time"

    lrcat "github.com/JeremyProffitt/lrcat-go"
)

func main() {
    // Create catalog
    catalog, err := lrcat.NewCatalog("photos.lrcat")
    if err != nil {
        log.Fatal(err)
    }
    defer catalog.Close()

    // Create keywords
    travel, _ := catalog.AddKeyword("Travel", nil)
    italy, _ := catalog.AddKeyword("Italy", &travel.ID)

    // Create collection
    coll, _ := catalog.AddCollection("Italy 2024", lrcat.CollectionTypeStandard, nil)

    // Scan and import photos
    inputs, _ := lrcat.ScanDirectory("/photos/italy-2024", true)
    _, images, _ := catalog.AddImages(inputs)

    // Tag and organize each image
    for _, img := range images {
        catalog.AddKeywordToImage(img.ID, italy.ID)
        catalog.AddImageToCollection(img.ID, coll.ID)
    }

    log.Printf("Imported %d photos", len(images))
}
```

---

## Testing

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## References

- [Lightroom Database Reference](https://github.com/camerahacks/lightroom-database) - Unofficial table documentation
- [Lightroom-SQL-tools](https://github.com/fdenivac/Lightroom-SQL-tools) - Python library for querying catalogs
- [XmpLibeRator](https://github.com/andyjohnson0/XmpLibeRator) - C# tool for XMP extraction
- [Adobe XMP Specification](https://www.adobe.com/devnet/xmp.html) - Official XMP documentation
