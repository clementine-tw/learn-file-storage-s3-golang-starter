package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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

	// Find video
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}
	// Check if user own the video
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not owner of the video", err)
		return
	}

	// Parse multipart form
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	// Read the uploaded thumbnail file
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form file", err)
		return
	}
	defer file.Close()

	mediatype, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn'g parse content-type", err)
		return
	}

	// Check media type
	if !(mediatype == "image/png" || mediatype == "image/jpeg") {
		respondWithError(w, http.StatusBadRequest, "Only accept PNG and JPEG", err)
		return
	}

	// save file
	randbytes := make([]byte, 32)
	_, err = rand.Read(randbytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fill random bytes", err)
		return
	}
	filename := base64.RawURLEncoding.EncodeToString(randbytes)

	assetPath := getAssetPath(filename, mediatype)
	dst, err := os.Create(cfg.getAssetDiskPath(assetPath))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write file", err)
		return
	}

	// Update thumbnail url of the video
	url := cfg.getAssetURL(assetPath)
	video.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update thumbnail url", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
