# lrcat-go

A Go library for creating and manipulating Adobe Lightroom Classic catalog files (.lrcat).

## Overview

Lightroom Classic catalogs are SQLite databases that store references to images, metadata, develop settings, collections, keywords, and more. This library allows you to:

- Create new Lightroom catalog files
- Add images to catalogs with metadata
- Manage folders and root folders
- Work with keywords (including hierarchical keywords)
- Create and manage collections
- Handle XMP metadata (with proper compression/decompression)

## Installation

```bash
go get github.com/JeremyProffitt/lrcat-go
```

## Requirements

- Go 1.21 or later
- CGO enabled (required for SQLite)
- GCC compiler (for building go-sqlite3)
  - **Linux**: `apt-get install build-essential` or equivalent
  - **macOS**: Xcode Command Line Tools
  - **Windows**: MinGW-w64 or TDM-GCC

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
    // Create a new catalog
    catalog, err := lrcat.NewCatalog("/path/to/catalog.lrcat")
    if err != nil {
        log.Fatal(err)
    }
    defer catalog.Close()

    // Add an image
    rating := 4
    image, err := catalog.AddImage(&lrcat.ImageInput{
        FilePath:    "/photos/2024/vacation/IMG_001.jpg",
        CaptureTime: time.Now(),
        Rating:      &rating,
        ColorLabel:  "Green",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Added image with ID: %d", image.ID)
}
```

### Opening an Existing Catalog

```go
// Read-only mode (safe while Lightroom is open)
catalog, err := lrcat.OpenCatalog("/path/to/catalog.lrcat", &lrcat.CatalogOptions{
    ReadOnly: true,
})
if err != nil {
    log.Fatal(err)
}
defer catalog.Close()

count, _ := catalog.ImageCount()
log.Printf("Catalog contains %d images", count)
```

### Adding Multiple Images

```go
inputs := []*lrcat.ImageInput{
    {FilePath: "/photos/IMG_001.jpg", CaptureTime: time.Now()},
    {FilePath: "/photos/IMG_002.jpg", CaptureTime: time.Now()},
    {FilePath: "/photos/IMG_003.jpg", CaptureTime: time.Now()},
}

importSession, images, err := catalog.AddImages(inputs)
if err != nil {
    log.Fatal(err)
}

log.Printf("Imported %d images in session %d", len(images), importSession.ID)
```

### Working with Keywords

```go
// Add a simple keyword
keyword, err := catalog.AddKeyword("vacation", nil)

// Create hierarchical keywords
leafKeyword, err := catalog.CreateHierarchicalKeywords("People/Family/John")

// Add keyword to image
err = catalog.AddKeywordToImage(image.ID, keyword.ID)

// Get all keywords for an image
keywords, err := catalog.GetImageKeywords(image.ID)
```

### Working with Collections

```go
// Create a collection
collection, err := catalog.AddCollection("Best Photos", lrcat.CollectionTypeStandard, nil)

// Create a collection group (folder)
group, err := catalog.AddCollection("Travel", lrcat.CollectionTypeGroup, nil)

// Create collection inside group
subCollection, err := catalog.AddCollection("Italy 2024", lrcat.CollectionTypeStandard, &group.ID)

// Add image to collection
err = catalog.AddImageToCollection(image.ID, collection.ID)

// Get collection images
images, err := catalog.GetCollectionImages(collection.ID)
```

### Working with XMP Metadata

```go
// Generate basic XMP content
rating := 5
xmp := lrcat.GenerateBasicXMP(&rating, "Red", "2024-06-15T14:30:00")

// Set XMP for an image (automatically compressed)
err = catalog.SetXMP(image.ID, xmp)

// Get XMP for an image (automatically decompressed)
xmpContent, err := catalog.GetXMP(image.ID)
```

## API Reference

### Catalog

| Method | Description |
|--------|-------------|
| `NewCatalog(path)` | Create a new catalog |
| `OpenCatalog(path, opts)` | Open an existing catalog |
| `Close()` | Close the catalog |
| `ImageCount()` | Get total image count |
| `FolderCount()` | Get total folder count |
| `GetDBVersion()` | Get Lightroom database version |

### Images

| Method | Description |
|--------|-------------|
| `AddImage(input)` | Add a single image |
| `AddImages(inputs)` | Add multiple images in a transaction |
| `GetImage(id)` | Get image by ID |
| `ListImages()` | List all images |
| `ImageExists(path)` | Check if image exists in catalog |

### Folders

| Method | Description |
|--------|-------------|
| `AddRootFolder(path)` | Add a root folder |
| `AddFolder(rootID, path)` | Add a subfolder |
| `GetOrCreateFolder(rootID, path)` | Get or create a folder |
| `ListRootFolders()` | List all root folders |
| `ListFolders(rootID)` | List folders under a root |

### Keywords

| Method | Description |
|--------|-------------|
| `AddKeyword(name, parentID)` | Add a keyword |
| `GetKeyword(id)` | Get keyword by ID |
| `GetKeywordByName(name)` | Get keyword by name |
| `CreateHierarchicalKeywords(path)` | Create keyword hierarchy |
| `AddKeywordToImage(imageID, keywordID)` | Associate keyword with image |
| `GetImageKeywords(imageID)` | Get keywords for an image |

### Collections

| Method | Description |
|--------|-------------|
| `AddCollection(name, type, parentID)` | Add a collection |
| `GetCollection(id)` | Get collection by ID |
| `AddImageToCollection(imageID, collID)` | Add image to collection |
| `GetCollectionImages(collID)` | Get images in collection |
| `DeleteCollection(id)` | Delete a collection |

### XMP

| Method | Description |
|--------|-------------|
| `SetXMP(imageID, xmp)` | Set XMP for an image |
| `GetXMP(imageID)` | Get XMP for an image |
| `CompressXMP(xmp)` | Compress XMP string |
| `DecompressXMP(data)` | Decompress XMP data |
| `GenerateBasicXMP(...)` | Generate basic XMP content |

## Lightroom Catalog Format

Lightroom Classic catalogs are SQLite databases with a specific schema. Key tables include:

- **Adobe_images**: Core image information
- **AgLibraryFile**: File details (name, extension)
- **AgLibraryFolder**: Folder structure
- **AgLibraryRootFolder**: Root folder paths
- **AgLibraryKeyword**: Keywords
- **AgLibraryKeywordImage**: Image-keyword associations
- **AgLibraryCollection**: Collections
- **Adobe_AdditionalMetadata**: XMP metadata (zlib compressed)

### Timestamp Format

Lightroom uses timestamps relative to January 1, 2001 (the Cocoa/Mac epoch):

```go
timestamp := lrcat.ToLightroomTimestamp(time.Now())
time := lrcat.FromLightroomTimestamp(timestamp)
```

### XMP Compression

XMP metadata in Lightroom is stored as compressed blobs:
- 4-byte big-endian length prefix
- Followed by zlib-compressed XML

This library handles compression/decompression automatically.

## Testing

```bash
go test -v ./...
```

## GitLab CI

This project includes a GitLab CI pipeline that:
- Runs tests on every push
- Tests against multiple Go versions
- Generates coverage reports
- Runs linting with golangci-lint
- Performs security vulnerability scanning

## References

- [Lightroom Database Reference](https://github.com/camerahacks/lightroom-database)
- [Lightroom-SQL-tools](https://github.com/fdenivac/Lightroom-SQL-tools)
- [XmpLibeRator](https://github.com/andyjohnson0/XmpLibeRator)

## License

MIT License
