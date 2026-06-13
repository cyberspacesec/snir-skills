package models

import (
	"time"
)

// RequestType are network log types
type RequestType int

const (
	HTTP RequestType = 0
	WS
)

// Result is a Go-Web-Screenshot result
type Result struct {
	Path string `json:"path"` // 截图文件存储路径
	ID   uint   `json:"id" gorm:"primarykey"`

	URL                   string    `json:"url"`
	ProbedAt              time.Time `json:"probed_at"`
	FinalURL              string    `json:"final_url"`
	ResponseCode          int       `json:"response_code"`
	ResponseReason        string    `json:"response_reason"`
	Protocol              string    `json:"protocol"`
	ContentLength         int64     `json:"content_length"`
	HTML                  string    `json:"html" gorm:"index"`
	Title                 string    `json:"title" gorm:"index"`
	PerceptionHash        string    `json:"perception_hash" gorm:"index"`
	PerceptionHashGroupId uint      `json:"perception_hash_group_id" gorm:"index"`
	Screenshot            string    `json:"screenshot"`

	// Name of the screenshot file
	Filename string `json:"filename"` // 截图文件名
	IsPDF    bool   `json:"is_pdf"`

	// Failed flag set if the result should be considered failed
	Failed       bool   `json:"failed"`
	FailedReason string `json:"failed_reason"`

	TLS          TLS          `json:"tls" gorm:"constraint:OnDelete:CASCADE"`
	Technologies []Technology `json:"technologies" gorm:"constraint:OnDelete:CASCADE"`

	Headers []Header     `json:"headers" gorm:"constraint:OnDelete:CASCADE"`
	Network []NetworkLog `json:"network" gorm:"constraint:OnDelete:CASCADE"`
	Console []ConsoleLog `json:"console" gorm:"constraint:OnDelete:CASCADE"`
	Cookies []Cookie     `json:"cookies" gorm:"constraint:OnDelete:CASCADE"`
}

// HeaderMap returns a map of headers
func (r *Result) HeaderMap() map[string][]string {
	headersMap := make(map[string][]string)
	for _, h := range r.Headers {
		headersMap[h.Name] = append(headersMap[h.Name], h.Value)
	}
	return headersMap
}

// Header represents an HTTP header
type Header struct {
	ID       uint   `json:"id" gorm:"primarykey"`
	ResultID uint   `json:"result_id"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

// NetworkLog represents a network request log
type NetworkLog struct {
	ID          uint        `json:"id" gorm:"primarykey"`
	ResultID    uint        `json:"result_id"`
	Type        RequestType `json:"type"`
	URL         string      `json:"url"`
	Method      string      `json:"method"`
	StatusCode  int         `json:"status_code"`
	ContentType string      `json:"content_type"`
	Body        string      `json:"body"`
}

// ConsoleLog represents a console log entry
type ConsoleLog struct {
	ID       uint   `json:"id" gorm:"primarykey"`
	ResultID uint   `json:"result_id"`
	Level    string `json:"level"`
	Message  string `json:"message"`
}

// Cookie represents a browser cookie
type Cookie struct {
	ID       uint   `json:"id" gorm:"primarykey"`
	ResultID uint   `json:"result_id"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
}

// TLS represents TLS information
type TLS struct {
	ID              uint      `json:"id" gorm:"primarykey"`
	ResultID        uint      `json:"result_id"`
	Version         string    `json:"version"`
	CipherSuite     string    `json:"cipher_suite"`
	Issuer          string    `json:"issuer"`
	Subject         string    `json:"subject"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	SANs            string    `json:"sans"`
	FingerprintSHA1 string    `json:"fingerprint_sha1"`
}

// Technology represents a detected technology
type Technology struct {
	ID       uint   `json:"id" gorm:"primarykey"`
	ResultID uint   `json:"result_id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}
