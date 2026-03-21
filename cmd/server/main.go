package main

import (
	_ "RIP_Golab/docs"
	"RIP_Golab/internal/app/config"
	"RIP_Golab/internal/app/dsn"
	"RIP_Golab/internal/app/handler"
	"RIP_Golab/internal/app/repository"
	"RIP_Golab/internal/pkg"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @title RIP Golab Photoeffect API
// @version 1.0
// @description API сервера для расчета фотоэффекта
// @host localhost:8000
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	router := gin.Default()

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	logrus.Infof("DB DSN: %s", postgresString)

	repSettings := &repository.RepositorySettings{
		PostgresDSN:     postgresString,
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucketName: os.Getenv("MINIO_BUCKET_NAME"),
	}

	rep, err := repository.New(repSettings)
	if err != nil {
		logrus.Fatalf("error initializing repository: %v", err)
	}

	hand := handler.NewHandler(rep)
	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
