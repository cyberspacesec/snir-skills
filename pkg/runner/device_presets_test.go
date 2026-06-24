package runner

import (
	"testing"
)

func TestGetDevicePreset_ExactMatch(t *testing.T) {
	preset, err := GetDevicePreset("iphone-15-pro")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}
	if preset.Name != "iPhone 15 Pro" {
		t.Errorf("Expected iPhone 15 Pro, got %s", preset.Name)
	}
	if preset.Width != 393 {
		t.Errorf("Expected width 393, got %d", preset.Width)
	}
	if preset.Height != 852 {
		t.Errorf("Expected height 852, got %d", preset.Height)
	}
	if !preset.IsMobile {
		t.Error("Expected IsMobile=true")
	}
	if !preset.HasTouch {
		t.Error("Expected HasTouch=true")
	}
}

func TestGetDevicePreset_CaseInsensitive(t *testing.T) {
	preset, err := GetDevicePreset("iPhone-15-Pro")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}
	if preset.Name != "iPhone 15 Pro" {
		t.Errorf("Expected iPhone 15 Pro, got %s", preset.Name)
	}
}

func TestGetDevicePreset_ByDisplayName(t *testing.T) {
	preset, err := GetDevicePreset("Pixel 8 Pro")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}
	if preset.Width != 412 {
		t.Errorf("Expected width 412, got %d", preset.Width)
	}
	if preset.Height != 915 {
		t.Errorf("Expected height 915, got %d", preset.Height)
	}
}

func TestGetDevicePreset_NotFound(t *testing.T) {
	_, err := GetDevicePreset("nonexistent-device")
	if err == nil {
		t.Error("Expected error for unknown device")
	}
}

func TestListDevicePresets(t *testing.T) {
	presets := ListDevicePresets()
	if len(presets) == 0 {
		t.Error("ListDevicePresets should return presets")
	}

	// Check for some expected presets
	names := make(map[string]bool)
	for _, p := range presets {
		names[p.Name] = true
	}

	expected := []string{"iPhone 15 Pro", "Pixel 8 Pro", "Desktop 1080p"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("Expected preset %q to be in the list", name)
		}
	}
}

func TestDevicePreset_ApplyToOptions(t *testing.T) {
	opts := &Options{}
	preset, err := GetDevicePreset("iphone-15")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}

	preset.ApplyToOptions(opts)

	if opts.Chrome.UserAgent != preset.UserAgent {
		t.Error("UserAgent should be set")
	}
	if opts.Chrome.WindowX != 393 {
		t.Errorf("WindowX should be 393, got %d", opts.Chrome.WindowX)
	}
	if opts.Chrome.WindowY != 852 {
		t.Errorf("WindowY should be 852, got %d", opts.Chrome.WindowY)
	}
	if !opts.Chrome.SpoofScreenSize {
		t.Error("SpoofScreenSize should be true")
	}
	if opts.Chrome.ScreenWidth != 393 {
		t.Errorf("ScreenWidth should be 393, got %d", opts.Chrome.ScreenWidth)
	}
	if opts.Chrome.ScreenHeight != 852 {
		t.Errorf("ScreenHeight should be 852, got %d", opts.Chrome.ScreenHeight)
	}
	if opts.Chrome.DeviceScaleFactor != 3.0 {
		t.Errorf("DeviceScaleFactor should be 3.0, got %f", opts.Chrome.DeviceScaleFactor)
	}
	if !opts.Chrome.IsMobile {
		t.Error("IsMobile should be true")
	}
	if !opts.Chrome.HasTouch {
		t.Error("HasTouch should be true")
	}
}

func TestDevicePreset_AndroidApply(t *testing.T) {
	opts := &Options{}
	preset, err := GetDevicePreset("pixel-8-pro")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}

	preset.ApplyToOptions(opts)

	if opts.Chrome.Platform != "Linux armv8l" {
		t.Errorf("Expected Platform=Linux armv8l, got %s", opts.Chrome.Platform)
	}
}

func TestDevicePreset_DesktopApply(t *testing.T) {
	opts := &Options{}
	preset, err := GetDevicePreset("desktop-1080p")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}

	preset.ApplyToOptions(opts)

	if opts.Chrome.WindowX != 1920 {
		t.Errorf("Expected width 1920, got %d", opts.Chrome.WindowX)
	}
	if opts.Chrome.WindowY != 1080 {
		t.Errorf("Expected height 1080, got %d", opts.Chrome.WindowY)
	}
	if opts.Chrome.ScreenWidth != 1920 {
		t.Errorf("ScreenWidth should be 1920, got %d", opts.Chrome.ScreenWidth)
	}
	if opts.Chrome.ScreenHeight != 1080 {
		t.Errorf("ScreenHeight should be 1080, got %d", opts.Chrome.ScreenHeight)
	}
}

func TestIPadPresets(t *testing.T) {
	devices := []struct {
		name   string
		width  int
		height int
	}{
		{"ipad-pro-12", 1024, 1366},
		{"ipad-air", 820, 1180},
		{"ipad-mini", 744, 1133},
	}

	for _, d := range devices {
		t.Run(d.name, func(t *testing.T) {
			preset, err := GetDevicePreset(d.name)
			if err != nil {
				t.Fatalf("GetDevicePreset error: %v", err)
			}
			if preset.Width != d.width {
				t.Errorf("Expected width %d, got %d", d.width, preset.Width)
			}
			if preset.Height != d.height {
				t.Errorf("Expected height %d, got %d", d.height, preset.Height)
			}
		})
	}
}

func TestSamsungPresets(t *testing.T) {
	preset, err := GetDevicePreset("samsung-galaxy-s24")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}
	if preset.DeviceScaleFactor != 2.0 {
		t.Errorf("Expected deviceScaleFactor 2.0, got %f", preset.DeviceScaleFactor)
	}
	if !preset.IsMobile {
		t.Error("Samsung Galaxy should be mobile")
	}
}

func TestDesktopPresets(t *testing.T) {
	preset, err := GetDevicePreset("desktop-4k")
	if err != nil {
		t.Fatalf("GetDevicePreset error: %v", err)
	}
	if preset.Width != 3840 {
		t.Errorf("Expected 4K width 3840, got %d", preset.Width)
	}
	if preset.Height != 2160 {
		t.Errorf("Expected 4K height 2160, got %d", preset.Height)
	}
	if preset.IsMobile {
		t.Error("Desktop should not be mobile")
	}
}
