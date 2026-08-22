package security

// Security limits for document processing.
// These constants are centralized to be shared across all parsers.
const (
	// MaxZipEntrySize is the maximum allowed uncompressed size for a single
	// ZIP entry. This prevents decompression bombs from exhausting memory.
	MaxZipEntrySize = 50 * 1024 * 1024 // 50 MiB

	// MaxTotalZipUncompressedSize is the maximum allowed total uncompressed
	// size across all ZIP entries. This prevents zip bombs with many
	// small-but-expanded entries.
	MaxTotalZipUncompressedSize = 100 * 1024 * 1024 // 100 MiB

	// MaxXMLDepth limits nesting depth to prevent stack exhaustion from
	// pathological XML documents.
	MaxXMLDepth = 256

	// MaxJSONDepth limits nesting depth for JSON documents.
	MaxJSONDepth = 256

	// MaxYAMLDocumentSize limits YAML document size.
	MaxYAMLDocumentSize = 50 * 1024 * 1024 // 50 MiB

	// MaxRTFNestingDepth limits RTF group nesting depth.
	MaxRTFNestingDepth = 256

	// MaxTokenCount limits the number of decoder tokens for streaming parsers.
	MaxTokenCount = 1_000_000
)
