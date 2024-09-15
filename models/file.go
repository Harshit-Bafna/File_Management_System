package models

import (
	"authentication/config"
	"database/sql"
	"fmt"
	"time"
)

type File struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	FileName   string    `json:"file_name"`
	UploadDate time.Time `json:"upload_date"`
	FileSize   int       `json:"file_size"`
	S3URL      string    `json:"s3_url"`
	SharedUser bool      `json:"shared_user"`
}

type FileMetadata struct {
	FileID     int          `json:"id"`
	UserID     int          `json:"user_id"`
	FileName   string       `json:"file_name"`
	FileSize   int          `json:"file_size"`
	FileURL    string       `json:"s3_url"`
	FileType   string       `json:"file_extension"`
	SharedUser bool         `json:"shared_user"`
	SharedAt   sql.NullTime `json:"shared_at"`
	ExpiryDate sql.NullTime `json:"expiry_date"`
}

func SaveFileMetadata(userID int, fileName string, fileSize int, fileURL, fileExtension string, sharedUser bool, expiryDate time.Time) (int, error) {
	var fileID int
	err := config.DB.QueryRow(`
        INSERT INTO files (user_id, file_name, file_size, s3_url, file_extension, shared_user, shared_at, expiry_date)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		userID, fileName, fileSize, fileURL, fileExtension, sharedUser, time.Now(), expiryDate,
	).Scan(&fileID)
	return fileID, err
}

func GetUserFiles(userID int) ([]FileMetadata, error) {
	rows, err := config.DB.Query(`
		SELECT id, user_id, file_name, file_size, s3_url, file_extension, shared_user, shared_at
		FROM files
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileMetadata
	for rows.Next() {
		var file FileMetadata
		err := rows.Scan(&file.FileID, &file.UserID, &file.FileName, &file.FileSize, &file.FileURL, &file.FileType, &file.SharedUser, &file.SharedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func UpdateFileName(userID, fileID int, newName string) error {
	_, err := config.DB.Exec(`
		UPDATE files
		SET file_name = $1
		WHERE user_id = $2 AND id = $3`,
		newName, userID, fileID,
	)
	return err
}

func SearchUserFiles(userID int, fileName, uploadDate, fileType string, limit, offset int) ([]FileMetadata, error) {
	query := `
		SELECT id, file_name, file_size, s3_url, file_extension, shared_user
		FROM files
		WHERE user_id = $1
	`

	params := []interface{}{userID}
	paramIndex := 2

	if fileName != "" {
		query += fmt.Sprintf(" AND file_name ILIKE $%d", paramIndex)
		params = append(params, "%"+fileName+"%")
		paramIndex++
	}
	if uploadDate != "" {
		query += fmt.Sprintf(" AND upload_date::date = $%d", paramIndex)
		params = append(params, uploadDate)
		paramIndex++
	}
	if fileType != "" {
		query += fmt.Sprintf(" AND file_extension = $%d", paramIndex)
		params = append(params, fileType)
		paramIndex++
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	params = append(params, limit, offset)

	rows, err := config.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("error querying the database: %w", err)
	}
	defer rows.Close()

	var files []FileMetadata

	for rows.Next() {
		var file FileMetadata
		if err := rows.Scan(&file.FileID, &file.FileName, &file.FileSize, &file.FileURL, &file.FileType, &file.SharedUser); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return files, nil
}

func UpdateSharedStatus(fileID int, userID int, sharedUser bool, sharedAt time.Time) error {
	_, err := config.DB.Exec(`
        UPDATE files
        SET shared_user = $1, shared_at = $2
        WHERE id = $3 AND user_id = $4`,
		sharedUser, sharedAt, fileID, userID)
	return err
}

func SetTemporaryLinkExpiry(fileID int, duration time.Duration) error {
	go func() {
		time.Sleep(duration)
		err := UpdateSharedStatus(fileID, 0, false, time.Now())
		if err != nil {
			fmt.Printf("Error resetting shared_user status for file_id %d: %v\n", fileID, err)
		}
	}()
	return nil
}

func GetFileByID(fileID int) (*FileMetadata, error) {
	var file FileMetadata

	err := config.DB.QueryRow(`
        SELECT id, file_name, file_size, s3_url, file_extension, shared_user, shared_at 
        FROM files 
        WHERE id = $1`, fileID).
		Scan(&file.FileID, &file.FileName, &file.FileSize, &file.FileURL, &file.FileType, &file.SharedUser, &file.SharedAt)

	if err != nil {
		return nil, err
	}
	return &file, nil
}
