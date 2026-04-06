package models

// Settings defines the application configuration
type Settings struct {
	HashAlgorithm string `json:"hashAlgorithm"`
	BufferSize    int    `json:"bufferSize"`
	Overwrite     bool   `json:"overwrite"`
	EndCheck      bool   `json:"endCheck"`
}

// DefaultSettings returns the default configuration
func DefaultSettings() *Settings {
	return &Settings{
		HashAlgorithm: "xxhash",
		BufferSize:    1024 * 1024, // 1MB
		Overwrite:     false,
		EndCheck:      true,
	}
}
