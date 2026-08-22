package security

import (
	"archive/zip"
	"fmt"
	"io"
)

// ValidateZipFile checks a ZIP entry for security issues before reading.
// It rejects path traversal, absolute paths, and oversized entries.
func ValidateZipFile(zf *zip.File) error {
	if !IsSafeZipEntry(zf.Name) {
		return fmt.Errorf("unsafe ZIP entry name: %q", zf.Name)
	}
	if zf.UncompressedSize64 > MaxZipEntrySize {
		return fmt.Errorf("ZIP entry %q exceeds maximum size limit of %d bytes (actual: %d bytes)", zf.Name, MaxZipEntrySize, zf.UncompressedSize64)
	}
	return nil
}

// ReadZipFile opens and reads a ZIP entry, applying security checks.
// It closes the reader and returns the content or an error.
func ReadZipFile(zf *zip.File) ([]byte, error) {
	if err := ValidateZipFile(zf); err != nil {
		return nil, err
	}

	rc, err := zf.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP entry %q: %w", zf.Name, err)
	}
	defer rc.Close()

	limited := io.LimitReader(rc, MaxZipEntrySize)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP entry %q: %w", zf.Name, err)
	}
	return content, nil
}

// CheckZipArchiveTotalSize checks if the total uncompressed size of all
// entries in a ZIP archive exceeds the maximum allowed total size.
func CheckZipArchiveTotalSize(files []*zip.File) error {
	var total uint64
	for _, f := range files {
		total += f.UncompressedSize64
		if total > MaxTotalZipUncompressedSize {
			return fmt.Errorf("total uncompressed ZIP content exceeds maximum size limit of %d bytes", MaxTotalZipUncompressedSize)
		}
	}
	return nil
}
