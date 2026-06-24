package techdetect

import (
	"regexp"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// Technology represents a detected technology with name and optional version
type Technology struct {
	Name     string
	Version  string
	Category string
}

// Fingerprint defines a single detection rule
type Fingerprint struct {
	Name     string            // Technology name (e.g., "jQuery", "WordPress")
	Category string            // Category (e.g., "js", "cms", "webserver")
	HTML     []string          // HTML patterns (regex)
	Headers  map[string]string // Header name -> value pattern (regex)
	Cookies  map[string]string // Cookie name -> value pattern (regex)
	Meta     map[string]string // Meta tag name -> content pattern (regex)
	Script   []string          // Script src patterns (regex)
	Version  string            // Version extraction regex (applied to matched content)
	Priority int              // Higher = more confident
}

// Detector performs technology detection
type Detector struct {
	fingerprints []Fingerprint
	htmlRegexes  map[int][]*regexp.Regexp // index -> compiled HTML regexes
	scriptRegexes map[int][]*regexp.Regexp // index -> compiled script regexes
	headerRegexes map[int]map[string]*regexp.Regexp // index -> header name -> compiled regex
	cookieRegexes map[int]map[string]*regexp.Regexp // index -> cookie name -> compiled regex
	metaRegexes   map[int]map[string]*regexp.Regexp // index -> meta name -> compiled regex
	versionRegexes map[int]*regexp.Regexp // index -> compiled version regex
}

// NewDetector creates a technology detector with built-in fingerprints
func NewDetector() *Detector {
	d := &Detector{}
	d.fingerprints = defaultFingerprints()
	d.compilePatterns()
	return d
}

// NewDetectorWithFingerprints creates a detector with custom fingerprints
func NewDetectorWithFingerprints(fingerprints []Fingerprint) *Detector {
	d := &Detector{}
	d.fingerprints = fingerprints
	d.compilePatterns()
	return d
}

// compilePatterns pre-compiles all regex patterns for performance
func (d *Detector) compilePatterns() {
	d.htmlRegexes = make(map[int][]*regexp.Regexp)
	d.scriptRegexes = make(map[int][]*regexp.Regexp)
	d.headerRegexes = make(map[int]map[string]*regexp.Regexp)
	d.cookieRegexes = make(map[int]map[string]*regexp.Regexp)
	d.metaRegexes = make(map[int]map[string]*regexp.Regexp)
	d.versionRegexes = make(map[int]*regexp.Regexp)

	for i, fp := range d.fingerprints {
		for _, pat := range fp.HTML {
			if re, err := regexp.Compile(pat); err == nil {
				d.htmlRegexes[i] = append(d.htmlRegexes[i], re)
			}
		}
		for _, pat := range fp.Script {
			if re, err := regexp.Compile(pat); err == nil {
				d.scriptRegexes[i] = append(d.scriptRegexes[i], re)
			}
		}
		if len(fp.Headers) > 0 {
			d.headerRegexes[i] = make(map[string]*regexp.Regexp)
			for name, pat := range fp.Headers {
				if re, err := regexp.Compile(pat); err == nil {
					d.headerRegexes[i][name] = re
				}
			}
		}
		if len(fp.Cookies) > 0 {
			d.cookieRegexes[i] = make(map[string]*regexp.Regexp)
			for name, pat := range fp.Cookies {
				if re, err := regexp.Compile(pat); err == nil {
					d.cookieRegexes[i][name] = re
				}
			}
		}
		if len(fp.Meta) > 0 {
			d.metaRegexes[i] = make(map[string]*regexp.Regexp)
			for name, pat := range fp.Meta {
				if re, err := regexp.Compile(pat); err == nil {
					d.metaRegexes[i][name] = re
				}
			}
		}
		if fp.Version != "" {
			if re, err := regexp.Compile(fp.Version); err == nil {
				d.versionRegexes[i] = re
			}
		}
	}
}

// DetectInput contains the data to analyze
type DetectInput struct {
	HTML    string            // Page HTML source
	Headers map[string]string // HTTP response headers
	Cookies map[string]string // Browser cookies
	URL     string            // Page URL
}

// Detect analyzes the input and returns detected technologies
func (d *Detector) Detect(input DetectInput) []Technology {
	var results []Technology
	seen := make(map[string]bool)

	for i, fp := range d.fingerprints {
		if seen[fp.Name] {
			continue
		}

		matched, version := d.matchFingerprint(i, fp, input)
		if matched {
			seen[fp.Name] = true
			tech := Technology{
				Name:     fp.Name,
				Category: fp.Category,
			}
			if version != "" {
				tech.Version = version
			}
			results = append(results, tech)
		}
	}

	return results
}

// DetectFromResult is a convenience method that extracts data from a models.Result
func (d *Detector) DetectFromResult(result *models.Result) []Technology {
	input := DetectInput{
		HTML:    result.HTML,
		Headers: make(map[string]string),
		Cookies: make(map[string]string),
		URL:     result.URL,
	}

	for _, h := range result.Headers {
		input.Headers[h.Name] = h.Value
	}
	for _, c := range result.Cookies {
		input.Cookies[c.Name] = c.Value
	}

	return d.Detect(input)
}

// ToModelsTechnologies converts detected technologies to models.Technology slice
func ToModelsTechnologies(techs []Technology) []models.Technology {
	result := make([]models.Technology, len(techs))
	for i, t := range techs {
		result[i] = models.Technology{
			Name:    t.Name,
			Version: t.Version,
		}
	}
	return result
}

// matchFingerprint checks if a fingerprint matches the input
func (d *Detector) matchFingerprint(idx int, fp Fingerprint, input DetectInput) (bool, string) {
	var versionMatch string

	// Check HTML patterns
	if len(d.htmlRegexes[idx]) > 0 {
		for _, re := range d.htmlRegexes[idx] {
			if re.MatchString(input.HTML) {
				// Try to extract version
				if vr, ok := d.versionRegexes[idx]; ok {
					if m := vr.FindStringSubmatch(input.HTML); len(m) > 1 {
						versionMatch = m[1]
					}
				}
				return true, versionMatch
			}
		}
	}

	// Check script patterns (look for <script src="..."> in HTML)
	if len(d.scriptRegexes[idx]) > 0 {
		for _, re := range d.scriptRegexes[idx] {
			if re.MatchString(input.HTML) {
				if vr, ok := d.versionRegexes[idx]; ok {
					if m := vr.FindStringSubmatch(input.HTML); len(m) > 1 {
						versionMatch = m[1]
					}
				}
				return true, versionMatch
			}
		}
	}

	// Check header patterns
	if hdrRegexes, ok := d.headerRegexes[idx]; ok {
		for hdrName, re := range hdrRegexes {
			for name, value := range input.Headers {
				if strings.EqualFold(name, hdrName) && re.MatchString(value) {
					if vr, ok := d.versionRegexes[idx]; ok {
						if m := vr.FindStringSubmatch(value); len(m) > 1 {
							versionMatch = m[1]
						}
					}
					return true, versionMatch
				}
			}
		}
	}

	// Check cookie patterns
	if ckRegexes, ok := d.cookieRegexes[idx]; ok {
		for ckName, re := range ckRegexes {
			for name, value := range input.Cookies {
				if strings.EqualFold(name, ckName) && re.MatchString(value) {
					return true, versionMatch
				}
			}
		}
	}

	// Check meta tag patterns
	if metaRegexes, ok := d.metaRegexes[idx]; ok {
		for metaName, re := range metaRegexes {
			// Extract meta content from HTML
			metaPattern := regexp.MustCompile(`(?i)<meta[^>]+name=["']` + regexp.QuoteMeta(metaName) + `["'][^>]+content=["']([^"']*)["']`)
			if m := metaPattern.FindStringSubmatch(input.HTML); len(m) > 1 {
				if re.MatchString(m[1]) {
					if vr, ok := d.versionRegexes[idx]; ok {
						if vm := vr.FindStringSubmatch(m[1]); len(vm) > 1 {
							versionMatch = vm[1]
						}
					}
					return true, versionMatch
				}
			}
		}
	}

	return false, ""
}
