package database

import (
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// Options 数据库选项
type Options struct {
	Path string // 数据库文件路径
}

// DB 数据库管理器
type DB struct {
	db      *gorm.DB
	options Options
}

// NewDB 创建新的数据库管理器
func NewDB(options Options) (*DB, error) {
	// 确保目录存在
	dir := filepath.Dir(options.Path)
	if _, err := islazy.CreateDir(dir); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %v", err)
	}

	// 设置日志级别
	logLevel := logger.Silent
	if log.IsDebugEnabled() {
		logLevel = logger.Info
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(options.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 初始化数据库
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %v", err)
	}

	return &DB{
		db:      db,
		options: options,
	}, nil
}

// initDB 初始化数据库
func initDB(db *gorm.DB) error {
	// 自动迁移表结构
	return db.AutoMigrate(
		&Screenshot{},
		&ScanSession{},
		&Tag{},
		&ScreenshotTag{},
	)
}

// Close 关闭数据库连接
func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SaveResult 保存扫描结果
func (d *DB) SaveResult(result *models.Result) error {
	screenshot := &Screenshot{}
	screenshot.FromResult(result)

	return d.db.Create(screenshot).Error
}

// SaveResults 批量保存扫描结果
func (d *DB) SaveResults(results []*models.Result) error {
	screenshots := make([]*Screenshot, 0, len(results))
	for _, result := range results {
		screenshot := &Screenshot{}
		screenshot.FromResult(result)
		screenshots = append(screenshots, screenshot)
	}

	return d.db.Create(&screenshots).Error
}

// GetScreenshot 获取截图信息
func (d *DB) GetScreenshot(id uint) (*Screenshot, error) {
	var screenshot Screenshot
	if err := d.db.First(&screenshot, id).Error; err != nil {
		return nil, err
	}
	return &screenshot, nil
}

// GetScreenshotByURL 通过URL获取截图信息
func (d *DB) GetScreenshotByURL(url string) (*Screenshot, error) {
	var screenshot Screenshot
	if err := d.db.Where("url = ?", url).First(&screenshot).Error; err != nil {
		return nil, err
	}
	return &screenshot, nil
}

// GetAllScreenshots 获取所有截图
func (d *DB) GetAllScreenshots() ([]*Screenshot, error) {
	var screenshots []*Screenshot
	if err := d.db.Find(&screenshots).Error; err != nil {
		return nil, err
	}
	return screenshots, nil
}

// GetScreenshotsByHost 获取指定 host 前缀的所有截图（按扫描时间倒序）
func (d *DB) GetScreenshotsByHost(host string) ([]*Screenshot, error) {
	var screenshots []*Screenshot
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("host 不能为空")
	}
	if err := d.db.Where("host LIKE ?", host+"%").Order("probed_at DESC").Find(&screenshots).Error; err != nil {
		return nil, err
	}
	return screenshots, nil
}

// GetScreenshotsByURL 获取指定 URL 的所有历史截图（按扫描时间倒序）
// 与 GetScreenshotByURL 不同，本方法返回该 URL 的全部历史记录而非仅最新一条
func (d *DB) GetScreenshotsByURL(url string) ([]*Screenshot, error) {
	var screenshots []*Screenshot
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("url 不能为空")
	}
	if err := d.db.Where("url = ?", url).Order("probed_at DESC").Find(&screenshots).Error; err != nil {
		return nil, err
	}
	return screenshots, nil
}

// CreateScanSession 创建扫描会话
func (d *DB) CreateScanSession(name string) (*ScanSession, error) {
	session := &ScanSession{
		Name:      name,
		StartedAt: models.Now(),
	}
	if err := d.db.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

// EndScanSession 结束扫描会话
func (d *DB) EndScanSession(id uint) error {
	return d.db.Model(&ScanSession{}).Where("id = ?", id).Update("ended_at", models.Now()).Error
}

// AddTag 添加标签
func (d *DB) AddTag(name string) (*Tag, error) {
	tag := &Tag{
		Name: name,
	}

	// 先查看标签是否已存在
	var existingTag Tag
	if err := d.db.Where("name = ?", name).First(&existingTag).Error; err == nil {
		return &existingTag, nil
	}

	// 创建新标签
	if err := d.db.Create(tag).Error; err != nil {
		return nil, err
	}
	return tag, nil
}

// AddTagToScreenshot 给截图添加标签
func (d *DB) AddTagToScreenshot(screenshotID uint, tagID uint) error {
	relation := &ScreenshotTag{
		ScreenshotID: screenshotID,
		TagID:        tagID,
	}
	return d.db.Create(relation).Error
}

// GetAllTags 获取所有标签
func (d *DB) GetAllTags() ([]*Tag, error) {
	var tags []*Tag
	if err := d.db.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// ExportResults 导出扫描结果
func (d *DB) ExportResults() ([]*models.Result, error) {
	var screenshots []*Screenshot
	if err := d.db.Find(&screenshots).Error; err != nil {
		return nil, err
	}

	results := make([]*models.Result, 0, len(screenshots))
	for _, screenshot := range screenshots {
		results = append(results, screenshot.ToResult())
	}

	return results, nil
}
