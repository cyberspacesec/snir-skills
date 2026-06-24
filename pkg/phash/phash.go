package phash

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"github.com/corona10/goimagehash"
)

// HashResult contains the perceptual hash and related info
type HashResult struct {
	Hash      string // Hex representation of the hash
	HashValue uint64 // Numeric hash value for comparison
	Method    string // Hash method used (dhash, ahash, phash)
}

// ComputeHash computes a difference hash (dHash) from image bytes
// dHash is fast and effective for screenshot similarity detection
func ComputeHash(imgBytes []byte) (*HashResult, error) {
	img, _, err := decodeImage(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	hash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		return nil, fmt.Errorf("计算dHash失败: %w", err)
	}

	return &HashResult{
		Hash:      fmt.Sprintf("%016x", hash.GetHash()),
		HashValue: hash.GetHash(),
		Method:    "dhash",
	}, nil
}

// ComputePerceptionHash computes a perceptual hash (pHash) from image bytes
// pHash is more robust to scaling and minor modifications
func ComputePerceptionHash(imgBytes []byte) (*HashResult, error) {
	img, _, err := decodeImage(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("计算pHash失败: %w", err)
	}

	return &HashResult{
		Hash:      fmt.Sprintf("%016x", hash.GetHash()),
		HashValue: hash.GetHash(),
		Method:    "phash",
	}, nil
}

// ComputeAverageHash computes an average hash (aHash) from image bytes
// aHash is the simplest and fastest hash, good for exact duplicate detection
func ComputeAverageHash(imgBytes []byte) (*HashResult, error) {
	img, _, err := decodeImage(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	hash, err := goimagehash.AverageHash(img)
	if err != nil {
		return nil, fmt.Errorf("计算aHash失败: %w", err)
	}

	return &HashResult{
		Hash:      fmt.Sprintf("%016x", hash.GetHash()),
		HashValue: hash.GetHash(),
		Method:    "ahash",
	}, nil
}

// Distance computes the Hamming distance between two hash strings
// Lower distance = more similar images (0 = identical)
func Distance(hash1, hash2 string) (int, error) {
	h1, err := goimagehash.ImageHashFromString(hash1)
	if err != nil {
		return 0, fmt.Errorf("解析hash1失败: %w", err)
	}
	h2, err := goimagehash.ImageHashFromString(hash2)
	if err != nil {
		return 0, fmt.Errorf("解析hash2失败: %w", err)
	}

	dist, err := h1.Distance(h2)
	return dist, err
}

// DistanceFromValues computes the Hamming distance between two hash values
func DistanceFromValues(h1, h2 uint64) int {
	// Hamming distance via XOR + popcount
	xor := h1 ^ h2
	distance := 0
	for xor != 0 {
		distance++
		xor &= xor - 1
	}
	return distance
}

// IsSimilar checks if two hash values are similar (within threshold)
// Typical thresholds: 5-10 for near-duplicates, 10-15 for similar pages
func IsSimilar(hash1, hash2 uint64, threshold int) bool {
	return DistanceFromValues(hash1, hash2) <= threshold
}

// HashGroup manages grouping of similar screenshots by perceptual hash
type HashGroup struct {
	mu     sync.RWMutex
	groups map[uint64][]string // hash -> list of URLs
	values map[string]uint64   // URL -> hash value
	ids    map[uint64]uint     // hash -> group ID
	nextID uint
}

// NewHashGroup creates a new HashGroup manager
func NewHashGroup() *HashGroup {
	return &HashGroup{
		groups: make(map[uint64][]string),
		values: make(map[string]uint64),
		ids:    make(map[uint64]uint),
		nextID: 1,
	}
}

// Add adds a URL with its hash value and returns the group ID
// If a similar hash already exists, the URL is added to that group
func (hg *HashGroup) Add(url string, hashValue uint64, threshold int) uint {
	hg.mu.Lock()
	defer hg.mu.Unlock()

	// Check for existing similar hash
	for existingHash, groupID := range hg.ids {
		if DistanceFromValues(existingHash, hashValue) <= threshold {
			// Add to existing group
			hg.groups[existingHash] = append(hg.groups[existingHash], url)
			hg.values[url] = existingHash
			return groupID
		}
	}

	// New group
	groupID := hg.nextID
	hg.nextID++
	hg.ids[hashValue] = groupID
	hg.groups[hashValue] = []string{url}
	hg.values[url] = hashValue
	return groupID
}

// GetGroupID returns the group ID for a URL
func (hg *HashGroup) GetGroupID(url string) (uint, bool) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	hashValue, ok := hg.values[url]
	if !ok {
		return 0, false
	}
	groupID, ok := hg.ids[hashValue]
	return groupID, ok
}

// GetGroupMembers returns all URLs in the same group as the given URL
func (hg *HashGroup) GetGroupMembers(url string) []string {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	hashValue, ok := hg.values[url]
	if !ok {
		return nil
	}
	return hg.groups[hashValue]
}

// GroupCount returns the number of distinct groups
func (hg *HashGroup) GroupCount() int {
	hg.mu.RLock()
	defer hg.mu.RUnlock()
	return len(hg.ids)
}

// decodeImage decodes image bytes into an image.Image
func decodeImage(data []byte) (image.Image, string, error) {
	reader := bytes.NewReader(data)
	return image.Decode(reader)
}

// HashValueToHex converts a uint64 hash value to hex string
func HashValueToHex(v uint64) string {
	return fmt.Sprintf("%016x", v)
}

// HexToHashValue converts a hex string to uint64 hash value
func HexToHashValue(hexStr string) (uint64, error) {
	var v uint64
	for _, c := range hexStr {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint64(c - '0')
		case c >= 'a' && c <= 'f':
			v |= uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v |= uint64(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return v, nil
}
