package routes

import (
	"testing"
	"time"

	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/services"
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

func TestAIHandlerUsesPricingPathEnvironment(t *testing.T) {
	t.Setenv("WAKAPI_AI_PRICING_PATH", "/pricing/ai-pricing.json")
	if got := NewAIHandler(nil, nil).pricingFile; got != "/pricing/ai-pricing.json" {
		t.Fatalf("pricing path = %q", got)
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

func TestBuildAIChartBucketsAllTimeUsesEveryCalendarMonth(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	daily := []*services.AIDailyActivity{
		{Date: "2024-07-05", AIActivityTotals: services.AIActivityTotals{AILinesAdded: 10, HumanLinesAdded: 2}},
		{Date: "2024-07-08", AIActivityTotals: services.AIActivityTotals{AILinesDeleted: 5}},
		{Date: "2024-09-02", AIActivityTotals: services.AIActivityTotals{AILinesAdded: 7}},
	}
	to := time.Date(2024, 10, 12, 10, 0, 0, 0, loc)
	buckets := buildAIChartBuckets(daily, time.Unix(0, 0), to, "all", loc)
	if len(buckets) != 4 {
		t.Fatalf("want July through October, got %d buckets", len(buckets))
	}
	wantDates := []string{"2024-07-01", "2024-08-01", "2024-09-01", "2024-10-01"}
	wantLines := []int64{15, 0, 7, 0}
	for i, bucket := range buckets {
		if bucket.Date != wantDates[i] || bucket.AILines != wantLines[i] {
			t.Errorf("bucket %d = %s, %d lines; want %s, %d", i, bucket.Date, bucket.AILines, wantDates[i], wantLines[i])
		}
	}
	if buckets[0].HumanAdded != 2 {
		t.Errorf("July human additions = %d, want 2", buckets[0].HumanAdded)
	}
}

func TestBuildAIChartBucketsSevenDaysIncludesInactiveDates(t *testing.T) {
	loc := time.UTC
	from := time.Date(2026, 10, 22, 0, 0, 0, 0, loc)
	to := time.Date(2026, 10, 28, 12, 0, 0, 0, loc)
	daily := []*services.AIDailyActivity{{Date: "2026-10-24", AIActivityTotals: services.AIActivityTotals{AILinesAdded: 9}}}
	buckets := buildAIChartBuckets(daily, from, to, "7d", loc)
	if len(buckets) != 7 || buckets[0].Date != "2026-10-22" || buckets[6].Date != "2026-10-28" {
		t.Fatalf("wrong seven-day calendar range: %#v", buckets)
	}
	if buckets[2].AILines != 9 || buckets[3].AILines != 0 {
		t.Fatalf("wrong daily counts around active date: %#v %#v", buckets[2], buckets[3])
	}
}

func TestAIChartBucketIncludesReportedHumanLines(t *testing.T) {
	bucket := &AIChartBucket{AILines: 80, HumanAdded: 12, HumanDeleted: 8}
	if bucket.HumanLines() != 20 || bucket.TotalLines() != 100 {
		t.Fatalf("wrong line totals: human=%d total=%d", bucket.HumanLines(), bucket.TotalLines())
	}
	if bucket.AIShare() != 80 || bucket.HumanShare() != 20 {
		t.Fatalf("wrong chart segments: AI=%d%% human=%d%%", bucket.AIShare(), bucket.HumanShare())
	}
	vm := &AIOverviewViewModel{MaxChartLines: 100}
	if vm.ChartBarHeight(bucket) != 100 {
		t.Fatalf("stacked bar height = %d, want 100", vm.ChartBarHeight(bucket))
	}
	onlyHuman := &AIChartBucket{HumanAdded: 5}
	if onlyHuman.AIShare() != 0 || onlyHuman.HumanShare() != 100 || vm.ChartBarHeight(onlyHuman) != 5 {
		t.Fatalf("human-only bucket not represented: %#v", onlyHuman)
	}
}
