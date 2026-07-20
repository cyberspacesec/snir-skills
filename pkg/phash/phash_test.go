package phash

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// createTestImage creates a simple test image with a gradient pattern
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

func encodeToPNG(img image.Image) []byte {
	b := &bytes.Buffer{}
	png.Encode(b, img)
	return b.Bytes()
}

func TestDistanceFromValues(t *testing.T) {
	tests := []struct {
		name     string
		h1       uint64
		h2       uint64
		expected int
	}{
		{
			name:     "identical hashes",
			h1:       0x1234567890abcdef,
			h2:       0x1234567890abcdef,
			expected: 0,
		},
		{
			name:     "one bit difference",
			h1:       0x1234567890abcdef,
			h2:       0x1234567890abcdee,
			expected: 1,
		},
		{
			name:     "completely different",
			h1:       0x0000000000000000,
			h2:       0xffffffffffffffff,
			expected: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DistanceFromValues(tt.h1, tt.h2)
			if result != tt.expected {
				t.Errorf("DistanceFromValues(%x, %x) = %d, want %d", tt.h1, tt.h2, result, tt.expected)
			}
		})
	}
}

func TestIsSimilar(t *testing.T) {
	tests := []struct {
		name      string
		h1        uint64
		h2        uint64
		threshold int
		expected  bool
	}{
		{
			name:      "identical - should be similar",
			h1:        0x1234567890abcdef,
			h2:        0x1234567890abcdef,
			threshold: 5,
			expected:  true,
		},
		{
			name:      "close enough within threshold",
			h1:        0x1234567890abcdef,
			h2:        0x1234567890abcdee,
			threshold: 5,
			expected:  true,
		},
		{
			name:      "too different for threshold",
			h1:        0x0000000000000000,
			h2:        0xffffffffffffffff,
			threshold: 5,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSimilar(tt.h1, tt.h2, tt.threshold)
			if result != tt.expected {
				t.Errorf("IsSimilar(%x, %x, %d) = %v, want %v", tt.h1, tt.h2, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestHashGroup_Add(t *testing.T) {
	hg := NewHashGroup()

	id1 := hg.Add("https://example.com", 0x1234567890abcdef, 5)
	if id1 != 1 {
		t.Errorf("First group ID should be 1, got %d", id1)
	}

	id2 := hg.Add("https://example.org", 0x1234567890abcdef, 5)
	if id2 != id1 {
		t.Errorf("Identical hash should be in same group, got %d vs %d", id2, id1)
	}

	id3 := hg.Add("https://different.com", 0xffffffffffffffff, 5)
	if id3 == id1 {
		t.Error("Very different hash should be in a new group")
	}
}

func TestHashGroup_GetGroupID(t *testing.T) {
	hg := NewHashGroup()
	hg.Add("https://example.com", 0x1234567890abcdef, 5)

	groupID, ok := hg.GetGroupID("https://example.com")
	if !ok {
		t.Error("Should find group ID for existing URL")
	}
	if groupID != 1 {
		t.Errorf("Group ID should be 1, got %d", groupID)
	}

	_, ok = hg.GetGroupID("https://nonexistent.com")
	if ok {
		t.Error("Should not find group ID for nonexistent URL")
	}
}

func TestHashGroup_GetGroupMembers(t *testing.T) {
	hg := NewHashGroup()
	hg.Add("https://example.com", 0x1234567890abcdef, 5)
	hg.Add("https://example.org", 0x1234567890abcdef, 5)
	hg.Add("https://different.com", 0xffffffffffffffff, 5)

	members := hg.GetGroupMembers("https://example.com")
	if len(members) != 2 {
		t.Errorf("Group should have 2 members, got %d", len(members))
	}
}

func TestHashGroup_GetGroupMembers_Nonexistent(t *testing.T) {
	hg := NewHashGroup()
	hg.Add("https://example.com", 0x1234567890abcdef, 5)
	if got := hg.GetGroupMembers("https://nonexistent.com"); got != nil {
		t.Fatalf("不存在的 URL 应返回 nil, got %v", got)
	}
}

func TestHashGroup_GroupCount(t *testing.T) {
	hg := NewHashGroup()
	hg.Add("https://example.com", 0x1234567890abcdef, 5)
	hg.Add("https://different.com", 0xffffffffffffffff, 5)

	if hg.GroupCount() != 2 {
		t.Errorf("Should have 2 groups, got %d", hg.GroupCount())
	}
}

func TestHashValueToHex(t *testing.T) {
	v := uint64(0x1234567890abcdef)
	hex := HashValueToHex(v)
	if hex != "1234567890abcdef" {
		t.Errorf("HashValueToHex(%x) = %s, want 1234567890abcdef", v, hex)
	}
}

func TestHexToHashValue(t *testing.T) {
	hex := "1234567890abcdef"
	v, err := HexToHashValue(hex)
	if err != nil {
		t.Errorf("HexToHashValue error: %v", err)
	}
	if v != 0x1234567890abcdef {
		t.Errorf("HexToHashValue(%s) = %x, want 1234567890abcdef", hex, v)
	}
}

func TestComputeHash_WithRealImage(t *testing.T) {
	img := createTestImage(64, 64)
	pngBytes := encodeToPNG(img)

	result, err := ComputeHash(pngBytes)
	if err != nil {
		t.Errorf("ComputeHash error: %v", err)
	}
	if result == nil {
		t.Fatal("ComputeHash returned nil result")
	}
	if result.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if result.Method != "dhash" {
		t.Errorf("Method should be dhash, got %s", result.Method)
	}
	if result.HashValue == 0 {
		t.Error("HashValue should not be 0 for a real image")
	}
}

func TestComputeAverageHash_WithRealImage(t *testing.T) {
	img := createTestImage(64, 64)
	pngBytes := encodeToPNG(img)

	result, err := ComputeAverageHash(pngBytes)
	if err != nil {
		t.Errorf("ComputeAverageHash error: %v", err)
	}
	if result.Method != "ahash" {
		t.Errorf("Method should be ahash, got %s", result.Method)
	}
}

func TestComputePerceptionHash_WithRealImage(t *testing.T) {
	img := createTestImage(64, 64)
	pngBytes := encodeToPNG(img)

	result, err := ComputePerceptionHash(pngBytes)
	if err != nil {
		t.Errorf("ComputePerceptionHash error: %v", err)
	}
	if result.Method != "phash" {
		t.Errorf("Method should be phash, got %s", result.Method)
	}
}

func TestSimilarImagesHaveCloseHashes(t *testing.T) {
	img1 := createTestImage(64, 64)
	img2 := createTestImage(64, 64) // identical image

	pngBytes1 := encodeToPNG(img1)
	pngBytes2 := encodeToPNG(img2)

	hash1, err := ComputeHash(pngBytes1)
	if err != nil {
		t.Fatalf("ComputeHash error: %v", err)
	}
	hash2, err := ComputeHash(pngBytes2)
	if err != nil {
		t.Fatalf("ComputeHash error: %v", err)
	}

	dist := DistanceFromValues(hash1.HashValue, hash2.HashValue)
	if dist != 0 {
		t.Errorf("Identical images should have distance 0, got %d", dist)
	}
}

func TestComputeHash_InvalidBytes(t *testing.T) {
	_, err := ComputeHash([]byte("not an image"))
	if err == nil {
		t.Error("Should return error for invalid image bytes")
	}
}

func TestComputePerceptionHash_InvalidBytes(t *testing.T) {
	_, err := ComputePerceptionHash([]byte("not an image"))
	if err == nil {
		t.Error("ComputePerceptionHash 应对非法字节返回错误")
	}
}

func TestComputeAverageHash_InvalidBytes(t *testing.T) {
	_, err := ComputeAverageHash([]byte("not an image"))
	if err == nil {
		t.Error("ComputeAverageHash 应对非法字节返回错误")
	}
}

func TestDistance_ValidHashes(t *testing.T) {
	// Distance 依赖 goimagehash 的 "Kind:hex" 字符串格式（如 "a:0000...")，
	// 用两个相同合法 hash 验证成功路径与零距离。
	h := "a:000001071f7fffff"
	dist, err := Distance(h, h)
	if err != nil {
		t.Fatalf("Distance 返回错误: %v", err)
	}
	if dist < 0 {
		t.Fatalf("Distance 不应为负: %d", dist)
	}
	if dist != 0 {
		t.Fatalf("相同 hash Distance 应为 0, 实际 %d", dist)
	}
}

func TestDistance_DifferentHashes(t *testing.T) {
	dist, err := Distance("a:0000000000000000", "a:0000000000000001")
	if err != nil {
		t.Fatalf("Distance 返回错误: %v", err)
	}
	if dist != 1 {
		t.Fatalf("单 bit 差异应为 1, 实际 %d", dist)
	}
}

func TestDistance_InvalidHash(t *testing.T) {
	// 第一个参数非法 → hash1 解析失败分支
	_, err := Distance("not-a-valid-hash", "a:0000000000000001")
	if err == nil {
		t.Fatal("非法 hash1 应返回错误")
	}
	// 第二个参数非法 → hash2 解析失败分支
	_, err = Distance("a:0000000000000001", "also-invalid")
	if err == nil {
		t.Fatal("非法 hash2 应返回错误")
	}
}

func TestDistanceFromValues_Extra(t *testing.T) {
	tests := []struct {
		name string
		a, b uint64
		want int
	}{
		{"相同值", 0x1234, 0x1234, 0},
		{"全异", 0x00FF, 0xFF00, 16},
		{"单bit差异", 0x0001, 0x0000, 1},
		{"全零", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DistanceFromValues(tt.a, tt.b); got != tt.want {
				t.Fatalf("DistanceFromValues(%#x,%#x) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsSimilar_Extra(t *testing.T) {
	if !IsSimilar(0x1234, 0x1234, 5) {
		t.Fatal("相同 hash 应判为相似")
	}
	if IsSimilar(0x00FF, 0xFF00, 5) {
		t.Fatal("距离 16 超过阈值 5 不应判为相似")
	}
	if !IsSimilar(0x0001, 0x0000, 5) {
		t.Fatal("距离 1 在阈值 5 内应判为相似")
	}
}

func TestHexToHashValue_Extra(t *testing.T) {
	v, err := HexToHashValue("0000000000000001")
	if err != nil {
		t.Fatalf("HexToHashValue 错误: %v", err)
	}
	if v != 1 {
		t.Fatalf("HexToHashValue = %#x, want 1", v)
	}
	if _, err := HexToHashValue("xyz"); err == nil {
		t.Fatal("非法 hex 应返回错误")
	}
	// 空字符串在当前实现中走空循环，返回 (0, nil) —— 如实记录此行为。
	if v, err := HexToHashValue(""); err != nil {
		t.Fatalf("空字符串实现返回错误: %v", err)
	} else if v != 0 {
		t.Fatalf("空字符串应返回 0, 实际 %#x", v)
	}
	// 大写 hex 也应正确解析
	if v, err := HexToHashValue("000000000000000F"); err != nil || v != 0xF {
		t.Fatalf("大写 hex 解析失败: v=%#x err=%v", v, err)
	}
}
