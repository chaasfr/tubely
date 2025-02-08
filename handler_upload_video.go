package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	r.ParseMultipartForm(maxMemory)
	http.MaxBytesReader(w, r.Body, maxMemory)

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

	videoDb, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "videoID unknown", err)
		return
	}
	if userID != videoDb.UserID {
		respondWithError(w, http.StatusUnauthorized, "video does not belong to user", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	r.ParseMultipartForm(maxMemory)

	// "video" should match the HTML form input name
	fileReceived, header, err := r.FormFile("video")
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
	fileType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "cannot parse Content-Type", err)
		return
	}
	if fileType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "only Content-Type accepted are video/mp4", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error creating temp dir", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, fileReceived)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving temp video", err)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart) //resets pointer to the beginning
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error reseting temp video", err)
		return
	}
	tempFileFastStartPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error creating fastprocess video", err)
		return
	}
	defer os.Remove(tempFileFastStartPath)

	tempFileFastStart, err := os.Open(tempFileFastStartPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading fastprocess video", err)
		return
	}
	defer tempFileFastStart.Close()

	aspectRatio, err := getVideoAspectRatio(tempFileFastStartPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading aspect ratio", err)
		return
	}

	randomFileId := make([]byte, 32)
	_, err = rand.Read(randomFileId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving thumbnail", err)
		return
	}

	//randomName := base64.RawURLEncoding.EncodeToString(randomFileId)
	//fileName := fmt.Sprintf("%s/%s", aspectRatio, randomName)
	fileName := aspectRatio + ".mp4"


	videoUrl := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileName)
	videoDb.VideoURL = &videoUrl

	err = cfg.db.UpdateVideo(videoDb)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving video metadata", err)
		return
	}

	videoDb, err = cfg.dbVideoToSignedVideo(videoDb)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error generated presigned url", err)
		return
	}

	
	s3PutParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileName,
		Body:        tempFileFastStart,
		ContentType: &mediaType,
	}


	_, err = cfg.s3Client.PutObject(r.Context(), &s3PutParams)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error saving video", err)
		return
	}

	w.WriteHeader(204)
}
