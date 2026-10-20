package server

import (
	"github.com/David3310273/go-agent/benchmarker/metrics"
	"github.com/David3310273/go-agent/core"
	"github.com/gin-gonic/gin"
)

// MetricsServer is a standalone Prometheus metrics server using Gin for the /metrics route.
// The agent core is completely unaware; data is written solely through the Collector's Record* methods.
type MetricsServer struct {
	config    *core.BenchmarkerConfig
	collector *metrics.PrometheusCollector
	router    *gin.Engine
	port      string
}

// NewMetricsServer creates the metrics server.
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

// Start starts the HTTP server and blocks until an error occurs.
func (m *MetricsServer) Start() error {
	return m.router.Run(m.port)
}

// Collector returns the PrometheusCollector for external metric writes.
func (m *MetricsServer) Collector() *metrics.PrometheusCollector {
	return m.collector
}

// Config returns the configuration.
func (m *MetricsServer) Config() *core.BenchmarkerConfig {
	return m.config
}

// Router returns the Gin Engine for external route additions.
func (m *MetricsServer) Router() *gin.Engine {
	return m.router
}

// Port returns the listening port (with colon prefix, e.g. ":9090").
func (m *MetricsServer) Port() string {
	return m.port
}
