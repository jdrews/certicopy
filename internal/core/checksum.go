package core

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/afero"
	"golang.org/x/crypto/blake2b"
)

type HashAlgorithm string

const (
	HashXXHash  HashAlgorithm = "xxhash"
	HashBLAKE2b HashAlgorithm = "blake2b"
	HashSHA256  HashAlgorithm = "sha256"
	HashSHA1    HashAlgorithm = "sha1"
	HashMD5     HashAlgorithm = "md5"
)

// CalculateChecksum computes the hash of a file using the specified algorithm
func CalculateChecksum(fs afero.Fs, filePath string, algo HashAlgorithm) (string, error) {
	file, err := fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher, err := NewHasher(algo)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// NewHasher returns a new hash.Hash interface for the given algorithm
func NewHasher(algo HashAlgorithm) (hash.Hash, error) {
	switch algo {
	case HashXXHash:
		return xxhash.New(), nil
	case HashBLAKE2b:
		return blake2b.New256(nil)
	case HashSHA256:
		return sha256.New(), nil
	case HashSHA1:
		return sha1.New(), nil
	case HashMD5:
		return md5.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}
