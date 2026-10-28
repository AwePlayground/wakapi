package imports

import (
	"testing"
	"time"

	"github.com/muety/wakapi/models"
	wakatime "github.com/muety/wakapi/models/compat/wakatime/v1"
)

func TestMapHeartbeatPreservesAIUsage(t *testing.T) {
	user := &models.User{ID: "alice"}
	entry := &wakatime.HeartbeatEntry{
		Id:                  "heartbeat-1",
		Entity:              "/work/main.go",
		Type:                "file",
		Time:                1_793_169_876.5,
		CreatedAt:           time.Unix(1_793_169_876, 0),
		AILineChanges:       42,
		HumanLineChanges:    -3,
		AISession:           "session-1",
		AIInputTokens:       100,
		AICachedInputTokens: 400,
		AIOutputTokens:      20,
		AIPromptLength:      55,
	}
	hb := mapHeartbeat(entry, nil, nil, user)
	if hb.AILineChanges != 42 || hb.HumanLineChanges != -3 || hb.AISession != "session-1" {
		t.Fatalf("AI line and session fields lost: %+v", hb)
	}
	if hb.AIInputTokens != 100 || hb.AICachedInputTokens != 400 || hb.AIOutputTokens != 20 || hb.AIPromptLength != 55 {
		t.Fatalf("AI token and prompt fields lost: %+v", hb)
	}
}
