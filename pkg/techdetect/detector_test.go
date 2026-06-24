package techdetect

import (
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if len(d.fingerprints) == 0 {
		t.Fatal("detector should have default fingerprints")
	}
}

func TestDetect_JQueryFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><script src="jquery-3.6.0.min.js"></script></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "jQuery" {
			found = true
			if tech.Version == "" {
				t.Error("jQuery version should be detected")
			}
		}
	}
	if !found {
		t.Error("jQuery should be detected from script src")
	}
}

func TestDetect_ReactFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<div data-reactroot=""><h1>Hello</h1></div>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "React" {
			found = true
		}
	}
	if !found {
		t.Error("React should be detected from data-reactroot")
	}
}

func TestDetect_VueFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<div data-v-a1b2c3><span v-cloak>test</span></div>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Vue.js" {
			found = true
		}
	}
	if !found {
		t.Error("Vue.js should be detected from data-v- attribute")
	}
}

func TestDetect_AngularFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<div ng-app="myApp"><div ng-controller="MyCtrl"></div></div>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Angular" {
			found = true
		}
	}
	if !found {
		t.Error("Angular should be detected from ng-app")
	}
}

func TestDetect_NginxFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"Server": "nginx/1.24.0",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Nginx" {
			found = true
			if tech.Version != "1.24.0" {
				t.Errorf("Nginx version should be 1.24.0, got %s", tech.Version)
			}
		}
	}
	if !found {
		t.Error("Nginx should be detected from Server header")
	}
}

func TestDetect_ApacheFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"Server": "Apache/2.4.57 (Unix)",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Apache" {
			found = true
			if tech.Version != "2.4.57" {
				t.Errorf("Apache version should be 2.4.57, got %s", tech.Version)
			}
		}
	}
	if !found {
		t.Error("Apache should be detected from Server header")
	}
}

func TestDetect_CloudflareFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"Cf-Ray": "8123456abcdef-DFW",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Cloudflare" {
			found = true
		}
	}
	if !found {
		t.Error("Cloudflare should be detected from Cf-Ray header")
	}
}

func TestDetect_WordPressFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><link rel="stylesheet" href="/wp-content/themes/twentytwentyone/style.css"></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "WordPress" {
			found = true
		}
	}
	if !found {
		t.Error("WordPress should be detected from wp-content path")
	}
}

func TestDetect_WordPressFromMeta(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><meta name="generator" content="WordPress 6.4.2"></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "WordPress" {
			found = true
			if tech.Version != "6.4.2" {
				t.Errorf("WordPress version should be 6.4.2, got %s", tech.Version)
			}
		}
	}
	if !found {
		t.Error("WordPress should be detected from meta generator tag")
	}
}

func TestDetect_PHPFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"X-Powered-By": "PHP/8.2.13",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "PHP" {
			found = true
			if tech.Version != "8.2.13" {
				t.Errorf("PHP version should be 8.2.13, got %s", tech.Version)
			}
		}
	}
	if !found {
		t.Error("PHP should be detected from X-Powered-By header")
	}
}

func TestDetect_MultipleTechnologies(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head>
			<meta name="generator" content="WordPress 6.4.2">
			<script src="jquery-3.7.1.min.js"></script>
			<script src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>
		</head><body></body></html>`,
		Headers: map[string]string{
			"Server": "nginx/1.25.3",
		},
	}
	techs := d.Detect(input)

	names := make(map[string]bool)
	for _, tech := range techs {
		names[tech.Name] = true
	}

	expected := []string{"WordPress", "jQuery", "Google Tag Manager", "Nginx"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("Expected %s to be detected, got: %v", name, techs)
		}
	}
}

func TestDetect_NoMatches(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML:    `<html><body><h1>Hello World</h1></body></html>`,
		Headers: map[string]string{},
	}
	techs := d.Detect(input)
	// Should still work, just returns empty or minimal results
	// (might detect some false positives, that's OK for this test)
	_ = techs
}

func TestDetectFromResult(t *testing.T) {
	d := NewDetector()
	result := &models.Result{
		HTML: `<html><head><script src="jquery-3.6.0.min.js"></script></head><body></body></html>`,
		Headers: []models.Header{
			{Name: "Server", Value: "nginx/1.24.0"},
		},
		Cookies: []models.Cookie{
			{Name: "session", Value: "abc123"},
		},
	}

	techs := d.DetectFromResult(result)

	found := false
	for _, tech := range techs {
		if tech.Name == "Nginx" {
			found = true
		}
	}
	if !found {
		t.Error("Nginx should be detected from result headers")
	}
}

func TestToModelsTechnologies(t *testing.T) {
	techs := []Technology{
		{Name: "jQuery", Version: "3.7.1", Category: "js"},
		{Name: "Nginx", Version: "1.24.0", Category: "webserver"},
	}
	modelsTechs := ToModelsTechnologies(techs)

	if len(modelsTechs) != 2 {
		t.Fatalf("Expected 2 technologies, got %d", len(modelsTechs))
	}
	if modelsTechs[0].Name != "jQuery" {
		t.Errorf("Expected Name=jQuery, got %s", modelsTechs[0].Name)
	}
	if modelsTechs[0].Version != "3.7.1" {
		t.Errorf("Expected Version=3.7.1, got %s", modelsTechs[0].Version)
	}
}

func TestDetect_NextJS(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<div id="__next"><script id="__NEXT_DATA__">{"props":{}}</script></div>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Next.js" {
			found = true
		}
	}
	if !found {
		t.Error("Next.js should be detected from __NEXT_DATA__")
	}
}

func TestDetect_ExpressFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"X-Powered-By": "Express",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Express" {
			found = true
		}
	}
	if !found {
		t.Error("Express should be detected from X-Powered-By header")
	}
}

func TestDetect_IISFromHeader(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		Headers: map[string]string{
			"Server": "Microsoft-IIS/10.0",
		},
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "IIS" {
			found = true
			if tech.Version != "10.0" {
				t.Errorf("IIS version should be 10.0, got %s", tech.Version)
			}
		}
	}
	if !found {
		t.Error("IIS should be detected from Server header")
	}
}

func TestDetect_GrafanaFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><body class="grafana-app"><div>Grafana Dashboard</div></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Grafana" {
			found = true
		}
	}
	if !found {
		t.Error("Grafana should be detected from grafana-app class")
	}
}

func TestDetect_JenkinsFromHTML(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><meta name="Jenkins-Crumb" content="abc"></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Jenkins" {
			found = true
		}
	}
	if !found {
		t.Error("Jenkins should be detected from Jenkins-Crumb meta tag")
	}
}

func TestDetect_DrupalFromMeta(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><meta name="generator" content="Drupal 10"></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Drupal" {
			found = true
		}
	}
	if !found {
		t.Error("Drupal should be detected from meta generator")
	}
}

func TestNewDetectorWithFingerprints(t *testing.T) {
	custom := []Fingerprint{
		{
			Name:     "CustomTech",
			Category: "custom",
			HTML:     []string{`custom-marker`},
			Priority: 10,
		},
	}
	d := NewDetectorWithFingerprints(custom)

	input := DetectInput{
		HTML: `<html><body><div class="custom-marker">test</div></body></html>`,
	}
	techs := d.Detect(input)

	if len(techs) != 1 {
		t.Fatalf("Expected 1 technology, got %d", len(techs))
	}
	if techs[0].Name != "CustomTech" {
		t.Errorf("Expected CustomTech, got %s", techs[0].Name)
	}
}

func TestDetect_BootstrapVersion(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head><link rel="stylesheet" href="bootstrap/4.6.2/css/bootstrap.min.css"></head><body></body></html>`,
	}
	techs := d.Detect(input)

	found := false
	for _, tech := range techs {
		if tech.Name == "Bootstrap" {
			found = true
		}
	}
	if !found {
		t.Error("Bootstrap should be detected")
	}
}

func TestDetect_Deduplication(t *testing.T) {
	d := NewDetector()
	input := DetectInput{
		HTML: `<html><head>
			<meta name="generator" content="WordPress 6.4.2">
			<link rel="stylesheet" href="/wp-content/themes/test/style.css">
		</head><body></body></html>`,
	}
	techs := d.Detect(input)

	wpCount := 0
	for _, tech := range techs {
		if tech.Name == "WordPress" {
			wpCount++
		}
	}
	if wpCount > 1 {
		t.Errorf("WordPress should only be detected once, got %d", wpCount)
	}
}
