# lrcat-go Library Reference

Go library for Adobe Lightroom Classic catalogs (.lrcat). SQLite-based, requires CGO.

## Import
```go
import lrcat "github.com/JeremyProffitt/lrcat-go"
```

## Catalog
```go
catalog, _ := lrcat.NewCatalog("path.lrcat")           // Create new
catalog, _ := lrcat.OpenCatalog("path.lrcat", nil)     // Open existing
catalog, _ := lrcat.OpenCatalog("path.lrcat", &lrcat.CatalogOptions{ReadOnly: true})
defer catalog.Close()

catalog.ImageCount()      // int, error
catalog.FolderCount()     // int, error
catalog.GetDBVersion()    // string, error
```

## Images
```go
// Add single image
rating := 4
image, _ := catalog.AddImage(&lrcat.ImageInput{
    FilePath:    "/path/to/image.jpg",  // Required
    CaptureTime: time.Now(),            // Required
    Rating:      &rating,               // *int (0-5), nil=no rating
    ColorLabel:  "Red",                 // Red/Yellow/Green/Blue/Purple
    Pick:        1,                     // -1=rejected, 0=unflagged, 1=picked
    FileFormat:  "JPG",                 // Auto-detected if empty
    Width:       &width,                // *int
    Height:      &height,               // *int
    Orientation: &orientation,          // *int (EXIF)
})

// Batch import
inputs := []*lrcat.ImageInput{{FilePath: "...", CaptureTime: time.Now()}, ...}
importSession, images, _ := catalog.AddImages(inputs)

// Query
image, _ := catalog.GetImage(id)
images, _ := catalog.ListImages()
exists, _ := catalog.ImageExists("/path/to/image.jpg")

// Scan directory
inputs, _ := lrcat.ScanDirectory("/photos", true)  // recursive
```

## Folders
```go
// Root folders (absolute paths)
root, _ := catalog.AddRootFolder("/photos")
root, _ := catalog.GetRootFolder(id)
root, _ := catalog.GetRootFolderByPath("/photos")
roots, _ := catalog.ListRootFolders()

// Subfolders (relative to root)
folder, _ := catalog.AddFolder(rootID, "2024/vacation")
folder, _ := catalog.GetOrCreateFolder(rootID, "2024/vacation")
folders, _ := catalog.ListFolders(rootID)
```

## Keywords
```go
kw, _ := catalog.AddKeyword("vacation", nil)              // Root keyword
kw, _ := catalog.AddKeyword("beach", &parentID)           // Child keyword
kw, _ := catalog.CreateHierarchicalKeywords("A/B/C")      // Creates hierarchy
kw, _ := catalog.GetKeyword(id)
kw, _ := catalog.GetKeywordByName("vacation")             // Case-insensitive
kw, _ := catalog.GetOrCreateKeyword("travel", nil)
keywords, _ := catalog.ListKeywords()

// Image associations
catalog.AddKeywordToImage(imageID, keywordID)
catalog.RemoveKeywordFromImage(imageID, keywordID)
keywords, _ := catalog.GetImageKeywords(imageID)
images, _ := catalog.GetKeywordImages(keywordID)
```

## Collections
```go
// Types: CollectionTypeStandard, CollectionTypeSmart, CollectionTypeGroup
coll, _ := catalog.AddCollection("name", lrcat.CollectionTypeStandard, nil)
coll, _ := catalog.AddCollection("child", lrcat.CollectionTypeStandard, &parentID)
coll, _ := catalog.GetCollection(id)
coll, _ := catalog.GetCollectionByName("name")
collections, _ := catalog.ListCollections()
catalog.DeleteCollection(id)

// Image associations
catalog.AddImageToCollection(imageID, collectionID)
catalog.RemoveImageFromCollection(imageID, collectionID)
images, _ := catalog.GetCollectionImages(collectionID)
collections, _ := catalog.GetImageCollections(imageID)
```

## XMP Metadata
```go
catalog.SetXMP(imageID, xmpString)    // Auto-compresses
xmp, _ := catalog.GetXMP(imageID)     // Auto-decompresses

// Manual compression (4-byte big-endian length + zlib)
compressed, _ := lrcat.CompressXMP(xmpString)
decompressed, _ := lrcat.DecompressXMP(blob)

// Generate basic XMP
xmp := lrcat.GenerateBasicXMP(&rating, "Red", "2024-06-15T14:30:00")

// Extract value from XMP
value := lrcat.ExtractXMPValue(xmp, "exif:DateTimeOriginal")
```

## Timestamps
```go
// Lightroom epoch: 2001-01-01 00:00:00 UTC
ts := lrcat.ToLightroomTimestamp(time.Now())     // float64
t := lrcat.FromLightroomTimestamp(ts)            // time.Time
s := lrcat.FormatCaptureTime(time.Now())         // "2006-01-02T15:04:05"
uuid := lrcat.NewUUID()                          // Uppercase UUID string
```

## Key Tables
- `Adobe_images`: Core image data
- `AgLibraryFile`: File names/extensions
- `AgLibraryFolder`: Folder paths
- `AgLibraryRootFolder`: Root paths
- `AgLibraryKeyword`: Keywords
- `AgLibraryKeywordImage`: Keyword-image links
- `AgLibraryCollection`: Collections
- `AgLibraryCollectionImage`: Collection-image links
- `Adobe_AdditionalMetadata`: XMP (zlib compressed)
