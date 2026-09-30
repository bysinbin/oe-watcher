//go:build !darwin && !windows && !linux

package media

import (
	"medya-filter/pkg/models"
)

// GetActiveMediaList fallback for unsupported OS platforms.
func GetActiveMediaList() ([]models.MediaInfo, error) {
	return []models.MediaInfo{}, nil
}

// SkipTrack fallback for unsupported OS platforms.
func SkipTrack(info models.MediaInfo) error {
	return nil
}

// PauseTrack fallback for unsupported OS platforms.
func PauseTrack(info models.MediaInfo) error {
	return nil
}

// PlayPauseTrack fallback for unsupported OS platforms.
func PlayPauseTrack(info models.MediaInfo) error {
	return nil
}
