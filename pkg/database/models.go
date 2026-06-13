package database

import (
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// Screenshot 表示数据库中的截图记录
type Screenshot struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	URL            string    `gorm:"index" json:"url"`
	Title          string    `json:"title"`
	Filename       string    `json:"filename"`
	FinalURL       string    `json:"final_url"`
	ResponseCode   int       `json:"response_code"`
	ResponseReason string    `json:"response_reason"`
	Protocol       string    `json:"protocol"`
	ContentLength  int64     `json:"content_length"`
	HTML           string    `json:"html"`
	ProbedAt       time.Time `json:"probed_at"`
	Failed         bool      `json:"failed"`
	FailedReason   string    `json:"failed_reason"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FromResult 从扫描结果创建数据库记录
func (s *Screenshot) FromResult(result *models.Result) {
	s.URL = result.URL
	s.Title = result.Title
	s.Filename = result.Filename
	s.FinalURL = result.FinalURL
	s.ResponseCode = result.ResponseCode
	s.ResponseReason = result.ResponseReason
	s.Protocol = result.Protocol
	s.ContentLength = result.ContentLength
	s.HTML = result.HTML
	s.ProbedAt = result.ProbedAt
	s.Failed = result.Failed
	s.FailedReason = result.FailedReason
}

// ToResult 转换为扫描结果
func (s *Screenshot) ToResult() *models.Result {
	return &models.Result{
		URL:            s.URL,
		Title:          s.Title,
		Filename:       s.Filename,
		FinalURL:       s.FinalURL,
		ResponseCode:   s.ResponseCode,
		ResponseReason: s.ResponseReason,
		Protocol:       s.Protocol,
		ContentLength:  s.ContentLength,
		HTML:           s.HTML,
		ProbedAt:       s.ProbedAt,
		Failed:         s.Failed,
		FailedReason:   s.FailedReason,
	}
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
