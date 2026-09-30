package media

import (
	"medya-filter/pkg/models"
)

// GetCurrentMedia returns the primary active media track across platforms.
func GetCurrentMedia() (*models.MediaInfo, error) {
	list, err := GetActiveMediaList()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &models.MediaInfo{State: "stopped"}, nil
	}
	return &list[0], nil
}
