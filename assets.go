package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func mediaTypeToExt(mimeType string) string {
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getAssetPath(filename string, mediaType string) string {
	return fmt.Sprintf("%s%s", filename, mediaTypeToExt(mediaType))
}

func (cfg *apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg *apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg *apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	req, err := presignClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", fmt.Errorf("Presign get object failed: %w", err)
	}

	return req.URL, nil

}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	// No need to generate presigned url if VideoURL is empty
	if video.VideoURL == nil {
		return video, nil
	}

	expiresTime := 10 * time.Minute
	info := strings.Split(*video.VideoURL, ",")
	if len(info) != 2 {
		return video, errors.New("VideoURL should have text in format '<bucket>,<key>'")
	}

	url, err := generatePresignedURL(
		cfg.s3Client,
		info[0],
		info[1],
		expiresTime,
	)
	if err != nil {
		return video, fmt.Errorf("Couldn't generate presigned url: %w", err)
	}

	video.VideoURL = &url
	return video, nil
}
