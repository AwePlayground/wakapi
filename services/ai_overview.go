package services

import (
	"sort"
	"time"

	"github.com/muety/wakapi/models"
	"gorm.io/gorm"
)

type AIActivityTotals struct {
	AILinesAdded      int64
	AILinesDeleted    int64
	HumanLinesAdded   int64
	HumanLinesDeleted int64
	InputTokens       int64
	CachedInputTokens int64
	OutputTokens      int64
	PromptChars       int64
	PromptEvents      int64
	AISessions        int64
	Heartbeats        int64
}

func (t AIActivityTotals) TotalInputTokens() int64 {
	return t.InputTokens + t.CachedInputTokens
}

func (t AIActivityTotals) AILineChanges() int64 {
	return t.AILinesAdded + t.AILinesDeleted
}

type AIDailyActivity struct {
	Date string
	AIActivityTotals
}

type AIActivityGroup struct {
	Name string
	AIActivityTotals
}

type AIOverview struct {
	Totals   AIActivityTotals
	Daily    []*AIDailyActivity
	Models   []*AIActivityGroup
	Projects []*AIActivityGroup
}

// AggregateAIOverview streams the selected heartbeat columns. Memory use grows with
// the number of days, models and projects, not with the number of heartbeats.
func AggregateAIOverview(db *gorm.DB, userID string, loc *time.Location, from, to time.Time) (*AIOverview, error) {
	if loc == nil {
		loc = time.UTC
	}
	fromArg, toArg := interface{}(models.CustomTime(from)), interface{}(models.CustomTime(to))
	if db.Dialector.Name() == "sqlite" {
		fromArg, toArg = from.UnixMilli(), to.UnixMilli()
	}
	rows, err := db.Raw(`select time, coalesce(project,''), coalesce(ai_model,''),
		coalesce(ai_line_changes,0), coalesce(human_line_changes,0),
		coalesce(ai_input_tokens,0), coalesce(ai_cached_input_tokens,0),
		coalesce(ai_output_tokens,0), coalesce(a_iprompt_length,0),
		coalesce(ai_session,'') from heartbeats
		where user_id = ? and time >= ? and time < ? order by time`, userID, fromArg, toArg).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &AIOverview{}
	days := make(map[string]*AIDailyActivity)
	modelsByName := make(map[string]*AIActivityGroup)
	projectsByName := make(map[string]*AIActivityGroup)
	sessions := make(map[string]struct{})
	daySessions := make(map[string]map[string]struct{})
	for rows.Next() {
		var at models.CustomTime
		var project, model, session string
		var aiLines, humanLines, input, cached, output, prompt int64
		if err := rows.Scan(&at, &project, &model, &aiLines, &humanLines, &input, &cached, &output, &prompt, &session); err != nil {
			return nil, err
		}
		day := at.T().In(loc).Format("2006-01-02")
		if days[day] == nil {
			days[day] = &AIDailyActivity{Date: day}
		}
		applyAIActivity(&result.Totals, aiLines, humanLines, input, cached, output, prompt)
		applyAIActivity(&days[day].AIActivityTotals, aiLines, humanLines, input, cached, output, prompt)
		if session != "" {
			sessions[session] = struct{}{}
			if daySessions[day] == nil {
				daySessions[day] = make(map[string]struct{})
			}
			daySessions[day][session] = struct{}{}
		}
		isAI := aiLines != 0 || input != 0 || cached != 0 || output != 0 || prompt != 0 || session != ""
		if isAI {
			if model == "" {
				model = "Unknown model"
			}
			if modelsByName[model] == nil {
				modelsByName[model] = &AIActivityGroup{Name: model}
			}
			applyAIActivity(&modelsByName[model].AIActivityTotals, aiLines, humanLines, input, cached, output, prompt)
		}
		if isAI {
			if project == "" {
				project = "Unknown project"
			}
			if projectsByName[project] == nil {
				projectsByName[project] = &AIActivityGroup{Name: project}
			}
			applyAIActivity(&projectsByName[project].AIActivityTotals, aiLines, humanLines, input, cached, output, prompt)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Totals.AISessions = int64(len(sessions))
	for day, stats := range days {
		stats.AISessions = int64(len(daySessions[day]))
		result.Daily = append(result.Daily, stats)
	}
	for _, stats := range modelsByName {
		result.Models = append(result.Models, stats)
	}
	for _, stats := range projectsByName {
		result.Projects = append(result.Projects, stats)
	}
	sort.Slice(result.Daily, func(i, j int) bool { return result.Daily[i].Date < result.Daily[j].Date })
	sort.Slice(result.Models, func(i, j int) bool { return result.Models[i].AILineChanges() > result.Models[j].AILineChanges() })
	sort.Slice(result.Projects, func(i, j int) bool { return result.Projects[i].AILineChanges() > result.Projects[j].AILineChanges() })
	return result, nil
}

func applyAIActivity(dst *AIActivityTotals, aiLines, humanLines, input, cached, output, prompt int64) {
	dst.Heartbeats++
	if aiLines >= 0 {
		dst.AILinesAdded += aiLines
	} else {
		dst.AILinesDeleted -= aiLines
	}
	if humanLines >= 0 {
		dst.HumanLinesAdded += humanLines
	} else {
		dst.HumanLinesDeleted -= humanLines
	}
	dst.InputTokens += input
	dst.CachedInputTokens += cached
	dst.OutputTokens += output
	if prompt > 0 {
		dst.PromptEvents++
		dst.PromptChars += prompt
	}
}
