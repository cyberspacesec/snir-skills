package report

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// ServerOptions 包含Web服务器选项
type ServerOptions struct {
	Host           string
	Port           int
	ScreenshotPath string
	ReportPath     string
}

// Server 表示Web服务器
type Server struct {
	Options ServerOptions
}

// NewServer 创建一个新的Web服务器
func NewServer(options ServerOptions) *Server {
	return &Server{
		Options: options,
	}
}

// Run 启动Web服务器
func (s *Server) Run() error {
	// 确保截图目录存在
	screenshotPath, err := islazy.CreateDir(s.Options.ScreenshotPath)
	if err != nil {
		return fmt.Errorf("创建截图目录失败: %v", err)
	}

	// 确保报告目录存在
	reportPath, err := islazy.CreateDir(s.Options.ReportPath)
	if err != nil {
		return fmt.Errorf("创建报告目录失败: %v", err)
	}

	// 设置HTTP处理函数
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.serveIndex(w, r, screenshotPath, reportPath)
	})

	// 设置静态文件服务
	http.Handle("/screenshots/", http.StripPrefix("/screenshots/", http.FileServer(http.Dir(screenshotPath))))
	http.Handle("/reports/", http.StripPrefix("/reports/", http.FileServer(http.Dir(reportPath))))

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", s.Options.Host, s.Options.Port)
	log.Info("启动Web服务器", "address", addr)
	log.Info(fmt.Sprintf("请访问 http://%s 查看结果", addr))

	return http.ListenAndServe(addr, nil)
}

// serveIndex 处理首页请求
func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request, screenshotPath, reportPath string) {
	// 如果请求的不是根路径，返回404
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// 获取截图文件列表
	screenshots, err := getFiles(screenshotPath, ".png", ".jpeg", ".jpg")
	if err != nil {
		http.Error(w, "获取截图列表失败", http.StatusInternalServerError)
		return
	}

	// 获取报告文件列表
	reports, err := getFiles(reportPath, ".json", ".csv", ".html")
	if err != nil {
		http.Error(w, "获取报告列表失败", http.StatusInternalServerError)
		return
	}

	// 生成HTML页面
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!DOCTYPE html>\n")
	fmt.Fprintf(w, "<html>\n")
	fmt.Fprintf(w, "<head>\n")
	fmt.Fprintf(w, "  <meta charset=\"utf-8\">\n")
	fmt.Fprintf(w, "  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(w, "  <title>Go Web Screenshot - 结果查看器</title>\n")
	fmt.Fprintf(w, "  <style>\n")
	fmt.Fprintf(w, "    body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }\n")
	fmt.Fprintf(w, "    h1 { color: #333; }\n")
	fmt.Fprintf(w, "    .container { max-width: 1200px; margin: 0 auto; }\n")
	fmt.Fprintf(w, "    .screenshots { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 20px; }\n")
	fmt.Fprintf(w, "    .screenshot { border: 1px solid #ddd; padding: 10px; border-radius: 5px; }\n")
	fmt.Fprintf(w, "    .screenshot img { width: 100%%; height: auto; }\n")
	fmt.Fprintf(w, "    .screenshot-info { margin-top: 10px; }\n")
	fmt.Fprintf(w, "    .reports { margin-top: 30px; }\n")
	fmt.Fprintf(w, "    .report-item { margin-bottom: 10px; }\n")
	fmt.Fprintf(w, "    .tabs { display: flex; margin-bottom: 20px; }\n")
	fmt.Fprintf(w, "    .tab { padding: 10px 20px; cursor: pointer; background: #f1f1f1; margin-right: 5px; }\n")
	fmt.Fprintf(w, "    .tab.active { background: #007bff; color: white; }\n")
	fmt.Fprintf(w, "    .tab-content { display: none; }\n")
	fmt.Fprintf(w, "    .tab-content.active { display: block; }\n")
	fmt.Fprintf(w, "  </style>\n")
	fmt.Fprintf(w, "</head>\n")
	fmt.Fprintf(w, "<body>\n")
	fmt.Fprintf(w, "  <div class=\"container\">\n")
	fmt.Fprintf(w, "    <h1>Go Web Screenshot - 结果查看器</h1>\n")
	fmt.Fprintf(w, "    <div class=\"tabs\">\n")
	fmt.Fprintf(w, "      <div class=\"tab active\" data-tab=\"screenshots\">截图 (%d)</div>\n", len(screenshots))
	fmt.Fprintf(w, "      <div class=\"tab\" data-tab=\"reports\">报告 (%d)</div>\n", len(reports))
	fmt.Fprintf(w, "    </div>\n")

	// 截图标签页内容
	fmt.Fprintf(w, "    <div id=\"screenshots\" class=\"tab-content active\">\n")
	if len(screenshots) > 0 {
		fmt.Fprintf(w, "      <div class=\"screenshots\">\n")
		for _, screenshot := range screenshots {
			fileName := filepath.Base(screenshot)
			fileInfo, err := os.Stat(screenshot)
			modTime := ""
			if err == nil {
				modTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
			}

			// 从文件名中提取URL（假设文件名格式为URL_时间戳.扩展名）
			urlPart := strings.Split(fileName, "_")[0]
			urlPart = strings.ReplaceAll(urlPart, "_", "/")

			fmt.Fprintf(w, "        <div class=\"screenshot\">\n")
			fmt.Fprintf(w, "          <img src=\"/screenshots/%s\" alt=\"%s\">\n", fileName, fileName)
			fmt.Fprintf(w, "          <div class=\"screenshot-info\">\n")
			fmt.Fprintf(w, "            <div><strong>文件:</strong> %s</div>\n", fileName)
			fmt.Fprintf(w, "            <div><strong>URL:</strong> %s</div>\n", urlPart)
			fmt.Fprintf(w, "            <div><strong>时间:</strong> %s</div>\n", modTime)
			fmt.Fprintf(w, "          </div>\n")
			fmt.Fprintf(w, "        </div>\n")
		}
		fmt.Fprintf(w, "      </div>\n")
	} else {
		fmt.Fprintf(w, "      <p>没有找到截图文件。请先运行扫描命令生成截图。</p>\n")
	}
	fmt.Fprintf(w, "    </div>\n")

	// 报告标签页内容
	fmt.Fprintf(w, "    <div id=\"reports\" class=\"tab-content\">\n")
	if len(reports) > 0 {
		fmt.Fprintf(w, "      <div class=\"reports\">\n")
		for _, report := range reports {
			fileName := filepath.Base(report)
			fileInfo, err := os.Stat(report)
			modTime := ""
			if err == nil {
				modTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
			}

			fmt.Fprintf(w, "        <div class=\"report-item\">\n")
			fmt.Fprintf(w, "          <a href=\"/reports/%s\" target=\"_blank\">%s</a> - %s\n", fileName, fileName, modTime)
			fmt.Fprintf(w, "        </div>\n")
		}
		fmt.Fprintf(w, "      </div>\n")
	} else {
		fmt.Fprintf(w, "      <p>没有找到报告文件。请先运行扫描命令生成报告。</p>\n")
	}
	fmt.Fprintf(w, "    </div>\n")

	// JavaScript代码
	fmt.Fprintf(w, "    <script>\n")
	fmt.Fprintf(w, "      document.addEventListener('DOMContentLoaded', function() {\n")
	fmt.Fprintf(w, "        const tabs = document.querySelectorAll('.tab');\n")
	fmt.Fprintf(w, "        tabs.forEach(tab => {\n")
	fmt.Fprintf(w, "          tab.addEventListener('click', function() {\n")
	fmt.Fprintf(w, "            const tabId = this.getAttribute('data-tab');\n")
	fmt.Fprintf(w, "            \n")
	fmt.Fprintf(w, "            // 移除所有标签页的active类\n")
	fmt.Fprintf(w, "            tabs.forEach(t => t.classList.remove('active'));\n")
	fmt.Fprintf(w, "            document.querySelectorAll('.tab-content').forEach(content => {\n")
	fmt.Fprintf(w, "              content.classList.remove('active');\n")
	fmt.Fprintf(w, "            });\n")
	fmt.Fprintf(w, "            \n")
	fmt.Fprintf(w, "            // 添加active类到当前标签页\n")
	fmt.Fprintf(w, "            this.classList.add('active');\n")
	fmt.Fprintf(w, "            document.getElementById(tabId).classList.add('active');\n")
	fmt.Fprintf(w, "          });\n")
	fmt.Fprintf(w, "        });\n")
	fmt.Fprintf(w, "      });\n")
	fmt.Fprintf(w, "    </script>\n")

	fmt.Fprintf(w, "  </div>\n")
	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
}

// getFiles 获取指定目录下的指定扩展名的文件列表
func getFiles(dir string, extensions ...string) ([]string, error) {
	var files []string

	// 检查目录是否存在
	if !islazy.DirExists(dir) {
		return files, nil
	}

	// 遍历目录
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件
		if !info.IsDir() {
			// 检查文件扩展名
			ext := strings.ToLower(filepath.Ext(path))
			for _, validExt := range extensions {
				if ext == validExt {
					files = append(files, path)
					break
				}
			}
		}
		return nil
	})

	return files, err
}
