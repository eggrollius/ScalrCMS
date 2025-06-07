package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppContext struct {
	Config   *Config
	S3Client *s3.Client
	DB       *gorm.DB
}

func AppContextMiddleware(appCtx *AppContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("appCtx", appCtx)
		c.Next()
	}
}

func GetAppContext(c *gin.Context) *AppContext {
	if v, exists := c.Get("appCtx"); exists {
		if appCtx, ok := v.(*AppContext); ok {
			return appCtx
		}
	}
	return nil
}

func NewS3Client(cfg *Config) (*s3.Client, error) {
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "http://localhost:8000",
			SigningRegion: cfg.S3Region,
		}, nil
	})

	s3Cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.S3Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			"",
		)),
		config.WithEndpointResolver(customResolver),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(s3Cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO/Kong
	}), nil
}

func main() {
	if err := InitDatabase(); err != nil {
		panic("failed to connect database")
	}
	DB.AutoMigrate(&Video{})

	cfg := LoadConfig()

	s3Client, err := NewS3Client(cfg)
	if err != nil {
		panic("failed to create S3 client: " + err.Error())
	}

	appCtx := &AppContext{
		Config:   cfg,
		S3Client: s3Client,
		DB:       DB,
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.Use(AppContextMiddleware(appCtx))

	router.POST("/api/videos/initialize", InitializeVideoHandler)

	router.Run(":8080")
}
