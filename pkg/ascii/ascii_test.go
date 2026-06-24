package ascii

import (
	"strings"
	"testing"
)

func TestLogo(t *testing.T) {
	logo := Logo()

	// 验证输出不为空
	if logo == "" {
		t.Error("Logo() 返回了空字符串")
	}

	// 检查是否包含 go-snir 文本（小写也算）
	if !strings.Contains(strings.ToLower(logo), "go-snir") &&
		!strings.Contains(strings.ToLower(logo), "snir") {
		t.Error("Logo() 应该包含 'Snir' 或 'go-snir' 文本")
	}

	// 检查是否包含版本信息
	if !strings.Contains(logo, "版本") {
		t.Error("Logo() 应该包含版本信息")
	}
}

func TestVersionInfo(t *testing.T) {
	info := VersionInfo()

	// 验证输出不为空
	if info == "" {
		t.Error("VersionInfo() 返回了空字符串")
	}

	// 检查是否包含必要的版本信息
	requiredFields := []string{"版本", "提交", "构建时间", "项目地址"}
	for _, field := range requiredFields {
		if !strings.Contains(info, field) {
			t.Errorf("VersionInfo() 应该包含 '%s' 字段", field)
		}
	}
}

func TestMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
		wantErr  bool
	}{
		{
			name:     "基本标题",
			markdown: "# Test Heading",
			wantErr:  false,
		},
		{
			name:     "链接文本",
			markdown: "[Link Text](https://example.com)",
			wantErr:  false,
		},
		{
			name:     "空字符串",
			markdown: "",
			wantErr:  false,
		},
		{
			name:     "图片",
			markdown: "![alt text](https://example.com/image.png)",
			wantErr:  false,
		},
		{
			name:     "水平分割线",
			markdown: "---",
			wantErr:  false,
		},
		{
			name:     "代码块",
			markdown: "```go\nfunc main() {}\n```",
			wantErr:  false,
		},
		{
			name:     "强调文本",
			markdown: "**bold** *italic*",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Markdown(tt.markdown)

			// 验证输出不为空（除非输入为空）
			if result == "" && tt.markdown != "" {
				t.Error("Markdown() 不应返回空字符串")
			}

			// 检查是否包含错误消息
			if tt.wantErr && !strings.Contains(result, "错误") {
				t.Error("Markdown() 应该返回错误信息")
			}

			// 对于链接文本情况，验证原始内容存在的更灵活方式
			if tt.name == "链接文本" {
				// 链接文本的 Markdown 转换后可能改变了格式，但应该包含 "Link Text" 和 "example.com"
				if !strings.Contains(result, "Link Text") || !strings.Contains(result, "example.com") {
					t.Errorf("Markdown() 输出应该包含链接文本的关键部分: %s", tt.markdown)
				}
				return
			}

			// 其他情况检查原始内容
			if !tt.wantErr && tt.markdown != "" && !strings.Contains(result, strings.TrimSpace(tt.markdown)) {
				// 特别处理: glamour 可能会添加格式，所以检查是否包含核心内容
				coreContent := tt.markdown
				switch tt.name {
				case "基本标题":
					coreContent = "Test Heading"
				case "图片":
					coreContent = "image.png"
				case "代码块":
					coreContent = "func main()"
				}

				if !strings.Contains(result, coreContent) {
					t.Errorf("Markdown() 输出应该包含原始内容的核心部分: %s", coreContent)
				}
			}
		})
	}
}
