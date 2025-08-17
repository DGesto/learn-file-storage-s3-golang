package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

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

	// Get media type
	mediaType := header.Header.Get("Content-Type")

	// Get thumbnail image
	image, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not get thumbnail image from file", err)
		return
	}
	defer file.Close()

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

	// Encode thumbanil image as a base64 string
	imageBase64 := base64.StdEncoding.EncodeToString(image)

	// Create data URL with the thumbnail
	thumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, imageBase64)

	// Change thumbnail URL in the DB to the new thumbnail
	videoData.ThumbnailURL = &thumbnailURL

	// Update information of the video in the DB (with the new thumbnail URL)
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not update video data in the database", nil)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
