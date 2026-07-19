package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"fxthreads/constants"
	"fxthreads/routes"
	"fxthreads/services"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	port        = os.Getenv("PORT")
	multiWriter io.Writer
)

func init() {
	if port == "" {
		panic("Env PORT is not set, exiting...")
	}

	if _, err := strconv.Atoi(port); err != nil {
		panic("The set PORT value is not an available port number.")
	}

	constants.BaseDomain = os.Getenv("BASE_DOMAIN")
	if constants.BaseDomain == "" {
		panic("Env BASE_DOMAIN is not set, exiting...")
	}

	if _, err := os.Stat(constants.LogDir); os.IsNotExist(err) {
		_ = os.MkdirAll(constants.LogDir, 0o755)
	}

	multiWriter = io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   filepath.Join(constants.LogDir, "latest.log"),
		MaxSize:    10,
		MaxBackups: 20,
		MaxAge:     30,
		Compress:   true,
	})
	slog.SetDefault(slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})))
}

func main() {
	gin.SetMode(gin.DebugMode)
	router := gin.New()

	router.Use(gin.LoggerWithWriter(multiWriter))
	router.Use(gin.RecoveryWithWriter(multiWriter))

	router.LoadHTMLGlob("templates/*")

	router.GET("/share/:shareID/", services.ProvideEmbed)

	router.GET("/share/:shareID", services.ProvideEmbed)

	router.GET("/@:username/post/:postID", services.ProvideEmbed)

	router.GET("@:username", func(ctx *gin.Context) {
		username := ctx.Param("username")
		target := constants.ThreadsURL
		if username != "" {
			target += "@" + username
		}
		ctx.Redirect(http.StatusMovedPermanently, target)
	})

	api := router.Group("/api/")
	routes.RegisterAPIRoutes(api)

	router.NoRoute(func(ctx *gin.Context) {
		ctx.Redirect(http.StatusMovedPermanently, constants.ProjectHome)
	})

	router.Run(fmt.Sprintf(":%s", port))
}
