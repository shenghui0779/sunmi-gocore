package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunmi-OS/gocore/v2/glog"
	"github.com/sunmi-OS/gocore/v2/lib/middleware"
	"github.com/sunmi-OS/gocore/v2/lib/prometheus"
	zipkin_opentracing "github.com/sunmi-OS/gocore/v2/lib/tracing/gin/zipkin-opentracing"
)

const (
	_HookStart hookType = "server_start"
	_HookClose hookType = "server_close"
	_HookExit  hookType = "sys_exit"
)

type hookType string

type HookFunc func(c context.Context)

type GinEngine struct {
	Gin              *gin.Engine
	server           *http.Server
	timeout          time.Duration
	addrPort         string
	IgnoreReleaseLog bool
	hookMaps         map[hookType][]func(c context.Context)
}

func NewGinServer(ops ...Option) *GinEngine {
	cfg := defaultServerConfig
	for _, o := range ops {
		o(cfg)
	}
	g := gin.New()
	engine := &GinEngine{Gin: g, addrPort: cfg.host + ":" + strconv.Itoa(cfg.port), hookMaps: make(map[hookType][]func(c context.Context))}
	engine.timeout = cfg.readTimeout
	engine.server = &http.Server{
		Addr:         engine.addrPort,
		Handler:      g,
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
	}
	g.Use(engine.logger(true), middleware.Recovery())
	if cfg.openTrace {
		//引入链路追踪中间件
		endPointUrl := os.Getenv("ZIPKIN_BASE_URL")
		appName := os.Getenv("APP_NAME")
		if endPointUrl == "" || appName == "" {
			panic("请配置环境变量 ZIPKIN_BASE_URL 和 APP_NAME")
		}
		g.Use(zipkin_opentracing.ZipKinOpentracing(appName, 1, endPointUrl))
	}
	if !cfg.debug {
		gin.SetMode(gin.ReleaseMode)
	}
	// 引入 prometheus 中间件
	prometheus.NewPrometheus("app").Use(g)
	// default health check
	g.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	// default pprof
	pp := g.Group("/debug/pprof")
	{
		pp.GET("/index", func(c *gin.Context) { pprof.Index(c.Writer, c.Request) })
		pp.GET("/cmdline", func(c *gin.Context) { pprof.Cmdline(c.Writer, c.Request) })
		pp.GET("/profile", func(c *gin.Context) { pprof.Profile(c.Writer, c.Request) })
		pp.GET("/symbol", func(c *gin.Context) { pprof.Symbol(c.Writer, c.Request) })
		pp.GET("/trace", func(c *gin.Context) { pprof.Trace(c.Writer, c.Request) })
	}
	return engine
}

// 添加 GinServer 服务启动时的钩子函数
func (g *GinEngine) AddStartHook(hooks ...HookFunc) *GinEngine {
	for _, fn := range hooks {
		if fn != nil {
			g.hookMaps[_HookStart] = append(g.hookMaps[_HookStart], fn)
		}
	}
	return g
}

// 添加 GinServer 服务关闭时的钩子函数
func (g *GinEngine) AddCloseHook(hooks ...HookFunc) *GinEngine {
	for _, fn := range hooks {
		if fn != nil {
			g.hookMaps[_HookClose] = append(g.hookMaps[_HookClose], fn)
		}
	}
	return g
}

// 添加 GinServer 进程退出时钩子函数
func (g *GinEngine) AddExitHook(hooks ...HookFunc) *GinEngine {
	for _, fn := range hooks {
		if fn != nil {
			g.hookMaps[_HookExit] = append(g.hookMaps[_HookExit], fn)
		}
	}
	return g
}

func (g *GinEngine) Start() {
	go func() {
		glog.WarnF("Listening and serving HTTP on %s", g.addrPort)
		if err := g.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				glog.Warn("http: Server closed")
				return
			}
			panic(fmt.Sprintf("server.ListenAndServe(), error(%+v).", err))
		}
	}()
	// call when server start hooks
	for _, fn := range g.hookMaps[_HookStart] {
		fn(context.Background())
	}
}

// 监听信号
func (g *GinEngine) NotifySignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		si := <-ch
		switch si {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			glog.WarnF("get a signal %s, stop the process", si.String())
			// close gin http server
			g.Close()
			ctx, cancelFunc := context.WithTimeout(context.Background(), g.timeout)
			// call before close hooks
			go func() {
				if a := recover(); a != nil {
					glog.ErrorF("panic: %v", a)
				}
				for _, fn := range g.hookMaps[_HookClose] {
					fn(ctx)
				}
			}()
			// wait for a second to finish processing
			time.Sleep(g.timeout)
			cancelFunc()
			// call after close hooks
			for _, fn := range g.hookMaps[_HookExit] {
				fn(context.Background())
			}
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}

func (g *GinEngine) StartAndNotify() {
	g.Start()
	g.NotifySignal()
}

func (g *GinEngine) Close() {
	if g.server != nil {
		// disable keep-alives on existing connections
		g.server.SetKeepAlivesEnabled(false)
		_ = g.server.Shutdown(context.Background())
	}
}

// logger
func (g *GinEngine) logger(ignoreRelease bool) gin.HandlerFunc {
	g.IgnoreReleaseLog = ignoreRelease
	return func(c *gin.Context) {
		// Start time
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()
		if raw != "" {
			path = path + "?" + raw
		}

		// ignore logger output
		if gin.Mode() == gin.ReleaseMode && g.IgnoreReleaseLog {
			return
		}

		// End time
		end := time.Now()
		fmt.Fprintf(os.Stdout, "[GIN] %s | %3d | %13v | %15s | %-7s %#v\n%s", end.Format("2006/01/02 - 15:04:05"), c.Writer.Status(), end.Sub(start), c.ClientIP(), c.Request.Method, path, c.Errors.ByType(gin.ErrorTypePrivate).String())
	}
}
