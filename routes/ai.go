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
	ChartDays      []*services.AIDailyActivity
	MaxDailyLines  int64
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
	vm.ChartDays = overview.Daily
	if len(vm.ChartDays) > 30 {
		vm.ChartDays = vm.ChartDays[len(vm.ChartDays)-30:]
	}
	for _, day := range vm.ChartDays {
		lines := day.AILineChanges()
		if lines > vm.MaxDailyLines {
			vm.MaxDailyLines = lines
		}
	}
	if err := templates[conf.AIOverviewTemplate].Execute(w, vm); err != nil {
		conf.Log().Request(r).Error("failed to render AI overview", "error", err)
	}
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
	human := vm.Overview.Totals.HumanLinesAdded + vm.Overview.Totals.HumanLinesDeleted
	if ai+human == 0 {
		return 0
	}
	return int(math.Round(float64(ai) * 100 / float64(ai+human)))
}

func (vm *AIOverviewViewModel) CostCoverage() int {
	total := vm.Cost.PricedTokens + vm.Cost.UnpricedTokens
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(vm.Cost.PricedTokens) * 100 / float64(total)))
}

func (vm *AIOverviewViewModel) DailyBarHeight(day *services.AIDailyActivity) int {
	if vm.MaxDailyLines <= 0 {
		return 0
	}
	value := day.AILineChanges()
	if value == 0 {
		return 0
	}
	return max(4, int(math.Round(float64(value)*100/float64(vm.MaxDailyLines))))
}

func (vm *AIOverviewViewModel) ShortDay(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return parsed.Format("01/02")
}

func (vm *AIOverviewViewModel) HalfDailyLines() int64 {
	return vm.MaxDailyLines / 2
}

func (vm *AIOverviewViewModel) ShowDailyLabel(index int) bool {
	return vm.SelectedRange == "7d" || index%5 == 0 || index == len(vm.ChartDays)-1
}
