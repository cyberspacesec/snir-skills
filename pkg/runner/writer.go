package runner

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// JSONLWriter implements the Writer interface for JSONL files
type JSONLWriter struct {
	file *os.File
}

// NewJSONLWriter creates a new JSONL writer
func NewJSONLWriter(filePath string) (*JSONLWriter, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if _, err := islazy.CreateDir(dir); err != nil {
		return nil, err
	}

	// 打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &JSONLWriter{
		file: file,
	}, nil
}

// Write implements the Writer interface
func (w *JSONLWriter) Write(result *models.Result) error {
	// 将结果序列化为JSON
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// 写入文件
	if _, err := w.file.Write(data); err != nil {
		return err
	}

	// 写入换行符
	if _, err := w.file.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// Close implements the Writer interface
func (w *JSONLWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// CSVWriter implements the Writer interface for CSV files
type CSVWriter struct {
	writer *csv.Writer
	file   *os.File
	header bool
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(filePath string) (*CSVWriter, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if _, err := islazy.CreateDir(dir); err != nil {
		return nil, err
	}

	// 检查文件是否存在
	fileExists := islazy.FileExists(filePath)

	// 打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &CSVWriter{
		writer: csv.NewWriter(file),
		file:   file,
		header: !fileExists, // 如果文件不存在，需要写入表头
	}, nil
}

// Write implements the Writer interface
func (w *CSVWriter) Write(result *models.Result) error {
	// 如果需要写入表头
	if w.header {
		header := []string{
			"URL", "标题", "响应码", "截图路径", "扫描时间", "最终URL", "状态",
		}
		if err := w.writer.Write(header); err != nil {
			return err
		}
		w.header = false
	}

	// 准备数据行
	status := "成功"
	if result.Failed {
		status = "失败: " + result.FailedReason
	}

	row := []string{
		result.URL,
		result.Title,
		fmt.Sprintf("%d", result.ResponseCode),
		result.Filename,
		result.ProbedAt.Format(time.RFC3339),
		result.FinalURL,
		status,
	}

	// 写入数据行
	if err := w.writer.Write(row); err != nil {
		return err
	}

	// 刷新缓冲区
	w.writer.Flush()

	return nil
}

// Close implements the Writer interface
func (w *CSVWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// StdoutWriter implements the Writer interface for stdout
type StdoutWriter struct{}

// NewStdoutWriter creates a new stdout writer
func NewStdoutWriter() *StdoutWriter {
	return &StdoutWriter{}
}

// Write implements the Writer interface
func (w *StdoutWriter) Write(result *models.Result) error {
	// 输出基本信息
	log.Info("扫描结果", "url", result.URL)
	log.Info("页面标题", "title", result.Title)
	log.Info("响应状态码", "code", result.ResponseCode)

	if result.Filename != "" {
		log.Info("截图保存路径", "path", result.Filename)
	}

	if result.Failed {
		log.Error("扫描失败", "reason", result.FailedReason)
	}

	return nil
}

// Close implements the Writer interface
func (w *StdoutWriter) Close() error {
	return nil
}

// CreateWriters creates writers based on options
func CreateWriters(opts *Options) ([]Writer, error) {
	var writers []Writer

	// 创建数据库写入器
	if opts.DB.Enable {
		dbOptions := database.Options{
			Path: opts.DB.Path,
		}

		// 创建数据库连接
		db, err := database.NewDB(dbOptions)
		if err != nil {
			return nil, fmt.Errorf("创建数据库连接失败: %v", err)
		}

		// 创建数据库写入器
		dbWriter := database.NewDBWriter(db)
		writers = append(writers, dbWriter)
		log.Debug("已创建数据库写入器", "path", dbOptions.Path)
	}

	// 创建JSONL写入器
	if opts.Writer.Jsonl {
		filePath := opts.Writer.JsonlFile
		if filePath == "" {
			filePath = "results.jsonl"
		}

		jsonlWriter, err := NewJSONLWriter(filePath)
		if err != nil {
			return nil, fmt.Errorf("创建JSONL写入器失败: %v", err)
		}
		writers = append(writers, jsonlWriter)
		log.Debug("已创建JSONL写入器", "path", filePath)
	}

	// 创建CSV写入器
	if opts.Writer.Csv {
		filePath := opts.Writer.CsvFile
		if filePath == "" {
			filePath = "results.csv"
		}

		csvWriter, err := NewCSVWriter(filePath)
		if err != nil {
			return nil, fmt.Errorf("创建CSV写入器失败: %v", err)
		}
		writers = append(writers, csvWriter)
		log.Debug("已创建CSV写入器", "path", filePath)
	}

	// 创建标准输出写入器
	if opts.Writer.Stdout || len(writers) == 0 {
		writers = append(writers, NewStdoutWriter())
		log.Debug("已创建标准输出写入器")
	}

	return writers, nil
}
