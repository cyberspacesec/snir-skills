package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// HTMLOptions 包含HTML报告选项
type HTMLOptions struct {
	InputFile  string // 输入文件
	OutputPath string // 输出路径
}

// ReportData 表示报告数据结构
type ReportData struct {
	GeneratedAt  string
	Results      []ReportResult
	SuccessCount int
	FailCount    int
	TechCount    int
}

// ReportResult 表示报告结果项
type ReportResult struct {
	URL             string
	Title           string
	Screenshot      string
	ResponseCode    int
	StatusCodeClass string
	ProbedAt        time.Time
	FinalURL        string
	ResponseReason  string
	Protocol        string
	TLSVersion      string
	TLSCipher       string
	TLSIssuer       string
	TLSSANs         string
	IsPDF           bool
	Failed          bool
	FailedReason    string
	Technologies    []string // tech names
	TechBadges      []string // HTML badges
	ConsoleErrors   []string
	ContentLength   string
	ServerHeader    string
	PoweredByHeader string
	Cloudflare      bool
}

// RichHTMLTemplate 是增强的HTML报告模板，包含丰富信息
const RichHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>网页截图扫描报告</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif;
            line-height: 1.6;
            color: #1a1a2e;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        h1 {
            color: #fff;
            font-size: 2em;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
        }
        .header { margin-bottom: 30px; }
        .header p { color: rgba(255,255,255,0.8); }

        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: rgba(255,255,255,0.95);
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 4px 15px rgba(0,0,0,0.1);
            text-align: center;
        }
        .stat-card .number { font-size: 2.5em; font-weight: bold; color: #667eea; }
        .stat-card .label { color: #666; font-size: 0.9em; margin-top: 5px; }

        .result-card {
            background: rgba(255,255,255,0.95);
            border-radius: 12px;
            overflow: hidden;
            margin-bottom: 25px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .result-card:hover { transform: translateY(-2px); }

        .result-header {
            padding: 20px;
            border-bottom: 1px solid #eee;
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            flex-wrap: wrap;
            gap: 10px;
        }
        .result-title-section { flex: 1; min-width: 200px; }
        .result-title { font-size: 1.2em; font-weight: 600; color: #1a1a2e; margin-bottom: 5px; }
        .result-url {
            font-size: 0.85em;
            color: #667eea;
            word-break: break-all;
        }
        .result-url a { color: #667eea; text-decoration: none; }
        .result-url a:hover { text-decoration: underline; }

        .badge-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
        .badge {
            display: inline-block;
            padding: 4px 10px;
            border-radius: 20px;
            font-size: 0.75em;
            font-weight: 600;
            letter-spacing: 0.5px;
        }
        .badge-2xx { background: #d4edda; color: #155724; }
        .badge-3xx { background: #d1ecf1; color: #0c5460; }
        .badge-4xx { background: #fff3cd; color: #856404; }
        .badge-5xx { background: #f8d7da; color: #721c24; }
        .badge-0xx { background: #e2e3e5; color: #383d41; }
        .badge-tech { background: #e8eaf6; color: #283593; }
        .badge-tls { background: #e0f2f1; color: #004d40; }
        .badge-cf { background: #fff8e1; color: #f57f17; }

        .result-body { display: grid; grid-template-columns: 1fr 400px; gap: 0; }
        @media (max-width: 900px) {
            .result-body { grid-template-columns: 1fr; }
            .result-sidebar { border-left: none; border-top: 1px solid #eee; }
        }
        .result-main { padding: 0; }
        .result-main img {
            width: 100%;
            max-height: 500px;
            object-fit: contain;
            background: #f5f5f5;
            display: block;
        }
        .no-screenshot {
            height: 200px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: #f5f5f5;
            color: #999;
            font-size: 1.1em;
        }
        .result-sidebar {
            padding: 20px;
            border-left: 1px solid #eee;
            font-size: 0.9em;
            max-height: 500px;
            overflow-y: auto;
        }
        .sidebar-section { margin-bottom: 18px; }
        .sidebar-section h4 {
            font-size: 0.75em;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: #999;
            margin-bottom: 8px;
            border-bottom: 1px solid #eee;
            padding-bottom: 4px;
        }
        .sidebar-section code {
            display: block;
            font-size: 0.85em;
            background: #f8f9fa;
            padding: 4px 8px;
            border-radius: 4px;
            margin: 3px 0;
            word-break: break-all;
        }
        .meta-grid { display: grid; grid-template-columns: auto 1fr; gap: 4px 10px; font-size: 0.85em; }
        .meta-label { color: #999; }
        .meta-value { color: #333; font-weight: 500; }
        .error-item {
            font-size: 0.8em;
            color: #e74c3c;
            background: #fff5f5;
            padding: 4px 8px;
            border-radius: 4px;
            margin: 2px 0;
            border-left: 3px solid #e74c3c;
            word-break: break-all;
        }
        .footer {
            text-align: center;
            padding: 30px;
            color: rgba(255,255,255,0.6);
            font-size: 0.85em;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌐 网页截图扫描报告</h1>
            <p>生成时间: {{.GeneratedAt}} &nbsp;|&nbsp; 总计: {{len .Results}} 张截图</p>
        </div>

        <div class="summary">
            <div class="stat-card">
                <div class="number">{{len .Results}}</div>
                <div class="label">📸 总截图数</div>
            </div>
            <div class="stat-card">
                <div class="number">{{.SuccessCount}}</div>
                <div class="label">✅ 成功</div>
            </div>
            <div class="stat-card">
                <div class="number">{{.FailCount}}</div>
                <div class="label">❌ 失败</div>
            </div>
            <div class="stat-card">
                <div class="number">{{.TechCount}}</div>
                <div class="label">🔍 技术识别</div>
            </div>
        </div>

        {{range .Results}}
        <div class="result-card">
            <div class="result-header">
                <div class="result-title-section">
                    <div class="result-title">{{if .Title}}{{.Title}}{{else}}<em>无标题</em>{{end}}</div>
                    <div class="result-url"><a href="{{.URL}}" target="_blank">{{.URL}}</a></div>
                </div>
                <div class="badge-row">
                    <span class="badge badge-{{.StatusCodeClass}}">{{.ResponseCode}} {{.ResponseReason}}</span>
                    {{if .Protocol}}<span class="badge badge-tls">{{.Protocol}}</span>{{end}}
                    {{if .Cloudflare}}<span class="badge badge-cf">☁️ Cloudflare</span>{{end}}
                    {{if .TLSCipher}}<span class="badge badge-tls">🔒 {{.TLSVersion}}</span>{{end}}
                    {{range .TechBadges}}<span class="badge badge-tech">{{.}}</span>{{end}}
                </div>
            </div>
            <div class="result-body">
                <div class="result-main">
                    {{if .Screenshot}}
                    <img src="{{.Screenshot}}" alt="{{.Title}}" loading="lazy">
                    {{else}}
                    <div class="no-screenshot">
                        {{if .Failed}}❌ 截图失败: {{.FailedReason}}{{else}}📄 无截图{{end}}
                    </div>
                    {{end}}
                </div>
                <div class="result-sidebar">
                    {{if .Technologies}}
                    <div class="sidebar-section">
                        <h4>🔍 检测到的技术</h4>
                        {{range .TechBadges}}<span class="badge badge-tech" style="margin:2px">{{.}}</span>{{end}}
                    </div>
                    {{end}}

                    <div class="sidebar-section">
                        <h4>📋 基本信息</h4>
                        <div class="meta-grid">
                            {{if .FinalURL}}
                            <span class="meta-label">最终URL:</span>
                            <span class="meta-value">{{.FinalURL}}</span>
                            {{end}}
                            {{if .Protocol}}
                            <span class="meta-label">协议:</span>
                            <span class="meta-value">{{.Protocol}}</span>
                            {{end}}
                            {{if .ContentLength}}
                            <span class="meta-label">大小:</span>
                            <span class="meta-value">{{.ContentLength}}</span>
                            {{end}}
                            {{if .ServerHeader}}
                            <span class="meta-label">服务器:</span>
                            <span class="meta-value">{{.ServerHeader}}</span>
                            {{end}}
                            {{if .PoweredByHeader}}
                            <span class="meta-label">Powered-By:</span>
                            <span class="meta-value">{{.PoweredByHeader}}</span>
                            {{end}}
                            <span class="meta-label">探测时间:</span>
                            <span class="meta-value">{{.ProbedAt.Format "2006-01-02 15:04:05"}}</span>
                        </div>
                    </div>

                    {{if .TLSVersion}}
                    <div class="sidebar-section">
                        <h4>🔒 TLS 信息</h4>
                        <div class="meta-grid">
                            <span class="meta-label">协议版本:</span>
                            <span class="meta-value">{{.TLSVersion}}</span>
                            <span class="meta-label">加密套件:</span>
                            <span class="meta-value">{{.TLSCipher}}</span>
                            {{if .TLSIssuer}}
                            <span class="meta-label">签发者:</span>
                            <span class="meta-value">{{.TLSIssuer}}</span>
                            {{end}}
                            {{if .TLSSANs}}
                            <span class="meta-label">SAN:</span>
                            <span class="meta-value" style="font-size:0.8em">{{.TLSSANs}}</span>
                            {{end}}
                        </div>
                    </div>
                    {{end}}

                    {{if .ConsoleErrors}}
                    <div class="sidebar-section">
                        <h4>⚠️ 控制台错误 ({{len .ConsoleErrors}})</h4>
                        {{range .ConsoleErrors}}
                        <div class="error-item">{{.}}</div>
                        {{end}}
                    </div>
                    {{end}}

                    {{if .Failed}}
                    <div class="sidebar-section">
                        <h4>❌ 失败原因</h4>
                        <div class="error-item">{{.FailedReason}}</div>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>
        {{end}}

        <div class="footer">
            Generated by snir 🚀 Security Intelligence Reconnaissance
        </div>
    </div>
</body>
</html>`

// GenerateHTML 生成HTML报告
func GenerateHTML(options HTMLOptions) error {
	if options.InputFile == "" {
		return fmt.Errorf("输入文件不能为空")
	}
	if !islazy.FileExists(options.InputFile) {
		return fmt.Errorf("输入文件不存在: %s", options.InputFile)
	}

	log.Info("读取结果文件", "file", options.InputFile)
	results, err := ReadJSONLResults(options.InputFile)
	if err != nil {
		return fmt.Errorf("读取结果文件失败: %v", err)
	}
	if len(results) == 0 {
		return fmt.Errorf("结果文件中没有有效的记录")
	}
	log.Info("读取到结果记录", "count", len(results))

	// 准备报告数据
	var successCount, failCount int
	var allTechNames map[string]bool // track unique technologies across all results
	allTechNames = make(map[string]bool)
	reportResults := make([]ReportResult, 0, len(results))

	for _, result := range results {
		// 解析结果详情
		if result.Failed {
			failCount++
		} else {
			successCount++
		}

		statusClass := getStatusClass(result.ResponseCode)

		// 处理截图路径
		screenshotPath := result.Filename
		if screenshotPath != "" {
			if filepath.IsAbs(screenshotPath) {
				if relPath, err := filepath.Rel(filepath.Dir(options.OutputPath), screenshotPath); err == nil {
					screenshotPath = relPath
				}
			}
		}

		// 收集技术名称
		var techNames []string
		var techBadges []string
		for _, tech := range result.Technologies {
			techNames = append(techNames, tech.Name)
			if tech.Version != "" {
				techBadges = append(techBadges, tech.Name+" "+tech.Version)
			} else {
				techBadges = append(techBadges, tech.Name)
			}
			allTechNames[tech.Name] = true
		}

		// 收集控制台错误
		var consoleErrors []string
		for _, c := range result.Console {
			if c.Level == "error" {
				consoleErrors = append(consoleErrors, c.Message)
			}
		}

		// 提取响应头
		var serverHeader, poweredByHeader string
		var cloudflare bool
		for _, h := range result.Headers {
			switch strings.ToLower(h.Name) {
			case "server":
				serverHeader = h.Value
				if strings.Contains(strings.ToLower(h.Value), "cloudflare") {
					cloudflare = true
				}
			case "x-powered-by":
				poweredByHeader = h.Value
			case "cf-ray":
				cloudflare = true
			}
		}

		// 内容长度格式
		var contentLengthStr string
		if result.ContentLength > 0 {
			const kb = 1024
			if result.ContentLength < kb {
				contentLengthStr = fmt.Sprintf("%d B", result.ContentLength)
			} else if result.ContentLength < kb*kb {
				contentLengthStr = fmt.Sprintf("%.1f KB", float64(result.ContentLength)/kb)
			} else {
				contentLengthStr = fmt.Sprintf("%.1f MB", float64(result.ContentLength)/(kb*kb))
			}
		}

		reportResults = append(reportResults, ReportResult{
			URL:             result.URL,
			Title:           result.Title,
			Screenshot:      screenshotPath,
			ResponseCode:    result.ResponseCode,
			StatusCodeClass: statusClass,
			ProbedAt:        result.ProbedAt,
			FinalURL:        result.FinalURL,
			ResponseReason:  result.ResponseReason,
			Protocol:        result.Protocol,
			TLSVersion:      result.TLS.Version,
			TLSCipher:       result.TLS.CipherSuite,
			TLSIssuer:       result.TLS.Issuer,
			TLSSANs:         result.TLS.SANs,
			IsPDF:           result.IsPDF,
			Failed:          result.Failed,
			FailedReason:    result.FailedReason,
			Technologies:    techNames,
			TechBadges:      techBadges,
			ConsoleErrors:   consoleErrors,
			ContentLength:   contentLengthStr,
			ServerHeader:    serverHeader,
			PoweredByHeader: poweredByHeader,
			Cloudflare:      cloudflare,
		})
	}

	reportData := ReportData{
		GeneratedAt:  time.Now().Format("2006-01-02 15:04:05"),
		Results:      reportResults,
		SuccessCount: successCount,
		FailCount:    failCount,
		TechCount:    len(allTechNames),
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(options.OutputPath)
	if _, err := islazy.CreateDir(outputDir); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	outputFile, err := os.Create(options.OutputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outputFile.Close()

	tmpl, err := template.New("report").Parse(RichHTMLTemplate)
	if err != nil {
		return fmt.Errorf("解析报告模板失败: %v", err)
	}
	if err := tmpl.Execute(outputFile, reportData); err != nil {
		return fmt.Errorf("生成报告失败: %v", err)
	}

	log.Info("HTML报告生成成功", "path", options.OutputPath)
	return nil
}

// getStatusClass returns CSS class for status code
func getStatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "0xx"
	}
}

// ReadJSONLResults 从JSONL文件读取结果
func ReadJSONLResults(filePath string) ([]*models.Result, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件内容
	var results []*models.Result
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 解析JSON
		var result models.Result
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			log.Error("解析JSON行失败", "error", err, "line", line)
			continue
		}

		results = append(results, &result)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
