package services

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	svg "github.com/ajstarks/svgo/float"
	"github.com/alitto/pond/v2"
	"github.com/duke-git/lancet/v2/condition"
	"github.com/duke-git/lancet/v2/datetime"
	"github.com/muety/wakapi/config"
	"github.com/muety/wakapi/helpers"
	"github.com/muety/wakapi/models"
	"github.com/muety/wakapi/utils"
	"github.com/patrickmn/go-cache"
)

const (
	gridRows         = 7
	cellWidth        = 14
	cellHeight       = 14
	cellSpacing      = 3
	noActivityDark   = "#30343D"
	noActivityLight  = "#E8EDF1"
	textDark         = "#D1D5DB"
	textLight        = "#37474F"
	activityCacheTTL = 15 * time.Minute
)

var activityColorsDark = [...]string{"#4F6886", "#6985A4", "#88A8C3", "#ACC6DC"}
var activityColorsLight = [...]string{"#D1E0EC", "#AAC5DA", "#7EA9C8", "#507FA6"}

func activityFillColor(total, maxTotal time.Duration, darkTheme bool) string {
	if total <= 0 {
		return condition.Ternary(darkTheme, noActivityDark, noActivityLight)
	}
	colors := activityColorsLight[:]
	if darkTheme {
		colors = activityColorsDark[:]
	}
	if maxTotal <= 0 {
		return colors[0]
	}
	level := int(math.Sqrt(float64(total)/float64(maxTotal)) * float64(len(colors)))
	if level >= len(colors) {
		level = len(colors) - 1
	}
	return colors[level]
}

type ActivityService struct {
	config         *config.Config
	cache          *cache.Cache
	summaryService ISummaryService
}

func NewActivityService(summaryService ISummaryService) *ActivityService {
	return &ActivityService{
		config:         config.Get(),
		cache:          cache.New(activityCacheTTL, activityCacheTTL),
		summaryService: summaryService,
	}
}

// GetChart generates an activity chart for a given user and the given time interval, similar to GitHub's contribution timeline. See https://github.com/muety/wakapi/issues/12.
// Please note: currently, only yearly charts ("last_12_months") are supported. However, we could fairly easily restructure this to support dynamic intervals.
func (s *ActivityService) GetChart(user *models.User, interval *models.IntervalKey, darkTheme, hideAttribution, skipCache bool) (string, error) {
	cacheKey := fmt.Sprintf("chart_%s_%s_%v_%v", user.ID, (*interval)[0], darkTheme, hideAttribution)
	if result, found := s.cache.Get(cacheKey); found && !skipCache {
		return result.(string), nil
	}

	switch interval {
	case models.IntervalPast12Months:
		chart, err := s.getChartPastYear(user, darkTheme, hideAttribution)
		if err == nil {
			s.cache.SetDefault(cacheKey, chart) // TODO: cache compressed?
		}
		return chart, err
	default:
		return "", errors.New("unsupported interval")
	}
}

func (s *ActivityService) getChartPastYear(user *models.User, darkTheme, hideAttribution bool) (string, error) {
	err, from, to := helpers.ResolveIntervalTZ(models.IntervalPast12Months, user.TZ(), user.StartOfWeekDay())
	// TODO: I am not sure if we have to handle startOfWeekDay here, but it seems like we do not need to.
	from = datetime.BeginOfWeek(from, time.Monday)
	if err != nil {
		return "", err
	}

	intervals := utils.SplitRangeByDays(from, to)
	summaries := make([]*models.Summary, len(intervals))

	wp := pond.NewPool(utils.HalfCPUs())
	mut := sync.RWMutex{}

	// fetch summaries
	for i, interval := range intervals {
		i := i // https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable
		interval := interval

		wp.Submit(func() {
			summary, err := s.summaryService.Retrieve(interval[0], interval[1], user, nil, nil)
			if err != nil {
				config.Log().Warn("failed to retrieve summary for activity chart", "userID", user.ID, "from", from, "to", to)
				summary = models.NewEmptySummary()
				summary.FromTime = models.CustomTime(from)
				summary.ToTime = models.CustomTime(to)
				summary.UserID = user.ID
				summary.User = user
			}

			mut.Lock()
			summaries[i] = summary
			mut.Unlock()
		})
	}

	wp.StopAndWait()

	maxTotal := models.Summaries(summaries).MaxTotalTime()

	var (
		colorText         = condition.Ternary[bool, string](darkTheme, textDark, textLight)
		gridCols          = math.Ceil(float64(len(summaries)) / float64(gridRows))
		w         float64 = gridCols*cellWidth + gridCols*cellSpacing
		h         float64 = gridRows*cellHeight + 25 + 24 + 5 + 5 + gridRows*cellSpacing
	)

	// regenerate svg
	buf := &bytes.Buffer{}

	canvas := svg.New(buf)
	canvas.Start(w, h)
	canvas.Style("text/css",
		fmt.Sprintf("text { font-family: 'IBM Plex Sans', sans-serif; font-size: 0.85rem; font-weight: 400; fill: %s; }", colorText),
		fmt.Sprintf("rect { fill-opacity: 1; rx: 3px; ry: 3px; }"),
		fmt.Sprintf("rect:hover { filter: brightness(0.9) }"),
	)

	canvas.Text(0, 15, fmt.Sprintf("%s to %s", helpers.FormatDateHuman(summaries[0].FromTime.T()), helpers.FormatDateHuman(summaries[len(summaries)-1].ToTime.T())))

	for i, s := range summaries {
		total := s.TotalTime()
		fillColor := activityFillColor(total, maxTotal, darkTheme)

		canvas.Group()
		canvas.Title(fmt.Sprintf("%s on %s", helpers.FmtWakatimeDuration(total), helpers.FormatDateHuman(s.FromTime.T())))
		canvas.Rect(float64(i/gridRows)*(cellWidth+cellSpacing), 25+float64((i%gridRows)*(cellHeight+cellSpacing)), cellWidth, cellHeight, fmt.Sprintf("fill: %s", fillColor))
		canvas.Gend()
	}

	if !hideAttribution {
		canvas.Group()
		canvas.Title("Wakapi.dev")
		canvas.Image(w-60, h-24, 60, 24, "https://wakapi.dev/assets/images/logo-gh.svg")
		canvas.Gend()
	}

	canvas.End()

	return buf.String(), nil
}
