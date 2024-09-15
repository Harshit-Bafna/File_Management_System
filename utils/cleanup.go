package utils

import (
	"authentication/config"
	"authentication/models"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func initS3Session() *s3.S3 {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	return s3.New(sess)
}

func DeleteExpiredFiles() {
	s3Client := initS3Session()

	for {
		files, err := getExpiredFiles()
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, file := range files {
			err := deleteFileFromS3(s3Client, file.FileURL)
			if err != nil {
				continue
			}

			err = deleteFileMetadata(file.FileID)
			if err != nil {
				continue
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func getExpiredFiles() ([]models.FileMetadata, error) {
	rows, err := config.DB.Query(`
		SELECT id, file_name, file_size, s3_url, file_extension, shared_user, expiry_date
		FROM files
		WHERE expiry_date <= NOW()
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying expired files: %w", err)
	}
	defer rows.Close()

	var files []models.FileMetadata
	for rows.Next() {
		var file models.FileMetadata
		err := rows.Scan(&file.FileID, &file.FileName, &file.FileSize, &file.FileURL, &file.FileType, &file.SharedUser, &file.ExpiryDate)
		if err != nil {
			return nil, fmt.Errorf("error scanning expired file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

func deleteFileFromS3(s3Client *s3.S3, s3URL string) error {
	bucketName := "go-file-management-system-bucket"
	objectKey := extractObjectKey(s3URL)

	_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("error deleting object from S3: %w", err)
	}

	return nil
}

func deleteFileMetadata(fileID int) error {
	_, err := config.DB.Exec("DELETE FROM files WHERE id = $1", fileID)
	if err != nil {
		return fmt.Errorf("error deleting file metadata: %w", err)
	}
	return nil
}

func extractObjectKey(s3URL string) string {
	return s3URL[strings.LastIndex(s3URL, "/")+1:]
}
