package database

import (
	"encoding/json"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// Screenshot 表示数据库中的截图记录
type Screenshot struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	URL                   string    `gorm:"index" json:"url"`
	SchemaVersion         string    `gorm:"index" json:"schema_version"`
	Scheme                string    `gorm:"index" json:"scheme"`
	Host                  string    `gorm:"index" json:"host"`
	Port                  int       `gorm:"index" json:"port"`
	Endpoint              string    `gorm:"index" json:"endpoint"`
	Title                 string    `json:"title"`
	Filename              string    `json:"filename"`
	FinalURL              string    `json:"final_url"`
	ResponseCode          int       `json:"response_code"`
	ResponseReason        string    `json:"response_reason"`
	Protocol              string    `json:"protocol"`
	ContentLength         int64     `json:"content_length"`
	HTML                  string    `json:"html"`
	PerceptionHash        string    `gorm:"index" json:"perception_hash"`
	PerceptionHashGroupId uint      `gorm:"index" json:"perception_hash_group_id"`
	TLSJSON               string    `json:"tls_json"`
	TechnologiesJSON      string    `json:"technologies_json"`
	HeadersJSON           string    `json:"headers_json"`
	NetworkJSON           string    `json:"network_json"`
	ConsoleJSON           string    `json:"console_json"`
	CookiesJSON           string    `json:"cookies_json"`
	ProbedAt              time.Time `json:"probed_at"`
	Failed                bool      `json:"failed"`
	FailedReason          string    `json:"failed_reason"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// FromResult 从扫描结果创建数据库记录
func (s *Screenshot) FromResult(result *models.Result) {
	models.EnrichEndpoint(result)
	s.URL = result.URL
	s.SchemaVersion = result.SchemaVersion
	s.Scheme = result.Scheme
	s.Host = result.Host
	s.Port = result.Port
	s.Endpoint = result.Endpoint
	s.Title = result.Title
	s.Filename = result.Filename
	s.FinalURL = result.FinalURL
	s.ResponseCode = result.ResponseCode
	s.ResponseReason = result.ResponseReason
	s.Protocol = result.Protocol
	s.ContentLength = result.ContentLength
	s.HTML = result.HTML
	s.PerceptionHash = result.PerceptionHash
	s.PerceptionHashGroupId = result.PerceptionHashGroupId
	s.TLSJSON = marshalJSON(result.TLS)
	s.TechnologiesJSON = marshalJSON(result.Technologies)
	s.HeadersJSON = marshalJSON(result.Headers)
	s.NetworkJSON = marshalJSON(result.Network)
	s.ConsoleJSON = marshalJSON(result.Console)
	s.CookiesJSON = marshalJSON(result.Cookies)
	s.ProbedAt = result.ProbedAt
	s.Failed = result.Failed
	s.FailedReason = result.FailedReason
}

// ToResult 转换为扫描结果
func (s *Screenshot) ToResult() *models.Result {
	result := &models.Result{
		URL:                   s.URL,
		SchemaVersion:         s.SchemaVersion,
		Scheme:                s.Scheme,
		Host:                  s.Host,
		Port:                  s.Port,
		Endpoint:              s.Endpoint,
		Title:                 s.Title,
		Filename:              s.Filename,
		FinalURL:              s.FinalURL,
		ResponseCode:          s.ResponseCode,
		ResponseReason:        s.ResponseReason,
		Protocol:              s.Protocol,
		ContentLength:         s.ContentLength,
		HTML:                  s.HTML,
		PerceptionHash:        s.PerceptionHash,
		PerceptionHashGroupId: s.PerceptionHashGroupId,
		ProbedAt:              s.ProbedAt,
		Failed:                s.Failed,
		FailedReason:          s.FailedReason,
	}
	unmarshalJSON(s.TLSJSON, &result.TLS)
	unmarshalJSON(s.TechnologiesJSON, &result.Technologies)
	unmarshalJSON(s.HeadersJSON, &result.Headers)
	unmarshalJSON(s.NetworkJSON, &result.Network)
	unmarshalJSON(s.ConsoleJSON, &result.Console)
	unmarshalJSON(s.CookiesJSON, &result.Cookies)
	models.EnrichEndpoint(result)
	return result
}

func marshalJSON(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return ""
	}
	return string(data)
}

func unmarshalJSON(data string, value interface{}) {
	if data == "" {
		return
	}
	_ = json.Unmarshal([]byte(data), value)
}

// ScanSession 表示一次扫描会话
type ScanSession struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tag 表示截图的标签
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScreenshotTag 表示截图和标签的多对多关系
type ScreenshotTag struct {
	ScreenshotID uint `gorm:"primaryKey" json:"screenshot_id"`
	TagID        uint `gorm:"primaryKey" json:"tag_id"`
}
