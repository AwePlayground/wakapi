package routes

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models/view"
	"github.com/muety/wakapi/services"
	"gorm.io/gorm"
)

type AIHandler struct {
	config      *conf.Config
	db          *gorm.DB
	userService services.IUserService
	pricingFile string
}

type AIOverviewViewModel struct {
	view.SharedLoggedInViewModel
	Overview       *services.AIOverview
	Cost           services.AICostSummary
	SelectedRange  string
	RangeLabel     string
	PriceUpdatedAt string
	HasPricing     bool
	ChartBuckets   []*AIChartBucket
	MaxChartLines  int64
}

type AIChartBucket struct {
	Date         string
	AILines      int64
	HumanAdded   int64
	HumanDeleted int64
}

func (bucket *AIChartBucket) HumanLines() int64 {
	return bucket.HumanAdded + bucket.HumanDeleted
}

func (bucket *AIChartBucket) TotalLines() int64 {
	return bucket.AILines + bucket.HumanLines()
}

func (bucket *AIChartBucket) AIShare() int {
	if bucket.TotalLines() == 0 {
		return 0
	}
	return int(math.Round(float64(bucket.AILines) * 100 / float64(bucket.TotalLines())))
}

func (bucket *AIChartBucket) HumanShare() int {
	if bucket.TotalLines() == 0 {
		return 0
	}
	return 100 - bucket.AIShare()
}

func NewAIHandler(db *gorm.DB, userService services.IUserService) *AIHandler {
	pricingFile := os.Getenv("WAKAPI_AI_PRICING_PATH")
	if pricingFile == "" {
		pricingFile = "/data/ai-pricing.json"
	}
	return &AIHandler{config: conf.Get(), db: db, userService: userService, pricingFile: pricingFile}
}

func (h *AIHandler) RegisterRoutes(router chi.Router) {
	r := chi.NewRouter()
	r.Use(middlewares.NewWebAuthenticateMiddleware(h.userService).
		WithRedirectTarget(defaultErrorRedirectTarget()).
		WithRedirectErrorMessage("unauthorized").Handler)
	r.Get("/", h.GetIndex)
	router.Mount("/ai", r)
}

func (h *AIHandler) GetIndex(w http.ResponseWriter, r *http.Request) {
	if h.config.IsDev() {
		loadTemplates()
	}
	user := middlewares.GetPrincipal(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	loc := user.TZ()
	from, to, selected := resolveAIOverviewRange(r.URL.Query().Get("range"), time.Now(), loc)
	overview, err := services.AggregateAIOverview(h.db, user.ID, loc, from, to)
	if err != nil {
		conf.Log().Request(r).Error("failed to aggregate AI activity", "userID", user.ID, "error", err)
		http.Error(w, "failed to load AI activity", http.StatusInternalServerError)
		return
	}
	catalog, err := services.LoadAIPriceCatalog(h.pricingFile)
	if err != nil && !os.IsNotExist(err) {
		conf.Log().Request(r).Warn("failed to load AI price cache", "error", err)
	}
	vm := &AIOverviewViewModel{
		SharedLoggedInViewModel: view.SharedLoggedInViewModel{
			SharedViewModel: view.NewSharedViewModel(h.config, nil), User: user,
		},
		Overview:      overview,
		Cost:          services.EstimateAICost(overview.Models, catalog),
		SelectedRange: selected,
		RangeLabel:    map[string]string{"7d": "Last 7 days", "30d": "Last 30 days", "all": "All time"}[selected],
		HasPricing:    catalog != nil,
	}
	if catalog != nil {
		vm.PriceUpdatedAt = catalog.UpdatedAt
	}
	vm.ChartBuckets = buildAIChartBuckets(overview.Daily, from, to, selected, loc)
	for _, bucket := range vm.ChartBuckets {
		if bucket.TotalLines() > vm.MaxChartLines {
			vm.MaxChartLines = bucket.TotalLines()
		}
	}
	if err := templates[conf.AIOverviewTemplate].Execute(w, vm); err != nil {
		conf.Log().Request(r).Error("failed to render AI overview", "error", err)
	}
}

func buildAIChartBuckets(daily []*services.AIDailyActivity, from, to time.Time, selected string, loc *time.Location) []*AIChartBucket {
	if loc == nil {
		loc = time.UTC
	}
	if selected == "all" {
		if len(daily) == 0 {
			return nil
		}
		first, err := time.ParseInLocation("2006-01-02", daily[0].Date, loc)
		if err != nil {
			return nil
		}
		byMonth := make(map[string]*AIChartBucket)
		for _, day := range daily {
			month := day.Date[:7]
			bucket := byMonth[month]
			if bucket == nil {
				bucket = &AIChartBucket{Date: month + "-01"}
				byMonth[month] = bucket
			}
			bucket.AILines += day.AILineChanges()
			bucket.HumanAdded += day.HumanLinesAdded
			bucket.HumanDeleted += day.HumanLinesDeleted
		}
		start := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, loc)
		endLocal := to.In(loc)
		end := time.Date(endLocal.Year(), endLocal.Month(), 1, 0, 0, 0, 0, loc)
		buckets := make([]*AIChartBucket, 0, (end.Year()-start.Year())*12+int(end.Month()-start.Month())+1)
		for month := start; !month.After(end); month = month.AddDate(0, 1, 0) {
			key := month.Format("2006-01")
			if bucket := byMonth[key]; bucket != nil {
				buckets = append(buckets, bucket)
			} else {
				buckets = append(buckets, &AIChartBucket{Date: key + "-01"})
			}
		}
		return buckets
	}
	byDate := make(map[string]*services.AIDailyActivity, len(daily))
	for _, day := range daily {
		byDate[day.Date] = day
	}
	buckets := make([]*AIChartBucket, 0, 30)
	for day := from.In(loc); !day.After(to.In(loc)); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		bucket := &AIChartBucket{Date: key}
		if stats := byDate[key]; stats != nil {
			bucket.AILines = stats.AILineChanges()
			bucket.HumanAdded = stats.HumanLinesAdded
			bucket.HumanDeleted = stats.HumanLinesDeleted
		}
		buckets = append(buckets, bucket)
	}
	return buckets
}

func resolveAIOverviewRange(selected string, now time.Time, loc *time.Location) (time.Time, time.Time, string) {
	if loc == nil {
		loc = time.UTC
	}
	if selected != "30d" && selected != "all" {
		selected = "7d"
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	switch selected {
	case "30d":
		return today.AddDate(0, 0, -29), now, selected
	case "all":
		return time.Unix(0, 0), now, selected
	default:
		return today.AddDate(0, 0, -6), now, selected
	}
}

func (vm *AIOverviewViewModel) Compact(value int64) string {
	abs := math.Abs(float64(value))
	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(value)/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(value)/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.1fK", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func (vm *AIOverviewViewModel) Dollars(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func (vm *AIOverviewViewModel) AIShare() int {
	ai := vm.Overview.Totals.AILineChanges()
	human := vm.HumanLineChanges()
	if ai+human == 0 {
		return 0
	}
	return int(math.Round(float64(ai) * 100 / float64(ai+human)))
}

func (vm *AIOverviewViewModel) HumanLineChanges() int64 {
	return vm.Overview.Totals.HumanLinesAdded + vm.Overview.Totals.HumanLinesDeleted
}

func (vm *AIOverviewViewModel) HumanShare() int {
	if vm.Overview.Totals.AILineChanges()+vm.HumanLineChanges() == 0 {
		return 0
	}
	return 100 - vm.AIShare()
}

func (vm *AIOverviewViewModel) CostCoverage() int {
	total := vm.Cost.PricedTokens + vm.Cost.UnpricedTokens
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(vm.Cost.PricedTokens) * 100 / float64(total)))
}

func (vm *AIOverviewViewModel) ChartBarHeight(bucket *AIChartBucket) int {
	if vm.MaxChartLines <= 0 {
		return 0
	}
	value := bucket.TotalLines()
	if value == 0 {
		return 0
	}
	return max(4, int(math.Round(float64(value)*100/float64(vm.MaxChartLines))))
}

func (vm *AIOverviewViewModel) ChartDate(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	if vm.SelectedRange == "all" {
		return parsed.Format("Jan '06")
	}
	return parsed.Format("01/02")
}

func (vm *AIOverviewViewModel) ChartPeriod(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	if vm.SelectedRange == "all" {
		return parsed.Format("January 2006")
	}
	return date
}

func (vm *AIOverviewViewModel) HalfChartLines() int64 {
	return vm.MaxChartLines / 2
}

func (vm *AIOverviewViewModel) ShowChartLabel(index int) bool {
	if vm.SelectedRange == "7d" {
		return true
	}
	if vm.SelectedRange == "all" {
		return index%3 == 0 || index == len(vm.ChartBuckets)-1
	}
	return index%5 == 0 || index == len(vm.ChartBuckets)-1
}
