package repository

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

type Repository struct {
	db              *gorm.DB
	minio           *minio.Client
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

	return &Repository{
		db:              db,
		minio:           minioClient,
		minioBucketName: settings.MinioBucketName,
	}, nil
}

func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

func (r *Repository) GetMinio() *minio.Client {
	return r.minio
}
