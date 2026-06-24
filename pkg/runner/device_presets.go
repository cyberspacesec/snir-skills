package runner

import (
	"fmt"
	"sort"
	"strings"
)

// DevicePreset represents a mobile device with predefined viewport and fingerprint settings
type DevicePreset struct {
	Name              string  // Human-readable name
	UserAgent         string  // Device-specific User-Agent
	Width             int     // Viewport width in pixels
	Height            int     // Viewport height in pixels
	DeviceScaleFactor float64 // Device pixel ratio
	IsMobile          bool    // Whether this is a mobile device
	HasTouch          bool    // Whether to simulate touch events
}

// devicePresets maps device names to their presets
var devicePresets = map[string]DevicePreset{
	// ===== iPhone =====
	"iphone-15-pro": {
		Name:              "iPhone 15 Pro",
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Width:             393,
		Height:            852,
		DeviceScaleFactor: 3.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"iphone-15": {
		Name:              "iPhone 15",
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Width:             393,
		Height:            852,
		DeviceScaleFactor: 3.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"iphone-14-pro-max": {
		Name:              "iPhone 14 Pro Max",
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		Width:             430,
		Height:            932,
		DeviceScaleFactor: 3.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"iphone-se": {
		Name:              "iPhone SE (3rd gen)",
		UserAgent:         "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
		Width:             375,
		Height:            667,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},

	// ===== iPad =====
	"ipad-pro-12": {
		Name:              "iPad Pro 12.9",
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Width:             1024,
		Height:            1366,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"ipad-air": {
		Name:              "iPad Air",
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Width:             820,
		Height:            1180,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"ipad-mini": {
		Name:              "iPad Mini",
		UserAgent:         "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		Width:             744,
		Height:            1133,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},

	// ===== Android Phones =====
	"pixel-8-pro": {
		Name:              "Pixel 8 Pro",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36",
		Width:             412,
		Height:            915,
		DeviceScaleFactor: 2.625,
		IsMobile:          true,
		HasTouch:          true,
	},
	"pixel-8": {
		Name:              "Pixel 8",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36",
		Width:             412,
		Height:            915,
		DeviceScaleFactor: 2.625,
		IsMobile:          true,
		HasTouch:          true,
	},
	"pixel-7": {
		Name:              "Pixel 7",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36",
		Width:             412,
		Height:            915,
		DeviceScaleFactor: 2.625,
		IsMobile:          true,
		HasTouch:          true,
	},
	"samsung-galaxy-s24": {
		Name:              "Samsung Galaxy S24",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S921B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36",
		Width:             384,
		Height:            854,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},
	"samsung-galaxy-s23-ultra": {
		Name:              "Samsung Galaxy S23 Ultra",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36",
		Width:             412,
		Height:            915,
		DeviceScaleFactor: 3.0,
		IsMobile:          true,
		HasTouch:          true,
	},

	// ===== Android Tablets =====
	"samsung-galaxy-tab-s9": {
		Name:              "Samsung Galaxy Tab S9",
		UserAgent:         "Mozilla/5.0 (Linux; Android 14; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Safari/537.36",
		Width:             1280,
		Height:            800,
		DeviceScaleFactor: 2.0,
		IsMobile:          true,
		HasTouch:          true,
	},

	// ===== Desktop (for reference) =====
	"desktop-1080p": {
		Name:              "Desktop 1080p",
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Width:             1920,
		Height:            1080,
		DeviceScaleFactor: 1.0,
		IsMobile:          false,
		HasTouch:          false,
	},
	"desktop-1440p": {
		Name:              "Desktop 1440p",
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Width:             2560,
		Height:            1440,
		DeviceScaleFactor: 1.0,
		IsMobile:          false,
		HasTouch:          false,
	},
	"desktop-4k": {
		Name:              "Desktop 4K",
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Width:             3840,
		Height:            2160,
		DeviceScaleFactor: 1.0,
		IsMobile:          false,
		HasTouch:          false,
	},
}

// GetDevicePreset returns a device preset by name (case-insensitive)
func GetDevicePreset(name string) (*DevicePreset, error) {
	// Try exact match first
	if preset, ok := devicePresets[strings.ToLower(name)]; ok {
		return &preset, nil
	}

	// Try fuzzy match against display names
	for key, preset := range devicePresets {
		if strings.EqualFold(key, name) || strings.EqualFold(preset.Name, name) {
			return &preset, nil
		}
	}

	return nil, fmt.Errorf("unknown device preset: %s (use --list-devices to see available presets)", name)
}

// ListDevicePresets returns all available device presets
func ListDevicePresets() []DevicePreset {
	seen := make(map[string]bool)
	var result []DevicePreset
	for _, preset := range devicePresets {
		if !seen[preset.Name] {
			seen[preset.Name] = true
			result = append(result, preset)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ApplyToOptions applies the device preset to runner Options
func (d *DevicePreset) ApplyToOptions(opts *Options) {
	opts.Chrome.DeviceName = d.Name
	opts.Chrome.UserAgent = d.UserAgent
	opts.Chrome.WindowX = d.Width
	opts.Chrome.WindowY = d.Height
	opts.Chrome.SpoofScreenSize = true
	opts.Chrome.ScreenWidth = d.Width
	opts.Chrome.ScreenHeight = d.Height
	opts.Chrome.DeviceScaleFactor = d.DeviceScaleFactor
	opts.Chrome.IsMobile = d.IsMobile
	opts.Chrome.HasTouch = d.HasTouch

	// Mobile-specific settings
	if d.IsMobile {
		opts.Chrome.Platform = "iPhone"
		if strings.Contains(d.UserAgent, "Android") {
			opts.Chrome.Platform = "Linux armv8l"
		}
	}
}
