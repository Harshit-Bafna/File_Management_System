package config

import (
	"context"
	"database/sql"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

var (
	DB          *sql.DB
	RedisClient *redis.Client
	S3Session   *session.Session
	S3Uploader  *s3manager.Uploader
	Ctx         = context.Background()
)

func InitDB() {
	var err error
	connStr := "host=localhost user=authenticator dbname=User sslmode=disable password=databasePassword"
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	S3Session, err = session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})
	if err != nil {
		log.Fatal("Failed to create AWS session:", err)
	}
	S3Uploader = s3manager.NewUploader(S3Session)
}
