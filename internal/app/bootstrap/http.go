package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
)

func NewEngine() *gin.Engine {
	registerCustomValidators()
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())
	RegisterCommonRoutes(engine)
	return engine
}

func registerCustomValidators() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	_ = v.RegisterValidation("checkform", func(fl validator.FieldLevel) bool {
		pattern := fl.Param()
		if pattern == "" {
			return true
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(fl.Field().String())
	})
}

func RegisterCommonRoutes(engine *gin.Engine) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
}

func NewHTTPServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func RunHTTPServer(server *http.Server, logger *zap.Logger, appName string, shutdownTimeout time.Duration) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("start server failed", zap.String("app", appName), zap.Error(err))
		}
	}()

	logger.Info(appName+" started", zap.String("addr", server.Addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
