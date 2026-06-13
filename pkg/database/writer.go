package database

import (
	"fmt"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// DBWriter 实现 runner.Writer 接口，将结果写入数据库
type DBWriter struct {
	db *DB
}

// NewDBWriter 创建新的数据库写入器
func NewDBWriter(db *DB) *DBWriter {
	return &DBWriter{
		db: db,
	}
}

// Write 实现 runner.Writer 接口，将结果写入数据库
func (w *DBWriter) Write(result *models.Result) error {
	err := w.db.SaveResult(result)
	if err != nil {
		log.Error("保存结果到数据库失败", "error", err, "url", result.URL)
		return fmt.Errorf("保存结果到数据库失败: %v", err)
	}
	log.Debug("已保存结果到数据库", "url", result.URL)
	return nil
}

// Close 实现 runner.Writer 接口，关闭数据库连接
func (w *DBWriter) Close() error {
	return nil // DB会通过主程序关闭
}
