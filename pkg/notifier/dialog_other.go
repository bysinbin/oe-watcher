//go:build !darwin && !windows && !linux

package notifier

import (
	"fmt"

	"medya-filter/pkg/models"
)

func PlayAlertSound() {}

func ShowNotification(title, subtitle, message string) {
	fmt.Printf("[%s] %s: %s\n", title, subtitle, message)
}

func PromptAction(match *models.MatchResult) (string, error) {
	return ActionSkip, nil
}
