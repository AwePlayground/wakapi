package routes

import (
	"testing"
	"time"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/utils"
	"github.com/muety/wakapi/views"
)

func TestAIOverviewTemplateParses(t *testing.T) {
	templates, err := utils.LoadTemplates(views.TemplateFiles, DefaultTemplateFuncs())
	if err != nil {
		t.Fatal(err)
	}
	if templates[conf.AIOverviewTemplate] == nil {
		t.Fatal("AI overview template missing")
	}
}

func TestResolveAIOverviewRangeUsesLocalCalendarDays(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 10, 28, 16, 0, 0, 0, loc)
	from, to, selected := resolveAIOverviewRange("7d", now, loc)
	if selected != "7d" || from.In(loc).Format("2006-01-02 15:04") != "2026-10-22 00:00" || !to.Equal(now) {
		t.Fatalf("wrong 7-day range: %v %v %s", from, to, selected)
	}
	from, _, selected = resolveAIOverviewRange("invalid", now, loc)
	if selected != "7d" || from.In(loc).Day() != 22 {
		t.Fatalf("invalid range was not normalized: %v %s", from, selected)
	}
}
