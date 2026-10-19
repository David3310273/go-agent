package server

import (
	"github.com/David3310273/go-agent/benchmarker/metrics"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

// MetricsServer 独立的 Prometheus 指标 App，使用 Gin 注册 /metrics 路由。
// agent core 完全无感知，仅通过 Collector 的 Record* 方法写入数据。
type MetricsServer struct {
	config    *core.BenchmarkerConfig
	collector *metrics.PrometheusCollector
	router    *gin.Engine
	port      string
}

// NewMetricsServer 创建指标 App。
func NewMetricsServer(config *core.BenchmarkerConfig, port string) *MetricsServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	ms := &MetricsServer{
		config:    config,
		collector: metrics.NewPrometheusCollector(),
		router:    router,
		port:      port,
	}

	ms.registerRoutes()
	return ms
}

func (m *MetricsServer) registerRoutes() {
	m.router.GET("/metrics", gin.WrapH(m.collector.Handler()))
}

// Start 启动 HTTP server 并阻塞直到出错。
func (m *MetricsServer) Start() error {
	return m.router.Run(m.port)
}

// Collector 返回 PrometheusCollector，供外部写入指标数据。
func (m *MetricsServer) Collector() *metrics.PrometheusCollector {
	return m.collector
}

// Config 返回配置。
func (m *MetricsServer) Config() *core.BenchmarkerConfig {
	return m.config
}

// Router 返回 Gin Engine，供外部追加路由。
func (m *MetricsServer) Router() *gin.Engine {
	return m.router
}

// Port 返回监听端口（含冒号前缀，如 "9090"）。
func (m *MetricsServer) Port() string {
	return m.port
}
