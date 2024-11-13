package newstorage

import "context"

// Storage is an interface that provides storing and retrieval of file like things
type Storage interface {
	// Name is the name of the storage implementation
	Name() string

	// Test verifies this storage is functioning and returns an error if not
	Test(ctx context.Context) error

	// Get retrieves the file from the given path
	Get(ctx context.Context, path string) (string, []byte, error)

	// Put stores the given file at the given path
	Put(ctx context.Context, path string, contentType string, contents []byte) (string, error)

	// BatchPut stores the given uploads, returning the URLs of the files after upload
	BatchPut(ctx context.Context, uploads []*Upload) error
}

// Upload is our type for a file to upload
type Upload struct {
	Path        string
	Body        []byte
	ContentType string
	ACL         string
	URL         string
	Error       error
}
