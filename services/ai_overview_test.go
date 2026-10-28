package services

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAggregateAIOverviewUsesUserTimezoneAndSeparatesTokenTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`create table heartbeats (
		user_id text, time integer, project text, ai_model text,
		ai_line_changes integer, human_line_changes integer,
		ai_input_tokens integer, ai_cached_input_tokens integer,
		ai_output_tokens integer, a_iprompt_length integer, ai_session text
	)`).Error; err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	insert := func(user, timestamp, model string, aiLines, humanLines, input, cached, output, prompt int, session string) {
		t.Helper()
		at, err := time.ParseInLocation("2006-01-02 15:04", timestamp, loc)
		if err != nil {
			t.Fatal(err)
		}
		err = db.Exec(`insert into heartbeats values (?,?,?,?,?,?,?,?,?,?,?)`, user, at.UnixMilli(), "demo", model,
			aiLines, humanLines, input, cached, output, prompt, session).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("alice", "2026-10-27 23:50", "Opus", 10, 0, 100, 200, 30, 50, "s1")
	insert("alice", "2026-10-28 00:10", "Opus", 20, -4, 300, 400, 50, 70, "s2")
	insert("alice", "2026-10-28 00:20", "Sonnet", -3, 8, 10, 0, 5, 0, "s2")
	insert("alice", "2026-10-28 00:30", "Androidstudio", 0, 0, 0, 0, 0, 0, "")
	insert("bob", "2026-10-28 00:25", "Opus", 999, 0, 999, 999, 999, 999, "other")
	from, _ := time.ParseInLocation("2006-01-02", "2026-10-28", loc)
	to := from.AddDate(0, 0, 1)
	result, err := AggregateAIOverview(db, "alice", loc, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if result.Totals.AILinesAdded != 20 || result.Totals.AILinesDeleted != 3 || result.Totals.HumanLinesAdded != 8 || result.Totals.HumanLinesDeleted != 4 {
		t.Fatalf("wrong line totals: %+v", result.Totals)
	}
	if result.Totals.InputTokens != 310 || result.Totals.CachedInputTokens != 400 || result.Totals.OutputTokens != 55 {
		t.Fatalf("wrong token totals: %+v", result.Totals)
	}
	if result.Totals.PromptEvents != 1 || result.Totals.PromptChars != 70 || result.Totals.AISessions != 1 {
		t.Fatalf("wrong prompt/session totals: %+v", result.Totals)
	}
	if len(result.Daily) != 1 || result.Daily[0].Date != "2026-10-28" || len(result.Models) != 2 {
		t.Fatalf("wrong breakdown: daily=%+v models=%+v", result.Daily, result.Models)
	}
}
