package controllers

import (
	"authentication/config"
	"authentication/models"
	"authentication/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis/v8"
)

const fileCacheExpiration = 5 * time.Minute

func getFileCacheKey(fileID int) string {
	return fmt.Sprintf("file_metadata_%d", fileID)
}

func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No token found in cookies", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileSize := handler.Size
	fileName := handler.Filename
	fileExtension := filepath.Ext(fileName)
	encodedFileName := url.PathEscape(fileName)

	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String("go-file-management-system-bucket"),
		Key:    aws.String(encodedFileName),
		Body:   file,
	}

	_, err = config.S3Uploader.Upload(uploadInput)
	if err != nil {
		http.Error(w, "Error uploading file to S3: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fileURL := "https://go-file-management-system-bucket.s3.eu-north-1.amazonaws.com/" + encodedFileName
	expiryDate := time.Now().Add(1 * time.Minute)

	fileID, err := models.SaveFileMetadata(userID, fileName, int(fileSize), fileURL, fileExtension, false, expiryDate)
	if err != nil {
		http.Error(w, "Error saving file metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	config.RedisClient.Del(config.Ctx, fmt.Sprintf("user_files_%d", userID))

	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"fileURL": fileURL,
		"fileID":  fileID,
	}
	json.NewEncoder(w).Encode(response)
}

func GetUserFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No token found in cookies", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	cacheKey := fmt.Sprintf("user_files_%d", userID)
	cachedFiles, err := config.RedisClient.Get(config.Ctx, cacheKey).Result()

	if err == redis.Nil {
		files, err := models.GetUserFiles(userID)
		if err != nil {
			http.Error(w, "Error retrieving file metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}

		cachedData, _ := json.Marshal(files)
		config.RedisClient.Set(config.Ctx, cacheKey, cachedData, fileCacheExpiration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"files": files,
		}
		json.NewEncoder(w).Encode(response)
	} else if err != nil {
		http.Error(w, "Error retrieving data from cache: "+err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedFiles))
	}
}

func RenameFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No token found in cookies", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.URL.Query().Get("id")
	newName := r.URL.Query().Get("new_name")

	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil || newName == "" {
		http.Error(w, "Invalid file ID or new name", http.StatusBadRequest)
		return
	}

	err = models.UpdateFileName(userID, fileID, newName)
	if err != nil {
		http.Error(w, "Error updating file name: "+err.Error(), http.StatusInternalServerError)
		return
	}

	config.RedisClient.Del(config.Ctx, getFileCacheKey(fileID))
	config.RedisClient.Del(config.Ctx, fmt.Sprintf("user_files_%d", userID))

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "File renamed successfully")
}

func SearchUserFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No token found in cookies", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fileName := r.URL.Query().Get("fileName")
	uploadDate := r.URL.Query().Get("uploadDate")
	fileExtension := r.URL.Query().Get("fileExtension")

	limit := 10
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	files, err := models.SearchUserFiles(userID, fileName, uploadDate, fileExtension, limit, offset)
	if err != nil {
		http.Error(w, "Error retrieving file metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"files": files,
	}
	json.NewEncoder(w).Encode(response)
}

func ShareFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No token found in cookies", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromToken(cookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.URL.Query().Get("id")
	if fileIDStr == "" {
		http.Error(w, "Missing file_id", http.StatusBadRequest)
		return
	}

	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil {
		http.Error(w, "Invalid file_id", http.StatusBadRequest)
		return
	}

	now := time.Now()

	err = models.UpdateSharedStatus(fileID, userID, true, now)
	if err != nil {
		http.Error(w, "Error updating shared status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tempLink := fmt.Sprintf("http://localhost:8080/share/%d", fileID)

	err = models.SetTemporaryLinkExpiry(fileID, 1*time.Minute)
	if err != nil {
		http.Error(w, "Error setting temporary link expiry: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"link": tempLink,
	}
	json.NewEncoder(w).Encode(response)
}

func AccessSharedFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	fileIDStr := strings.TrimPrefix(r.URL.Path, "/share/")
	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil || fileIDStr == "" {
		http.Error(w, "Invalid file_id", http.StatusBadRequest)
		return
	}

	file, err := models.GetFileByID(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if !file.SharedUser {
		http.Error(w, "File is not shared or the link has expired", http.StatusUnauthorized)
		return
	}

	if file.SharedAt.Valid {
		if time.Since(file.SharedAt.Time) > 1*time.Minute {
			err := models.UpdateSharedStatus(fileID, file.UserID, false, time.Now())
			if err != nil {
				http.Error(w, "Error revoking shared status", http.StatusInternalServerError)
				return
			}
			http.Error(w, "The file sharing link has expired", http.StatusUnauthorized)
			return
		}
	} else {
		http.Error(w, "File sharing link has expired", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"file_name": file.FileName,
		"s3_url":    file.FileURL,
		"file_size": file.FileSize,
		"file_type": file.FileType,
	}
	json.NewEncoder(w).Encode(response)
}
