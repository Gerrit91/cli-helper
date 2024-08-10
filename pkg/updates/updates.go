package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Gerrit91/cli-helper/pkg/waybar"
)

type (
	updates struct {
		pacman int
		aur    int
	}
)

func get() (*updates, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "checkupdates", "--nocolor")
	checkupdatesOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error checking updates: %w (%s)", err, string(checkupdatesOutput))
	}

	// this sometimes failed for some reason
	// cmd = exec.CommandContext(ctx, "yay", "-Qu", "--aur")
	// yayOutput, err := cmd.CombinedOutput()
	// if err != nil {
	// 	// return nil, fmt.Errorf("error retrieving updates from yay: %w (%s)", err, string(yayOutput))
	// }

	return &updates{
		pacman: countOutputLines(string(checkupdatesOutput)),
		// aur:    countOutputLines(string(yayOutput)),
	}, nil
}

func countOutputLines(s string) int {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0
	}

	lines := strings.Split(trimmed, "\n")

	return len(lines)
}

func PrintForWaybar() error {
	var o waybar.Output
	updates, err := get()
	if err != nil {
		o = waybar.Output{
			Text:       "--",
			Alt:        "",
			Tooltip:    err.Error(),
			Class:      "",
			Percentage: "",
		}
	} else {
		o = waybar.Output{
			Text:       fmt.Sprintf("%d", updates.pacman+updates.aur),
			Alt:        "",
			Tooltip:    fmt.Sprintf("There are updates available!"),
			Class:      "",
			Percentage: "",
		}
	}

	raw, err := json.Marshal(o)
	if err != nil {
		return err
	}

	fmt.Println(string(raw))

	return nil
}
