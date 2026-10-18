package agent

import (
	_ "embed"

	"github.com/amigoer/bucky/internal/core"
)

//go:embed skills/bucky/SKILL.md
var skillText string

func SkillText() (string, error) {
	if skillText == "" {
		return "", core.NewError("internal", "The embedded skill is unavailable")
	}
	return skillText, nil
}
