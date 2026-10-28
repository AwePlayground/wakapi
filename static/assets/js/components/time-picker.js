function TimePicker({fromDate, toDate, timeSelection}) {
    return {
        $template: '#time-picker-template',
        $delimiters: ['${', '}'],
        open: false,
        selectedInterval: new URLSearchParams(window.location.search).get('interval') || (new URLSearchParams(window.location.search).has('from') ? null : 'today'),
        presets: [
            {key: 'today', label: 'Today', hotkey: 'd'},
            {key: 'yesterday', label: 'Yesterday', hotkey: 'e'},
            {key: 'week', label: 'This week', hotkey: 'w'},
            {key: 'month', label: 'This month', hotkey: 'm'},
            {key: 'year', label: 'This year', hotkey: 'y'},
            {key: 'last_7_days', label: 'Past 7 days', hotkey: '1'},
            {key: 'last_30_days', label: 'Past 30 days', hotkey: '2'},
            {key: 'last_6_months', label: 'Past 6 months', hotkey: '3'},
            {key: 'last_12_months', label: 'Past 12 months', hotkey: '4'},
            {key: 'any', label: 'All time', hotkey: 'a'}
        ],
        fromDate: fromDate,
        toDate: toDate,
        customEndDate: new Date(Date.parse(toDate + 'T00:00:00Z') - 86400000).toISOString().slice(0, 10),
        timeSelection: timeSelection,
        toggle() { this.open = !this.open },
        displayRange() {
            const preset = this.presets.find(item => item.key === this.selectedInterval)
            if (preset) return preset.label
            const from = new Date(this.fromDate + 'T00:00:00Z')
            const end = new Date(this.toDate + 'T00:00:00Z')
            end.setUTCDate(end.getUTCDate() - 1)
            if (isNaN(from) || isNaN(end)) return this.timeSelection
            const format = date => date.toLocaleDateString('en-US', {month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC'})
            return this.fromDate === this.customEndDate ? format(from) : `${format(from)} – ${format(end)}`
        },
        intervalLink(interval) {
            const queryParams = new URLSearchParams(window.location.search)
            queryParams.delete('from')
            queryParams.delete('to')
            queryParams.delete('start')
            queryParams.set('interval', interval)
            return `summary?${queryParams.toString()}`
        },
        // Step to the previous (-1) or next (+1) period. `toDate` is exclusive (midnight after the range).
        // Named periods snap to the full previous / next period of their nominal length, anchored at their start, so a partial "this week" steps to the whole week before.
        // Custom ranges spanning whole calendar months step by months, all others by their length in days.
        shiftedRange(direction) {
            const parse = (s) => {
                const [y, m, d] = s.split('-').map(Number)
                return new Date(Date.UTC(y, m - 1, d))
            }
            const format = (date) => date.toISOString().slice(0, 10)
            const add = (date, unit, n) => {
                const d = new Date(date)
                if (unit === 'months') d.setUTCMonth(d.getUTCMonth() + n)
                else d.setUTCDate(d.getUTCDate() + n)
                return d
            }

            const from = parse(this.fromDate)
            const to = parse(this.toDate)
            if (isNaN(from) || isNaN(to) || to <= from) return null

            // keys mirror the aliases in models/interval.go
            const namedSteps = {
                'today': ['days', 1],
                'day': ['days', 1],
                'yesterday': ['days', 1],
                '24_hours': ['days', 1],
                'last_24_hours': ['days', 1],
                'last_day': ['days', 1],
                'week': ['days', 7],
                'last_week': ['days', 7],
                '7_days': ['days', 7],
                'last_7_days': ['days', 7],
                '14_days': ['days', 14],
                'last_14_days': ['days', 14],
                '30_days': ['days', 30],
                'last_30_days': ['days', 30],
                'month': ['months', 1],
                'last_month': ['months', 1],
                '6_months': ['months', 6],
                'last_6_months': ['months', 6],
                'year': ['months', 12],
                '12_months': ['months', 12],
                'last_12_months': ['months', 12],
                'last_year': ['months', 12],
            }
            const params = new URLSearchParams(window.location.search)
            const interval = params.get('interval') || params.get('start')  // start is a server-side alias for interval (see helpers.ParseSummaryParams)
            if (['any', 'all_time', 'All Time'].includes(interval)) return null  // no shifting for all time interval

            let step = namedSteps[interval]
            if (!step) {
                if (from.getUTCDate() === 1 && to.getUTCDate() === 1) {
                    step = ['months', (to.getUTCFullYear() - from.getUTCFullYear()) * 12 + (to.getUTCMonth() - from.getUTCMonth())]
                } else {
                    step = ['days', Math.round((to - from) / 86400000)]
                }
            }

            const [unit, size] = step
            const newFrom = add(from, unit, direction * size)
            return {from: format(newFrom), to: format(add(newFrom, unit, size))}
        },
        canShift(direction) {
            const range = this.shiftedRange(direction)
            if (!range) return false
            if (direction < 0) return true
            const now = new Date()
            const today = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())).toISOString().slice(0, 10)
            return range.from <= today // don't navigate into ranges that start in the future
        },
        shiftLink(direction) {
            const range = this.shiftedRange(direction)
            if (!range) return null
            const query = new URLSearchParams(window.location.search)
            query.delete('interval')
            query.delete('start')
            query.set('from', range.from)
            query.set('to', range.to)
            return `summary?${query.toString()}`
        },
        onSubmit() {
            const query = new URLSearchParams(window.location.search)
            query.delete('interval')
            query.delete('start')
            query.set('from', this.fromDate)
            const endExclusive = new Date(this.customEndDate + 'T00:00:00Z')
            endExclusive.setUTCDate(endExclusive.getUTCDate() + 1)
            query.set('to', endExclusive.toISOString().slice(0, 10))
            this.open = false
            window.location.search = query.toString()
        },
        mounted() {
            const root = document.getElementById('summary-date-picker')
            document.addEventListener('pointerdown', event => {
                if (!root.contains(event.target)) this.open = false
            })
            document.addEventListener('keydown', event => {
                if (event.key === 'Escape' && this.open) {
                    this.open = false
                    document.getElementById('summary-date-trigger')?.focus()
                    return
                }
                if (!this.open || event.altKey || event.ctrlKey || event.metaKey || event.target !== document.body) return
                const preset = this.presets.find(item => item.hotkey === event.key.toLowerCase())
                if (preset) window.location.href = this.intervalLink(preset.key)
            })
        }
    }
}
