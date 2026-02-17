package models

// Settings defines the application configuration
type Settings struct {
	Theme             string `json:"theme"`
	DefaultDestPath   string `json:"defaultDestPath"`
	HashAlgorithm     string `json:"hashAlgorithm"`
	BufferSize        int    `json:"bufferSize"`
	ShowNotifications bool   `json:"showNotifications"`
	PlaySoundOnFinish bool   `json:"playSoundOnFinish"`
	AutoVerify        bool   `json:"autoVerify"`
}

// DefaultSettings returns the default configuration
func DefaultSettings() *Settings {
	return &Settings{
		Theme:             "dark",
		DefaultDestPath:   "",
		HashAlgorithm:     "xxhash",
		BufferSize:        1024 * 1024, // 1MB
		ShowNotifications: true,
		PlaySoundOnFinish: true,
		AutoVerify:        true,
	}
}
