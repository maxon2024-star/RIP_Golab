package repository

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db              *gorm.DB
	minio           *minio.Client
	redis           *redis.Client
	minioBucketName string
}

type RepositorySettings struct {
	PostgresDSN     string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucketName string
}

func New(settings *RepositorySettings) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(settings.PostgresDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(settings.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(settings.MinioAccessKey, settings.MinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	// Проверка наличия бакета, если нет - создание
	ctx := context.Background()
	exists, errBucketExists := minioClient.BucketExists(ctx, settings.MinioBucketName)
	if errBucketExists == nil && !exists {
		err = minioClient.MakeBucket(ctx, settings.MinioBucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("Minio MakeBucket Error: %v\n", err)
		}
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // из docker-compose
		Password: "password",       // из docker-compose
		DB:       0,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Repository{
		db:              db,
		minio:           minioClient,
		redis:           redisClient,
		minioBucketName: settings.MinioBucketName,
	}, nil
}

func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

func (r *Repository) GetMinio() *minio.Client {
	return r.minio
}

func (r *Repository) GetRedis() *redis.Client {
	return r.redis
}
