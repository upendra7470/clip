package registry

import (
	"sync"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Registry manages the mapping between file types and their corresponding parsers.
type Registry struct {
	mu      sync.RWMutex
	parsers map[filetype.FileType]parser.Parser
}

// New creates a new, empty registry.
func New() *Registry {
	return &Registry{
		parsers: make(map[filetype.FileType]parser.Parser),
	}
}

// Register adds a parser for the given file type.
// Returns an error if the file type is already registered.
func (r *Registry) Register(fileType filetype.FileType, p parser.Parser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.parsers[fileType]; exists {
		return &DuplicateRegistrationError{FileType: fileType}
	}

	r.parsers[fileType] = p
	return nil
}

// Lookup finds the parser for the given file type.
// Returns an error if no parser is registered for the file type.
func (r *Registry) Lookup(fileType filetype.FileType) (parser.Parser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.parsers[fileType]
	if !exists {
		return nil, &ParserNotFoundError{FileType: fileType}
	}

	return p, nil
}

// FileTypes returns a list of all registered file types.
func (r *Registry) FileTypes() []filetype.FileType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fileTypes := make([]filetype.FileType, 0, len(r.parsers))
	for ft := range r.parsers {
		fileTypes = append(fileTypes, ft)
	}
	return fileTypes
}

// Count returns the number of registered parsers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.parsers)
}
