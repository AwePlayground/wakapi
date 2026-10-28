PetiteVue.createApp({
    $delimiters: ['${', '}'],
    activityChartSvg: '',
    get currentInterval() {
        const urlParams = new URLSearchParams(window.location.search)
        if (urlParams.has('interval')) return urlParams.get('interval')
        if (!urlParams.has('from') && !urlParams.has('to')) return 'today'
        return null
    },
    mounted({userId, version}) {
        fetch(`api/activity/chart/${userId}.svg?dark&noattr&v=${version}`)
            .then(res => res.text())
            .then(data => this.activityChartSvg = data)
    }
}).mount('#summary-page')
