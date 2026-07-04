import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

// snir 文档站配置
// 导航与侧边栏覆盖项目的每一个模块与功能点
export default withMermaid(
  defineConfig({
  // 部署在 GitHub Pages 子路径 /snir-skills/ 下，必须设置 base
  // 否则 CSS/JS 等静态资源会按根路径 / 加载而 404，导致布局错乱
  base: '/snir-skills/',
  lang: 'zh-CN',
  title: 'snir',
  description: 'AI 原生的网页截图与情报采集工具 — 让 AI 代理与自动化系统拥有浏览器级取证能力',
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/snir-skills/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#3aa676' }],
    ['meta', { name: 'og:title', content: 'snir — AI 原生网页情报采集' }],
    ['meta', { name: 'og:description', content: '基于 Chrome DevTools Protocol 的截图与 Web 情报子系统，支持 CLI、HTTP API、Go SDK 与共享 CDP Provider。' }]
  ],

  lastUpdated: true,
  cleanUrls: true,

  markdown: {
    lineNumbers: true,
    theme: { light: 'github-light', dark: 'vitesse-dark' }
  },

  themeConfig: {
    siteTitle: 'snir 文档',
    logo: '/logo.svg',

    nav: [
      { text: '🏠 首页', link: '/' },
      { text: '📖 指南', link: '/guide/what-is-snir' },
      { text: '🖥️ CLI', link: '/cli/overview' },
      { text: '🧩 SDK', link: '/sdk/overview' },
      { text: '🌐 HTTP API', link: '/api/overview' },
      { text: '🔬 内部模块', link: '/internals/overview' },
      { text: '🚀 进阶', link: '/advanced/proxy' },
      {
        text: '📦 更多',
        items: [
          { text: '📚 参考手册', link: '/reference/result-schema' },
          { text: '🤝 社区与贡献', link: '/community/contributing' },
          { text: '🎮 GitHub', link: 'https://github.com/cyberspacesec/snir-skills' }
        ]
      }
    ],

    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
          modal: {
            noResultsText: '无法找到相关结果',
            resetButtonTitle: '清除查询条件',
            footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' }
          }
        }
      }
    },

    sidebar: {
      '/guide/': [
        {
          text: '🚀 入门',
          collapsed: false,
          items: [
            { text: 'snir 是什么', link: '/guide/what-is-snir' },
            { text: '解决什么问题', link: '/guide/problem-it-solves' },
            { text: '核心概念', link: '/guide/core-concepts' },
            { text: '快速开始', link: '/guide/quick-start' },
            { text: '安装', link: '/guide/installation' },
            { text: '五分钟教程', link: '/guide/five-minutes' }
          ]
        },
        {
          text: '🧭 架构',
          collapsed: false,
          items: [
            { text: '整体架构', link: '/guide/architecture' },
            { text: '集成模式', link: '/guide/integration-modes' },
            { text: '数据流', link: '/guide/data-flow' },
            { text: 'Skill Bundle', link: '/guide/skill-bundle' }
          ]
        },
        {
          text: '🎯 场景',
          collapsed: false,
          items: [
            { text: 'AI 代理集成', link: '/guide/ai-agent' },
            { text: '安全侦察', link: '/guide/security-recon' },
            { text: '自动化巡检', link: '/guide/automation' },
            { text: '内容监控', link: '/guide/monitoring' }
          ]
        }
      ],

      '/cli/': [
        {
          text: '🖥️ 命令行',
          collapsed: false,
          items: [
            { text: 'CLI 总览', link: '/cli/overview' },
            { text: '全局选项', link: '/cli/global-options' },
            { text: '退出码', link: '/cli/exit-codes' }
          ]
        },
        {
          text: '📸 scan 命令族',
          collapsed: false,
          items: [
            { text: 'scan 概览', link: '/cli/scan' },
            { text: 'scan 单URL', link: '/cli/scan-single' },
            { text: 'scan file 批量', link: '/cli/scan-file' },
            { text: 'scan cidr 网段', link: '/cli/scan-cidr' },
            { text: '截图选项', link: '/cli/scan-screenshot' },
            { text: '证据选项', link: '/cli/scan-evidence' },
            { text: 'Chrome 选项', link: '/cli/scan-chrome' },
            { text: '代理选项', link: '/cli/scan-proxy' },
            { text: 'Cookie 选项', link: '/cli/scan-cookie' },
            { text: '设备模拟', link: '/cli/scan-device' },
            { text: 'JS 注入', link: '/cli/scan-js' },
            { text: '输出选项', link: '/cli/scan-output' },
            { text: '数据库选项', link: '/cli/scan-db' },
            { text: '黑名单', link: '/cli/scan-blacklist' },
            { text: '端口展开', link: '/cli/scan-ports' }
          ]
        },
        {
          text: '🌐 api 命令',
          collapsed: false,
          items: [
            { text: 'api 总览', link: '/cli/api' },
            { text: 'api 鉴权', link: '/cli/api-auth' }
          ]
        },
        {
          text: '🔌 provider 命令',
          collapsed: false,
          items: [{ text: 'provider 总览', link: '/cli/provider' }]
        },
        {
          text: '📊 report 命令族',
          collapsed: false,
          items: [
            { text: 'report 概览', link: '/cli/report' },
            { text: 'report html', link: '/cli/report-html' },
            { text: 'report convert', link: '/cli/report-convert' },
            { text: 'report merge', link: '/cli/report-merge' }
          ]
        },
        {
          text: '🌐 webserve 命令',
          collapsed: false,
          items: [{ text: 'webserve 总览', link: '/cli/webserve' }]
        },
        {
          text: 'ℹ️ version 命令',
          collapsed: false,
          items: [{ text: 'version 总览', link: '/cli/version' }]
        }
      ],

      '/sdk/': [
        {
          text: '🧩 Go SDK',
          collapsed: false,
          items: [
            { text: 'SDK 总览', link: '/sdk/overview' },
            { text: '安装与依赖', link: '/sdk/installation' },
            { text: 'Client 客户端', link: '/sdk/client' },
            { text: 'ClientOptions', link: '/sdk/client-options' },
            { text: '选项构建器', link: '/sdk/builders' },
            { text: '结果与证据', link: '/sdk/result' },
            { text: '共享池', link: '/sdk/shared' },
            { text: '自动连接', link: '/sdk/autoconnect' },
            { text: '目标展开', link: '/sdk/targets' },
            { text: '批量采集', link: '/sdk/batch' }
          ]
        },
        {
          text: '🛠️ 构建器参考',
          collapsed: false,
          items: [
            { text: '截图构建器', link: '/sdk/builder-screenshot' },
            { text: '视口与设备', link: '/sdk/builder-viewport' },
            { text: '代理构建器', link: '/sdk/builder-proxy' },
            { text: 'Cookie 构建', link: '/sdk/builder-cookie' },
            { text: '指纹构建', link: '/sdk/builder-fingerprint' },
            { text: 'JS 与交互', link: '/sdk/builder-js' },
            { text: '表单构建', link: '/sdk/builder-form' },
            { text: '黑名单构建', link: '/sdk/builder-blacklist' },
            { text: '端口与协议', link: '/sdk/builder-ports' }
          ]
        }
      ],

      '/api/': [
        {
          text: '🌐 HTTP API',
          collapsed: false,
          items: [
            { text: 'API 总览', link: '/api/overview' },
            { text: 'Server 与选项', link: '/api/server' },
            { text: '请求类型', link: '/api/request-types' },
            { text: '鉴权', link: '/api/auth' },
            { text: '并发限流', link: '/api/concurrency' },
            { text: '中间件', link: '/api/middleware' },
            { text: '辅助函数', link: '/api/helpers' },
            { text: '响应格式', link: '/api/response' }
          ]
        },
        {
          text: '📡 端点',
          collapsed: false,
          items: [
            { text: 'POST /screenshot', link: '/api/endpoint-screenshot' },
            { text: 'POST /batch', link: '/api/endpoint-batch' },
            { text: 'GET /health', link: '/api/endpoint-health' },
            { text: 'GET /stats', link: '/api/endpoint-stats' }
          ]
        }
      ],

      '/internals/': [
        {
          text: '🔬 内部模块',
          collapsed: false,
          items: [
            { text: '内部模块总览', link: '/internals/overview' },
            { text: 'pkg/runner', link: '/internals/runner' },
            { text: 'pkg/scan', link: '/internals/scan' },
            { text: 'pkg/provider', link: '/internals/provider' },
            { text: 'pkg/sdk', link: '/internals/sdk' },
            { text: 'pkg/api', link: '/internals/api' },
            { text: 'pkg/models', link: '/internals/models' },
            { text: 'pkg/phash', link: '/internals/phash' },
            { text: 'pkg/techdetect', link: '/internals/techdetect' },
            { text: 'pkg/database', link: '/internals/database' },
            { text: 'pkg/report', link: '/internals/report' },
            { text: 'pkg/log', link: '/internals/log' },
            { text: 'pkg/islazy', link: '/internals/islazy' },
            { text: 'pkg/ascii', link: '/internals/ascii' }
          ]
        },
        {
          text: '⚙️ runner 子模块',
          collapsed: true,
          items: [
            { text: 'Driver 接口', link: '/internals/runner-driver' },
            { text: 'ChromeDP', link: '/internals/runner-chromedp' },
            { text: 'DriverPool', link: '/internals/runner-pool' },
            { text: 'PoolDriver', link: '/internals/runner-pool-driver' },
            { text: '共享池单例', link: '/internals/runner-pool-singleton' },
            { text: 'Pool 事件', link: '/internals/runner-pool-events' },
            { text: 'Runner 核心', link: '/internals/runner-core' },
            { text: 'Options', link: '/internals/runner-options' },
            { text: 'Writer', link: '/internals/runner-writer' },
            { text: 'CookieJar', link: '/internals/runner-cookie-jar' },
            { text: 'Netscape Cookie', link: '/internals/runner-cookie-netscape' },
            { text: 'Cookie 工具', link: '/internals/runner-cookie-util' },
            { text: 'Device Presets', link: '/internals/runner-device' },
            { text: 'Discovery', link: '/internals/runner-discovery' },
            { text: 'Proxy', link: '/internals/runner-proxy' },
            { text: 'Blacklist', link: '/internals/runner-blacklist' }
          ]
        }
      ],

      '/advanced/': [
        {
          text: '🚀 进阶',
          collapsed: false,
          items: [
            { text: '代理与轮换', link: '/advanced/proxy' },
            { text: '设备模拟', link: '/advanced/device' },
            { text: '浏览器指纹', link: '/advanced/fingerprint' },
            { text: 'Cookie 管理', link: '/advanced/cookie' },
            { text: 'JS 注入', link: '/advanced/js-injection' },
            { text: '表单与交互', link: '/advanced/forms' },
            { text: '黑名单', link: '/advanced/blacklist' },
            { text: '证据采集', link: '/advanced/evidence' },
            { text: '感知哈希', link: '/advanced/perceptual-hash' },
            { text: '技术检测', link: '/advanced/tech-detection' },
            { text: '远程 Chrome', link: '/advanced/remote-chrome' },
            { text: '并发与池', link: '/advanced/concurrency' },
            { text: '输出格式', link: '/advanced/output-formats' },
            { text: '报告生成', link: '/advanced/reports' },
            { text: '数据库存储', link: '/advanced/database' }
          ]
        },
        {
          text: '🛠️ 运维',
          collapsed: false,
          items: [
            { text: 'Docker 部署', link: '/advanced/docker' },
            { text: 'CI/CD 集成', link: '/advanced/cicd' },
            { text: '性能调优', link: '/advanced/performance' },
            { text: '故障排查', link: '/advanced/troubleshooting' },
            { text: '安全注意', link: '/advanced/security' }
          ]
        }
      ],

      '/reference/': [
        {
          text: '📚 参考手册',
          collapsed: false,
          items: [
            { text: 'Result Schema', link: '/reference/result-schema' },
            { text: '字段字典', link: '/reference/fields' },
            { text: 'CLI 标志全表', link: '/reference/cli-flags' },
            { text: '错误码', link: '/reference/error-codes' },
            { text: '配置文件', link: '/reference/config' },
            { text: '更新日志', link: '/reference/changelog' },
            { text: 'FAQ', link: '/reference/faq' }
          ]
        }
      ],

      '/community/': [
        {
          text: '🤝 社区',
          collapsed: false,
          items: [
            { text: '贡献指南', link: '/community/contributing' },
            { text: '文档贡献', link: '/community/docs' },
            { text: '行为准则', link: '/community/code-of-conduct' },
            { text: '路线图', link: '/community/roadmap' },
            { text: '致谢', link: '/community/credits' }
          ]
        }
      ]
    },

    footer: {
      message: '基于 MIT 许可发布',
      copyright: 'Copyright © 2026 cyberspacesec'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/cyberspacesec/snir-skills' }
    ],

    outline: { level: [2, 3], label: '本页导航' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    lastUpdatedText: '最后更新',
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
    darkModeSwitchLabel: '主题',
    lightModeSwitchTitle: '切换到浅色主题',
    darkModeSwitchTitle: '切换到深色主题'
  }
}),
  {
    // mermaid 客户端渲染：把 ```mermaid 代码块渲染为流程图/时序图等
    // 主题随站点明暗切换
    mermaid: {
      theme: 'default'
    }
  }
)
