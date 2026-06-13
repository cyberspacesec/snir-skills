package runner

// Options contains all the configuration options for the runner
type Options struct {
	// API server options
	API struct {
		Host          string // API服务监听地址
		Port          int    // API服务监听端口
		APIKey        string // API密钥，用于API鉴权
		MaxConcurrent int    // 最大并发请求数
		QueueSize     int    // 请求队列大小
	}
	// Logging options
	Logging struct {
		Debug   bool // 是否启用调试日志
		Silence bool // 是否静默日志输出
	}

	// Chrome options
	Chrome struct {
		Path             string // Chrome可执行文件路径
		UserAgent        string // 自定义User-Agent
		Proxy            string // 代理服务器地址
		Timeout          int    // 页面加载超时时间（秒）
		Delay            int    // 截图前等待时间（秒）
		WindowX          int    // 窗口宽度
		WindowY          int    // 窗口高度
		WSS              string // WebSocket服务器地址
		Headless         bool   // 是否使用无头模式
		IgnoreCertErrors bool   // 是否忽略证书错误

		// 代理池和轮换
		ProxyList     []string      // 代理列表（多个代理地址轮换）
		ProxyFile     string        // 代理文件路径（每行一个代理，支持热加载）
		ProxyURL      string        // 代理 API URL（动态代理服务，每次请求获取新代理）
		ProxyStrategy ProxyStrategy // 代理轮换策略：round-robin / random / sequential

		// 高级浏览器控制
		AcceptLanguage  string            // 接受的语言
		Platform        string            // 平台
		Vendor          string            // 浏览器供应商
		Plugins         []string          // 插件列表
		WebGLVendor     string            // WebGL供应商
		WebGLRenderer   string            // WebGL渲染器
		CustomHeaders   map[string]string // 自定义HTTP头
		DisableWebRTC   bool              // 是否禁用WebRTC
		SpoofScreenSize bool              // 是否欺骗屏幕尺寸
		ScreenWidth     int               // 屏幕宽度
		ScreenHeight    int               // 屏幕高度
	}

	// Scan options
	Scan struct {
		Driver             string   // 使用的驱动（chromedp）
		Threads            int      // 并发线程数
		ScreenshotPath     string   // 截图保存路径
		ScreenshotFormat   string   // 截图格式（jpeg或png）
		ScreenshotQuality  int      // 截图质量（仅对JPEG有效）
		ScreenshotSkipSave bool     // 是否跳过保存截图
		SaveHTML           bool     // 是否保存HTML内容
		SaveHeaders        bool     // 是否保存HTTP头
		SaveConsole        bool     // 是否保存控制台日志
		SaveCookies        bool     // 是否保存Cookie
		SaveNetwork        bool     // 是否保存网络请求日志
		HTTP               bool     // 是否使用HTTP协议
		HTTPS              bool     // 是否使用HTTPS协议
		Ports              []int    // 扫描的端口列表
		Timeout            int      // 扫描超时时间（秒）
		MaxRetries         int      // 最大重试次数
		JavaScript         string   // 要在页面上执行的JavaScript代码
		JavaScriptFile     string   // 包含JavaScript代码的文件路径
		FilePath           string   // URL文件路径，用于批量扫描
		EnableBlacklist    bool     // 是否启用URL黑名单
		DefaultBlacklist   bool     // 是否使用默认黑名单
		BlacklistPatterns  []string // 自定义黑名单规则（支持CIDR和正则表达式）
		BlacklistFile      string   // 黑名单文件路径

		// 高级功能
		RunJSBefore     bool                // 在页面加载前执行JS
		RunJSAfter      bool                // 在页面加载后执行JS
		Cookies         []CustomCookie      // 自定义Cookie
		CookiesFile      string              // Cookie持久化文件路径
		Selector        string              // CSS选择器，用于元素截图
		XPath           string              // XPath，用于元素截图
		CaptureFullPage bool                // 是否捕获整个页面
		Actions         []InteractionAction // 交互操作列表
		Form            Form                // 表单配置
	}

	// Writer options
	Writer struct {
		Db        bool   // 是否写入数据库
		DbURI     string // 数据库连接URI
		DbDebug   bool   // 是否启用数据库调试
		Jsonl     bool   // 是否写入JSONL文件
		JsonlFile string // JSONL文件路径
		Csv       bool   // 是否写入CSV文件
		CsvFile   string // CSV文件路径
		Stdout    bool   // 是否输出到标准输出
	}

	// Database options
	DB struct {
		Enable bool   // 是否启用数据库
		Path   string // 数据库文件路径
	}

	// Report options
	Report struct {
		OutputPath string // 报告输出路径
		Format     string // 报告格式 (html, json, csv)
		Port       int    // Web服务器端口
		Host       string // Web服务器主机地址
		InputFile  string // 输入文件路径，用于生成报告
	}
}

// CustomCookie 表示自定义Cookie
type CustomCookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Secure   bool
	HttpOnly bool
}

// InteractionAction 表示交互操作
type InteractionAction struct {
	Type        string // click, scroll, type, wait, hover
	Selector    string // CSS选择器
	XPath       string // XPath
	Value       string // 用于输入的值或滚动距离
	WaitTime    int    // 等待时间(毫秒)
	WaitVisible bool   // 等待元素可见
}

// FormField 表示表单字段
type FormField struct {
	Selector string // CSS选择器
	XPath    string // XPath
	Value    string // 填充的值
	Type     string // input, select, checkbox, radio
}

// Form 表示表单配置
type Form struct {
	Fields          []FormField // 表单字段
	SubmitSelector  string      // 提交按钮选择器
	SubmitXPath     string      // 提交按钮XPath
	WaitAfterSubmit int         // 提交后等待时间(毫秒)
}
