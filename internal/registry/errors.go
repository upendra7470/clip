package registry

import (
	"errors"
	"fmt"

	"github.com/upendra7470/clip/internal/filetype"
)

// Common registry errors.
var (
	ErrParserNotFound        = errors.New("parser not found")
	ErrDuplicateRegistration = errors.New("duplicate registration")
)

// UnsupportedFileTypeError is returned when attempting to register or lookup
// an unsupported file type.
type UnsupportedFileTypeError struct {
	FileType filetype.FileType
}

func (e *UnsupportedFileTypeError) Error() string {
	return fmt.Sprintf("unsupported file type: %s", e.FileType)
}

func (e *UnsupportedFileTypeError) Is(target error) bool {
	return target == ErrParserNotFound
}

// ParserNotFoundError is returned when a parser is not found for a file type.
type ParserNotFoundError struct {
	FileType filetype.FileType
}

func (e *ParserNotFoundError) Error() string {
	return fmt.Sprintf("no parser registered for file type: %s", e.FileType)
}

func (e *ParserNotFoundError) Is(target error) bool {
	return target == ErrParserNotFound
}

// DuplicateRegistrationError is returned when attempting to register
// a parser for a file type that already has a parser.
type DuplicateRegistrationError struct {
	FileType filetype.FileType
}

func (e *DuplicateRegistrationError) Error() string {
	return fmt.Sprintf("parser already registered for file type: %s", e.FileType)
}

func (e *DuplicateRegistrationError) Is(target error) bool {
	return target == ErrDuplicateRegistration
}
