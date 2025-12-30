package ports

// FileHasher provides file hashing capabilities
type FileHasher interface {
	// HashFile generates a hash for the given file
	// Returns the hash as a hexadecimal string
	HashFile(filePath string) (string, error)
}
