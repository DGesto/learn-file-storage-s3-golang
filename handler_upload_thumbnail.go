package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// Parse Multipart form from the body of the request
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse Multipart Form", err)
		return
	}

	// Get thumbnail information from multipart form
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get thumbnail from form", err)
		return
	}
	defer file.Close()

	// Get media type
	mediaType := header.Header.Get("Content-Type")

	// Get data from the video the thumbnail will be associated with from the DB
	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get video information from database", err)
		return
	}

	// Check if video was found on the database
	if videoData.ID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", nil)
		return
	}

	// Check if video belongs to the user that is trying to upload the thumbnail
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner of this video", nil)
		return
	}

	// Get file extension from media type
	extensions, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get file extension", err)
		return
	}
	var extension string
	if len(extensions) > 0 {
		extension = extensions[0]
	} else {
		respondWithError(w, http.StatusBadRequest, "Could not get file extension", nil)
		return
	}

	// Create file name and file path
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	thumbnailURL := base64.StdEncoding.EncodeToString(randomBytes)
	filename := fmt.Sprintf("%s%s", thumbnailURL, extension)
	diskPath := filepath.Join(cfg.assetsRoot, filename)

	// Create file
	thumbnailFile, err := os.Create(diskPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create file", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = io.Copy(thumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create thumbnail copy", err)
		return
	}

	// Update video information
	newURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	videoData.ThumbnailURL = &newURL
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create thumbnail copy", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
