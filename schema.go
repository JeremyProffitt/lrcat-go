package lrcat

// Schema contains the SQL statements for creating the Lightroom catalog database tables.
// This schema is based on Adobe Lightroom Classic catalog format.

const schemaVersion = "1901000"

var schemaSQL = []string{
	// Core variables table
	`CREATE TABLE Adobe_variablesTable (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		name,
		type,
		value NOT NULL DEFAULT ''
	)`,

	// Main images table
	`CREATE TABLE Adobe_images (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		aspectRatioCache NOT NULL DEFAULT -1,
		bitDepth NOT NULL DEFAULT 0,
		captureTime,
		colorChannels NOT NULL DEFAULT 0,
		colorLabels NOT NULL DEFAULT '',
		colorMode NOT NULL DEFAULT -1,
		copyCreationTime NOT NULL DEFAULT -63113817600,
		copyName,
		copyReason,
		developSettingsIDCache,
		editLock INTEGER NOT NULL DEFAULT 0,
		fileFormat NOT NULL DEFAULT 'unset',
		fileHeight,
		fileWidth,
		hasMissingSidecars INTEGER,
		masterImage INTEGER,
		orientation,
		originalCaptureTime,
		originalRootEntity INTEGER,
		panningDistanceH,
		panningDistanceV,
		pick NOT NULL DEFAULT 0,
		positionInFolder NOT NULL DEFAULT 'z',
		propertiesCache,
		pyramidIDCache,
		rating,
		rootFile INTEGER NOT NULL DEFAULT 0,
		sidecarStatus,
		touchCount NOT NULL DEFAULT 0,
		touchTime NOT NULL DEFAULT 0
	)`,

	// Additional metadata table (stores XMP)
	`CREATE TABLE Adobe_AdditionalMetadata (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		additionalInfoSet INTEGER NOT NULL DEFAULT 0,
		embeddedXmp INTEGER NOT NULL DEFAULT 0,
		externalXmpIsDirty INTEGER NOT NULL DEFAULT 0,
		image INTEGER,
		incrementalWhiteBalance INTEGER NOT NULL DEFAULT 0,
		internalXmpDigest,
		isRawFile INTEGER NOT NULL DEFAULT 0,
		lastSynchronizedHash,
		lastSynchronizedTimestamp NOT NULL DEFAULT -63113817600,
		metadataPresetID,
		metadataVersion,
		monochrome INTEGER NOT NULL DEFAULT 0,
		xmp NOT NULL DEFAULT ''
	)`,

	// Root folder table
	`CREATE TABLE AgLibraryRootFolder (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		absolutePath UNIQUE NOT NULL DEFAULT '',
		name NOT NULL DEFAULT '',
		relativePathFromCatalog
	)`,

	// Folder table
	`CREATE TABLE AgLibraryFolder (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		parentId INTEGER,
		pathFromRoot NOT NULL DEFAULT '',
		rootFolder INTEGER NOT NULL DEFAULT 0,
		visibility INTEGER
	)`,

	// File table
	`CREATE TABLE AgLibraryFile (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		baseName NOT NULL DEFAULT '',
		errorMessage,
		errorTime,
		extension NOT NULL DEFAULT '',
		externalModTime,
		folder INTEGER NOT NULL DEFAULT 0,
		idx_filename NOT NULL DEFAULT '',
		importHash,
		lc_idx_filename NOT NULL DEFAULT '',
		lc_idx_filenameExtension NOT NULL DEFAULT '',
		md5,
		modTime,
		originalFilename NOT NULL DEFAULT '',
		sidecarExtensions
	)`,

	// Import table
	`CREATE TABLE AgLibraryImport (
		id_local INTEGER PRIMARY KEY,
		imageCount,
		importDate NOT NULL DEFAULT '',
		name
	)`,

	// Import-Image relationship table
	`CREATE TABLE AgLibraryImportImage (
		id_local INTEGER PRIMARY KEY,
		image INTEGER NOT NULL DEFAULT 0,
		import INTEGER NOT NULL DEFAULT 0
	)`,

	// EXIF metadata tables
	`CREATE TABLE AgHarvestedExifMetadata (
		id_local INTEGER PRIMARY KEY,
		image INTEGER,
		aperture,
		cameraModelRef INTEGER,
		cameraSNRef INTEGER,
		dateDay,
		dateMonth,
		dateYear,
		flashFired INTEGER,
		focalLength,
		gpsLatitude,
		gpsLongitude,
		gpsSequence NOT NULL DEFAULT 0,
		hasGPS INTEGER,
		isoSpeedRating,
		lensRef INTEGER,
		shutterSpeed
	)`,

	`CREATE TABLE AgInternedExifCameraModel (
		id_local INTEGER PRIMARY KEY,
		searchIndex,
		value
	)`,

	`CREATE TABLE AgInternedExifLens (
		id_local INTEGER PRIMARY KEY,
		searchIndex,
		value
	)`,

	`CREATE TABLE AgInternedExifCameraSN (
		id_local INTEGER PRIMARY KEY,
		searchIndex,
		value
	)`,

	// IPTC metadata tables
	`CREATE TABLE AgHarvestedIptcMetadata (
		id_local INTEGER PRIMARY KEY,
		image INTEGER,
		cityRef INTEGER,
		copyrightState INTEGER,
		countryRef INTEGER,
		creatorRef INTEGER,
		isoCountryCodeRef INTEGER,
		jobIdentifierRef INTEGER,
		locationDataOrigination NOT NULL DEFAULT 'unset',
		locationGPSSequence NOT NULL DEFAULT -1,
		locationRef INTEGER,
		stateRef INTEGER
	)`,

	`CREATE TABLE AgLibraryIPTC (
		id_local INTEGER PRIMARY KEY,
		altTextAccessibility,
		caption,
		copyright,
		extDescrAccessibility,
		image INTEGER NOT NULL DEFAULT 0
	)`,

	`CREATE TABLE AgInternedIptcCreator (
		id_local INTEGER PRIMARY KEY,
		searchIndex,
		value
	)`,

	// Keywords tables
	`CREATE TABLE AgLibraryKeyword (
		id_local INTEGER PRIMARY KEY,
		id_global UNIQUE NOT NULL,
		dateCreated NOT NULL DEFAULT '',
		genealogy NOT NULL DEFAULT '',
		imageCountCache DEFAULT -1,
		includeOnExport INTEGER NOT NULL DEFAULT 1,
		includeParents INTEGER NOT NULL DEFAULT 1,
		includeSynonyms INTEGER NOT NULL DEFAULT 1,
		keywordType,
		lastApplied,
		lc_name,
		name,
		parent INTEGER
	)`,

	`CREATE TABLE AgLibraryKeywordImage (
		id_local INTEGER PRIMARY KEY,
		image INTEGER NOT NULL DEFAULT 0,
		tag INTEGER NOT NULL DEFAULT 0
	)`,

	// Collections tables
	`CREATE TABLE AgLibraryCollection (
		id_local INTEGER PRIMARY KEY,
		creationId NOT NULL DEFAULT '',
		genealogy NOT NULL DEFAULT '',
		imageCount,
		name NOT NULL DEFAULT '',
		parent INTEGER,
		systemOnly NOT NULL DEFAULT ''
	)`,

	`CREATE TABLE AgLibraryCollectionImage (
		id_local INTEGER PRIMARY KEY,
		collection INTEGER NOT NULL DEFAULT 0,
		image INTEGER NOT NULL DEFAULT 0,
		pick NOT NULL DEFAULT 0,
		positionInCollection
	)`,

	`CREATE TABLE AgLibraryCollectionContent (
		id_local INTEGER PRIMARY KEY,
		collection INTEGER NOT NULL DEFAULT 0,
		content,
		owningModule
	)`,

	// Develop settings table
	`CREATE TABLE Adobe_imageDevelopSettings (
		id_local INTEGER PRIMARY KEY,
		allowFastRender INTEGER,
		beforeSettingsIDCache,
		croppedHeight,
		croppedWidth,
		digest,
		fileHeight,
		fileWidth,
		grayscale INTEGER,
		hasAIMasks INTEGER NOT NULL DEFAULT 0,
		hasBigData INTEGER NOT NULL DEFAULT 0,
		hasDevelopAdjustments INTEGER,
		hasDevelopAdjustmentsEx,
		hasLensBlur INTEGER NOT NULL DEFAULT 0,
		hasMasks INTEGER NOT NULL DEFAULT 0,
		hasPointColor INTEGER NOT NULL DEFAULT 0,
		hasRetouch,
		hasSettings1,
		hasSettings2,
		historySettingsID,
		image INTEGER,
		isHdrEditMode INTEGER NOT NULL DEFAULT 0,
		processVersion,
		profileCorrections,
		removeChromaticAberration,
		settingsID,
		snapshotID,
		text,
		validatedForVersion,
		whiteBalance
	)`,

	// Metadata search index
	`CREATE TABLE AgMetadataSearchIndex (
		id_local INTEGER PRIMARY KEY,
		exifSearchIndex NOT NULL DEFAULT '',
		image INTEGER,
		iptcSearchIndex NOT NULL DEFAULT '',
		otherSearchIndex NOT NULL DEFAULT '',
		searchIndex NOT NULL DEFAULT ''
	)`,

	// Video info table
	`CREATE TABLE AgVideoInfo (
		id_local INTEGER PRIMARY KEY,
		duration,
		frame_rate,
		has_audio INTEGER NOT NULL DEFAULT 1,
		has_video INTEGER NOT NULL DEFAULT 1,
		image INTEGER NOT NULL DEFAULT 0,
		poster_frame NOT NULL DEFAULT '0000000000000000/0000000000000001',
		poster_frame_set_by_user INTEGER NOT NULL DEFAULT 0,
		trim_end NOT NULL DEFAULT '0000000000000000/0000000000000001',
		trim_start NOT NULL DEFAULT '0000000000000000/0000000000000001'
	)`,

	// Indexes for performance
	`CREATE INDEX idx_Adobe_images_rootFile ON Adobe_images (rootFile)`,
	`CREATE INDEX idx_AgLibraryFile_folder ON AgLibraryFile (folder)`,
	`CREATE INDEX idx_AgLibraryFolder_rootFolder ON AgLibraryFolder (rootFolder)`,
	`CREATE INDEX idx_AgHarvestedExifMetadata_image ON AgHarvestedExifMetadata (image)`,
	`CREATE INDEX idx_AgLibraryKeywordImage_image ON AgLibraryKeywordImage (image)`,
	`CREATE INDEX idx_AgLibraryKeywordImage_tag ON AgLibraryKeywordImage (tag)`,
	`CREATE INDEX idx_AgLibraryCollectionImage_collection ON AgLibraryCollectionImage (collection)`,
	`CREATE INDEX idx_AgLibraryCollectionImage_image ON AgLibraryCollectionImage (image)`,
	`CREATE INDEX idx_Adobe_AdditionalMetadata_image ON Adobe_AdditionalMetadata (image)`,
}

var requiredVariables = map[string]string{
	"Adobe_DBVersion":              schemaVersion,
	"AgLibraryKeyword_rootTagID":   "",
	"Adobe_storeProviderData":      "",
	"AgDeleteImages_trashedImages": "",
}
