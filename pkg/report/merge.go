package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// MergeOptions 包含报告合并选项
type MergeOptions struct {
	SourceFiles []string // 源文件列表
	SourcePath  string   // 源文件目录
	OutputFile  string   // 输出文件
}

// Merge 合并多个报告
func Merge(options MergeOptions) error {
	// 验证输出文件
	if options.OutputFile == "" {
		return fmt.Errorf("输出文件不能为空")
	}

	// 获取源文件列表
	var sourceFiles []string
	if len(options.SourceFiles) > 0 {
		sourceFiles = options.SourceFiles
	} else if options.SourcePath != "" {
		// 从目录读取源文件
		var err error
		sourceFiles, err = findSourceFiles(options.SourcePath)
		if err != nil {
			return fmt.Errorf("查找源文件失败: %v", err)
		}
	} else {
		return fmt.Errorf("必须指定源文件列表或源文件目录")
	}

	if len(sourceFiles) == 0 {
		return fmt.Errorf("没有找到可合并的源文件")
	}

	log.Info("开始合并报告", "files", len(sourceFiles))

	// 读取所有结果
	var allResults []*models.Result
	for _, sourceFile := range sourceFiles {
		// 获取文件扩展名
		ext := strings.ToLower(filepath.Ext(sourceFile))
		if !isValidExtension(ext) {
			log.Warn("跳过不支持的文件格式", "file", sourceFile, "ext", ext)
			continue
		}

		// 读取结果
		log.Info("正在读取文件", "file", sourceFile)
		results, err := readResults(sourceFile, ext)
		if err != nil {
			log.Error("读取文件失败", "file", sourceFile, "error", err)
			continue
		}

		log.Info("已读取记录", "file", sourceFile, "count", len(results))
		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		return fmt.Errorf("没有从源文件中读取到有效的记录")
	}

	// 获取输出文件扩展名
	outputExt := strings.ToLower(filepath.Ext(options.OutputFile))
	if !isValidExtension(outputExt) {
		return fmt.Errorf("不支持的输出文件格式: %s", outputExt)
	}

	// 写入所有结果
	if err := writeResults(options.OutputFile, outputExt, allResults); err != nil {
		return fmt.Errorf("写入合并后的报告失败: %v", err)
	}

	log.Info("报告合并完成", "output", options.OutputFile, "records", len(allResults))
	return nil
}

// findSourceFiles 在指定目录中查找可合并的源文件
func findSourceFiles(dir string) ([]string, error) {
	// 检查目录是否存在
	if !islazy.DirExists(dir) {
		return nil, fmt.Errorf("目录不存在: %s", dir)
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否是文件
		if !info.IsDir() {
			// 检查扩展名
			ext := strings.ToLower(filepath.Ext(path))
			if isValidExtension(ext) {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, err
}

// deduplicateResults 去除重复的结果
func deduplicateResults(results []*models.Result) []*models.Result {
	urlMap := make(map[string]bool)
	var uniqueResults []*models.Result

	for _, result := range results {
		if !urlMap[result.URL] {
			urlMap[result.URL] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	return uniqueResults
}
