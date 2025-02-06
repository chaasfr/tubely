package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	fileReceived, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer fileReceived.Close()

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "missing content-type", err)
		return
	}

	videoDb, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "video not found", err)
		return
	}

	if videoDb.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "video does not belong to user", err)
		return
	}

	mediaTypeSlice := strings.Split(mediaType, "/")
	fileExtension := mediaTypeSlice[len(mediaTypeSlice)-1]
	filename := fmt.Sprintf("%s.%s", videoID,fileExtension)
	fileFullPath := filepath.Join(cfg.assetsRoot, filename)
	fileSaved, err := os.Create(fileFullPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving thumbnial", err)
		return
	}
	_, err = io.Copy(fileSaved, fileReceived)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving thumbnial", err)
		return
	}

	thUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	videoDb.ThumbnailURL = &thUrl

	err = cfg.db.UpdateVideo(videoDb)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving thumbnail", err)
	}

	respondWithJSON(w, http.StatusOK, videoDb)
}
