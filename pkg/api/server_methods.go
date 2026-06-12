package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
	"github.com/cyberspacesec/go-snir/pkg/runner"
	"github.com/gorilla/mux"
)

// NewServer 创建一个新的API服务器
func NewServer(options ServerOptions) *Server {
	// 初始化并发限制器
	InitConcurrencyLimiter(options.MaxConcurrentRequests, options.RequestQueueSize)

	return &Server{
		Options: options,
		Router:  mux.NewRouter(),
	}
}

// InitPool 初始化浏览器连接池
// 必须在服务器启动前调用，使用共享的 Chrome 进程处理所有截图请求
func (s *Server) InitPool(opts *runner.Options) error {
	pool, err := runner.NewDriverPool(opts, s.Options.MaxConcurrentRequests)
	if err != nil {
		return fmt.Errorf("初始化浏览器连接池失败: %v", err)
	}
	s.pool = pool
	log.Info("API服务器浏览器连接池已初始化", "max_concurrent", s.Options.MaxConcurrentRequests)
	return nil
}

// ClosePool 关闭浏览器连接池
func (s *Server) ClosePool() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// GetBlacklist 获取URL黑名单实例
func (s *Server) GetBlacklist(opts *runner.Options) (*runner.URLBlacklist, error) {
	return runner.NewURLBlacklist(opts)
}

// ProcessScreenshot 处理单个URL的截图请求
// 优先使用连接池，若池未初始化则回退到单次创建模式
func (s *Server) ProcessScreenshot(req ScreenshotRequest, opts runner.Options) (*models.Result, error) {
	// 优先使用连接池
	if s.pool != nil {
		result, err := s.pool.Screenshot(req.URL, &opts)
		if err != nil {
			return nil, fmt.Errorf("截图失败: %v", err)
		}
		if result.Failed {
			return nil, fmt.Errorf(result.FailedReason)
		}
		return result, nil
	}

	// 回退：连接池未初始化时使用单次模式
	driver, err := runner.NewChromeDP(&opts)
	if err != nil {
		return nil, fmt.Errorf("创建浏览器驱动失败: %v", err)
	}
	defer driver.Close()

	runnerInstance, err := runner.NewRunner(log.GetLogger(), driver, opts, nil)
	if err != nil {
		return nil, fmt.Errorf("创建截图运行器失败: %v", err)
	}
	defer runnerInstance.Close()

	result, err := driver.Witness(req.URL, &opts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return nil, fmt.Errorf(result.FailedReason)
	}

	return result, nil
}

// SetupRoutes 设置API路由
func (s *Server) SetupRoutes() {
	// 添加API密钥验证中间件到所有API请求
	apiAuth := s.CreateAuthMiddleware()

	// 添加并发限制中间件
	s.Router.Use(CreateConcurrencyLimitMiddleware())

	// 应用认证中间件
	s.Router.Use(apiAuth)

	// 截图相关路由
	s.Router.HandleFunc("/screenshot", s.HandleScreenshot).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/batch", s.HandleBatchScreenshot).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/screenshots_list", s.HandleListScreenshots).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/get_screenshot/{filename}", s.HandleGetScreenshot).Methods("GET", "OPTIONS")

	// 设置静态文件服务
	s.Router.PathPrefix("/screenshots/").Handler(
		http.StripPrefix("/screenshots/", http.FileServer(http.Dir(s.Options.ScreenshotPath))))

	// 添加首页路由，显示API信息
	s.Router.HandleFunc("/", s.HandleRoot).Methods("GET", "OPTIONS")

	// 添加状态监控和健康检查端点
	s.Router.HandleFunc("/stats", HandleStats).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/health", HandleHealth).Methods("GET", "OPTIONS")
}

// HandleRoot 处理根路径请求
func (s *Server) HandleRoot(w http.ResponseWriter, r *http.Request) {
	// 返回API信息
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "GO-SNIR API 服务",
		Data: map[string]interface{}{
			"version":        "1.0.0",
			"documentation":  "https://github.com/cyberspacesec/go-snir",
			"endpoints":      []string{"/screenshot", "/batch", "/screenshots_list", "/stats", "/health"},
			"screenshot_dir": s.Options.ScreenshotPath,
		},
	})
}

// Run 启动API服务器
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.Options.Host, s.Options.Port)
	log.Info("启动API服务器", "address", addr)

	// 输出配置信息
	active, waiting, max, queue, _ := GetConcurrencyStats()
	log.Info("服务器并发设置",
		"active", active,
		"waiting", waiting,
		"max_concurrent", max,
		"queue_size", queue,
	)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         addr,
		Handler:      s.Router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}
