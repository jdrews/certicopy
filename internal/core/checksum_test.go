package core

import (
	"testing"

	"github.com/spf13/afero"
)

func TestNewHasher(t *testing.T) {
	tests := []struct {
		algo    HashAlgorithm
		wantErr bool
	}{
		{HashXXHash, false},
		{HashBLAKE2b, false},
		{HashSHA256, false},
		{HashSHA1, false},
		{HashMD5, false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			hasher, err := NewHasher(tt.algo)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHasher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hasher == nil {
				t.Error("NewHasher() returned nil hasher without error")
			}
		})
	}
}

func TestCalculateChecksum(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/test.txt"
	content := "hello world"
	afero.WriteFile(fs, filePath, []byte(content), 0644)

	tests := []struct {
		algo HashAlgorithm
		want string
	}{
		{HashMD5, "5eb63bbbe01eeed093cb22bb8f5acdc3"},
		{HashSHA1, "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"},
		{HashSHA256, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"},
		{HashXXHash, "45ab6734b21e6968"},
		{HashBLAKE2b, "256c83b297114d201b30179f3f0ef0cace9783622da5974326b436178aeef610"},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			got, err := CalculateChecksum(fs, filePath, tt.algo)
			if err != nil {
				t.Errorf("CalculateChecksum() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("CalculateChecksum() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateChecksum_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := CalculateChecksum(fs, "/nonexistent", HashMD5)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCalculateChecksum_InvalidAlgo(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/test.txt"
	afero.WriteFile(fs, filePath, []byte("test"), 0644)

	_, err := CalculateChecksum(fs, filePath, "invalid")
	if err == nil {
		t.Error("Expected error for invalid algorithm, got nil")
	}
}
