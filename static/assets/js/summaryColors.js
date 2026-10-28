// Muted categorical hues remain distinct against the dark dashboard.
const baseColors = [
    '#789dbb', '#c18773', '#86a68f', '#a18bb9', '#bca475',
    '#75a8a6', '#b4819b', '#8c9cad', '#a4aa7d', '#b5938a'
]

const chartSemanticColors = {
    editor: {
        'claude code': '#c18773', 'claude': '#a57970',
        'codex vscode': '#789dbb', 'codex cli': '#8b9cbe', 'codex-wakapi': '#8b9cbe',
        'xcode': '#a18bb9', 'vs code': '#78a4b8', 'cursor': '#75a8a6',
        'android studio': '#86a68f', 'notion': '#a4aa7d', 'notes': '#bca475'
    },
    category: {
        'ai coding': '#a18bb9', 'coding': '#789dbb',
        'writing docs': '#86a68f', 'building': '#bca475',
        'debugging': '#c18773', 'writing tests': '#75a8a6', 'browsing': '#8c9cad'
    }
}

const chartPaletteOffsets = { project: 0, os: 0, editor: 1, language: 2, machine: 5, label: 7, category: 3 }

function chartColor(dimension, key, index) {
    const semantic = chartSemanticColors[dimension]?.[String(key).toLowerCase()]
    if (semantic) return semantic
    const offset = chartPaletteOffsets[dimension] || 0
    return baseColors[(index + offset) % baseColors.length]
}

function mergeChartItems(items, dimension) {
    if (dimension !== 'os' && dimension !== 'category') return items
    const merged = new Map()
    for (const item of items) {
        const rawKey = String(item.key)
        const key = dimension === 'os' && /^(mac|macos|mac os x|os x)$/i.test(rawKey) ? 'Mac' : rawKey
        const lookup = key.toLowerCase()
        const existing = merged.get(lookup)
        if (existing) existing.total += Number(item.total)
        else merged.set(lookup, { ...item, key, total: Number(item.total) })
    }
    return Array.from(merged.values())
}

Chart.defaults.color = '#b3b7c0'
Chart.defaults.borderColor = '#35353b'

