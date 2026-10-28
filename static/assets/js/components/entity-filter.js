function SummaryFilters({options}) {
    const definitions = [
        {type: 'project', label: 'Project'},
        {type: 'language', label: 'Language'},
        {type: 'machine', label: 'Machine'},
        {type: 'label', label: 'Label'},
        {type: 'category', label: 'Category'}
    ]
    const query = new URLSearchParams(window.location.search)
    const selected = Object.fromEntries(definitions.map(({type}) => [type, query.get(type) === '-' ? 'unknown' : query.get(type)]))
    return {
        $template: '#summary-filters-template',
        $delimiters: ['${', '}'],
        definitions,
        options,
        selected,
        open: false,
        activeType: 'project',
        search: '',
        selectedEntries() {
            return this.definitions.filter(({type}) => this.selected[type]).map(({type, label}) => ({type, label, value: this.selected[type]}))
        },
        activeLabel() {
            return this.definitions.find(({type}) => type === this.activeType)?.label || 'Filter'
        },
        filteredOptions() {
            const needle = this.search.trim().toLocaleLowerCase()
            const values = this.options[this.activeType] || []
            return values.filter(value => String(value).toLocaleLowerCase().includes(needle)).slice(0, 100)
        },
        toggle() {
            this.open = !this.open
            if (this.open) this.$nextTick(() => document.getElementById('summary-filter-search')?.focus())
        },
        selectType(type) {
            this.activeType = type
            this.search = ''
            this.$nextTick(() => document.getElementById('summary-filter-search')?.focus())
        },
        update(type, value) {
            const query = new URLSearchParams(window.location.search)
            if (value) query.set(type, value === 'unknown' ? '-' : value)
            else query.delete(type)
            window.location.search = query.toString()
        },
        clearAll() {
            const query = new URLSearchParams(window.location.search)
            this.definitions.forEach(({type}) => query.delete(type))
            window.location.search = query.toString()
        },
        mounted() {
            const root = document.getElementById('summary-filters')
            document.addEventListener('pointerdown', event => {
                if (!root.contains(event.target)) this.open = false
            })
            document.addEventListener('keydown', event => {
                if (event.key === 'Escape' && this.open) {
                    this.open = false
                    document.getElementById('summary-filter-trigger')?.focus()
                }
            })
        }
    }
}
