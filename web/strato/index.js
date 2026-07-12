// Charts instances
let historyChart = null;
let dailyChart = null;

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initCharts();
    startLiveStream();
    loadLatestTotals(); // Lade aktuelle Zählerstände aus DB
    loadHistoryData(); // load current day by default
    loadDailyUsageData();
    updateChartsForTheme(getCurrentTheme());
});

// Theme Management
function getCurrentTheme() {
    return localStorage.getItem('theme') || 'dark';
}

function initTheme() {
    const theme = getCurrentTheme();
    document.documentElement.setAttribute('data-theme', theme);
    updateThemeToggleUI(theme);

    const toggleBtn = document.getElementById('theme-toggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', () => {
            const current = getCurrentTheme();
            const next = current === 'dark' ? 'light' : 'dark';
            localStorage.setItem('theme', next);
            document.documentElement.setAttribute('data-theme', next);
            updateThemeToggleUI(next);
            updateChartsForTheme(next);
        });
    }
}

function updateThemeToggleUI(theme) {
    const iconSpan = document.getElementById('theme-toggle-icon');
    const toggleBtn = document.getElementById('theme-toggle');
    if (iconSpan) {
        iconSpan.innerText = theme === 'dark' ? '☀️' : '🌙';
    }
    if (toggleBtn) {
        toggleBtn.title = theme === 'dark' ? 'Zu hellem Design wechseln' : 'Zu dunklem Design wechseln';
    }
}

function updateChartsForTheme(theme) {
    const gridColor = theme === 'light' ? 'rgba(0, 0, 0, 0.05)' : 'rgba(255, 255, 255, 0.03)';
    const textColor = theme === 'light' ? '#64748b' : '#9ca3af';

    if (historyChart) {
        historyChart.options.scales.x.grid.color = gridColor;
        historyChart.options.scales.y.grid.color = gridColor;
        historyChart.options.scales.x.ticks.color = textColor;
        historyChart.options.scales.y.ticks.color = textColor;
        historyChart.options.plugins.legend.labels.color = textColor;
        historyChart.update();
    }
    if (dailyChart) {
        dailyChart.options.scales.x.grid.color = gridColor;
        dailyChart.options.scales.y.grid.color = gridColor;
        dailyChart.options.scales.x.ticks.color = textColor;
        dailyChart.options.scales.y.ticks.color = textColor;
        dailyChart.options.plugins.legend.labels.color = textColor;
        dailyChart.update();
    }
}

// SSE Connection disabled on Strato (using static database)
function startLiveStream() {
    document.getElementById('sml-status').className = "badge badge-success";
    document.getElementById('sml-status').innerText = "SML: Online (Backup)";
    document.getElementById('sma-status').className = "badge badge-success";
    document.getElementById('sma-status').innerText = "SMA: Online (Backup)";
}

// --- Helper: Update badge status ---
function updateBadge(id, active, prefix) {
    const badge = document.getElementById(id);
    if (active) {
        badge.className = "badge badge-success";
        badge.innerText = `${prefix}: verbunden`;
    } else {
        badge.className = "badge badge-error";
        badge.innerText = `${prefix}: inaktiv`;
    }
}

// --- Helper: Update meter detail values (SML or SMA card) ---
function updateMeterValues(idPrefix, data, keyPrefix, phaseOrder) {
    document.getElementById(`${idPrefix}-import-val`).innerText = `${(data[`${keyPrefix}_energy_import`] / 1000).toFixed(2)} kWh`;
    document.getElementById(`${idPrefix}-export-val`).innerText = `${(data[`${keyPrefix}_energy_export`] / 1000).toFixed(2)} kWh`;

    const totalPower = data[`${keyPrefix}_power_import`] > 0 ? data[`${keyPrefix}_power_import`] : -data[`${keyPrefix}_power_export`];
    document.getElementById(`${idPrefix}-power-total-val`).innerText = `${totalPower > 0 ? "+" : ""}${Math.round(totalPower)} W`;

    const [p1, p2, p3] = phaseOrder;
    document.getElementById(`${idPrefix}-power-phases-val`).innerText = `${Math.round(data[`${keyPrefix}_power_${p1}`])} / ${Math.round(data[`${keyPrefix}_power_${p2}`])} / ${Math.round(data[`${keyPrefix}_power_${p3}`])} W`;
    document.getElementById(`${idPrefix}-voltage-phases-val`).innerText = `${data[`${keyPrefix}_voltage_${p1}`].toFixed(1)} / ${data[`${keyPrefix}_voltage_${p2}`].toFixed(1)} / ${data[`${keyPrefix}_voltage_${p3}`].toFixed(1)} V`;
    document.getElementById(`${idPrefix}-current-phases-val`).innerText = `${data[`${keyPrefix}_current_${p1}`].toFixed(2)} / ${data[`${keyPrefix}_current_${p2}`].toFixed(2)} / ${data[`${keyPrefix}_current_${p3}`].toFixed(2)} A`;
    document.getElementById(`${idPrefix}-freq-val`).innerText = `${data[`${keyPrefix}_frequency`].toFixed(2)} Hz`;
    document.getElementById(`${idPrefix}-interval-val`).innerText = `${data[`${keyPrefix}_interval_ms`] || 0} ms`;
}

// --- Helper: Determine power direction and value ---
function getPowerState(data, keyPrefix) {
    if (data[`${keyPrefix}_power_import`] > 0) {
        return { value: data[`${keyPrefix}_power_import`], isImport: true };
    } else if (data[`${keyPrefix}_power_export`] > 0) {
        return { value: data[`${keyPrefix}_power_export`], isImport: false };
    }
    return { value: 0, isImport: true };
}

// --- Helper: Apply import/export visual state to power display elements ---
function applyPowerState(elements, isImport) {
    const mode = isImport ? 'import' : 'export';
    const label = isImport ? 'Netzbezug' : 'Netzeinspeisung';
    const colorClass = isImport ? 'import-text' : 'export-text';

    if (elements.pulse) {
        elements.pulse.className = `pulse-dot active-${mode}`;
    }
    if (elements.label) {
        elements.label.innerText = label;
        elements.label.className = `${elements.labelBase} ${colorClass}`;
    }
    if (elements.val) {
        elements.val.className = `${elements.valBase} ${colorClass}`;
    }
    if (elements.card) {
        elements.card.className = `card live-card ${isImport ? 'import' : 'surplus'}`;
    }
    if (elements.bar) {
        elements.bar.style.backgroundColor = isImport ? '#f59e0b' : '#10b981';
    }
}

// --- Helper: Reset power display to inactive/no-data state ---
function resetPowerState(elements) {
    if (elements.pulse) {
        elements.pulse.className = "pulse-dot";
    }
    if (elements.val) {
        elements.val.innerText = "0 W";
        elements.val.className = elements.valBase;
        elements.val.style.color = "#9ca3af";
    }
    if (elements.label) {
        elements.label.innerText = "Keine Daten";
        elements.label.className = elements.labelBase;
        elements.label.style.color = "";
    }
    if (elements.card) {
        elements.card.className = "card live-card";
    }
    if (elements.bar) {
        elements.bar.style.width = "0%";
    }
}

let lastHistoryUpdateMinute = -1;

// Update DOM elements with live data
function updateLiveDashboard(data) {
    const sml = data.sml || {};
    const sma = data.sma || {};

    const now = new Date();
    document.getElementById('last-update-time').innerText = now.toLocaleTimeString();

    // Auto-refresh history chart every 5 minutes if viewing 'Heute'
    const currentMinute = now.getMinutes();
    if (currentMinute % 5 === 0 && currentMinute !== lastHistoryUpdateMinute) {
        lastHistoryUpdateMinute = currentMinute;
        
        const isTrackingToday = document.getElementById('btn-date-today').classList.contains('active');
        if (isTrackingToday) {
            if (currentHistoryDate.toDateString() !== now.toDateString()) {
                currentHistoryDate = new Date(); // Roll over to new day
                document.getElementById('history-chart-title').innerText = 'Leistungsverlauf (Heute)';
            }
            loadHistoryData();
        }
    }

    const smlActive = sml.sml_active;
    const smaActive = sma.sma_active;

    // Badge updates
    updateBadge('sml-status', smlActive, 'SML');
    updateBadge('sma-status', smaActive, 'SMA');

    // Meter detail values
    if (smlActive) {
        updateMeterValues('sml', sml, 'sml', ['l1', 'l2', 'l3']);
    }
    if (smaActive) {
        updateMeterValues('sma', sma, 'sma', ['l1', 'l3', 'l2']);
    }

    // SML Power Display (Primary)
    const smlElements = {
        card: document.getElementById('live-power-card'),
        pulse: document.getElementById('sml-power-pulse'),
        val: document.getElementById('sml-power-val'),
        label: document.getElementById('sml-power-label'),
        bar: document.getElementById('sml-power-bar-fill'),
        valBase: 'power-value',
        labelBase: 'power-label',
    };

    if (smlActive) {
        const { value, isImport } = getPowerState(sml, 'sml');
        smlElements.val.innerText = `${Math.round(value)} W`;
        smlElements.bar.style.width = `${Math.min((value / 6000) * 100, 100)}%`;
        applyPowerState(smlElements, isImport);
    } else {
        resetPowerState(smlElements);
    }

    // SMA Power Display (Secondary)
    const smaElements = {
        pulse: document.getElementById('sma-power-pulse'),
        val: document.getElementById('sma-power-val'),
        label: document.getElementById('sma-power-label'),
        valBase: 'sma-value',
        labelBase: 'sma-status-text',
    };

    if (smaActive) {
        const { value, isImport } = getPowerState(sma, 'sma');
        smaElements.val.innerText = `${Math.round(value)} W`;
        applyPowerState(smaElements, isImport);
    } else {
        resetPowerState(smaElements);
    }
}

// --- Shared Chart styling ---
const baseChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
        legend: {
            labels: { color: '#9ca3af', font: { family: 'Inter' } }
        }
    },
    scales: {
        x: {
            grid: { color: 'rgba(255, 255, 255, 0.03)' },
            ticks: { color: '#9ca3af', font: { family: 'Inter', size: 10 } }
        },
        y: {
            grid: { color: 'rgba(255, 255, 255, 0.03)' },
            ticks: { color: '#9ca3af', font: { family: 'Inter', size: 10 } }
        }
    }
};

let currentHistoryDate = new Date();
let currentHistorySource = 'SMA';
let currentHistoryMode = 'both';

function setHistorySource(src) {
    currentHistorySource = src;
    document.getElementById('btn-src-sma').classList.toggle('active', src === 'SMA');
    document.getElementById('btn-src-sml').classList.toggle('active', src === 'SML');
    loadHistoryData();
}

function setHistoryMode(mode) {
    currentHistoryMode = mode;
    document.getElementById('btn-mode-both').classList.toggle('active', mode === 'both');
    document.getElementById('btn-mode-import').classList.toggle('active', mode === 'import');
    document.getElementById('btn-mode-export').classList.toggle('active', mode === 'export');
    updateHistoryChartVisibility();
}

function navigateHistory(offset) {
    if (offset === 'today') {
        currentHistoryDate = new Date();
    } else {
        currentHistoryDate.setDate(currentHistoryDate.getDate() + offset);
    }
    
    // Prevent going into the future
    if (currentHistoryDate > new Date()) {
        currentHistoryDate = new Date();
    }
    
    const today = new Date();
    const isToday = currentHistoryDate.toDateString() === today.toDateString();
    document.getElementById('btn-date-today').classList.toggle('active', isToday);
    
    // Update title
    const titleDate = isToday ? "Heute" : currentHistoryDate.toLocaleDateString();
    document.getElementById('history-chart-title').innerText = `Leistungsverlauf (${titleDate})`;
    
    loadHistoryData();
}

// Generates an array of time strings from "00:00" to "23:55"
function generateDayLabels() {
    const labels = [];
    for (let h = 0; h < 24; h++) {
        for (let m = 0; m < 60; m += 5) {
            labels.push(`${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`);
        }
    }
    return labels;
}

function updateHistoryChartVisibility() {
    const mode = currentHistoryMode;
    const showImport = mode === 'both' || mode === 'import';
    const showExport = mode === 'both' || mode === 'export';
    
    for (let i = 0; i < 3; i++) {
        historyChart.setDatasetVisibility(i, showImport);
        historyChart.setDatasetVisibility(i + 3, showExport);
    }
    historyChart.update();
}

// Chart initialization
function initCharts() {
    const dayLabels = generateDayLabels();
    historyChart = new Chart(document.getElementById('historyChart').getContext('2d'), {
        type: 'line',
        data: {
            labels: dayLabels,
            datasets: [
                { label: 'Bezug (Max)', borderColor: 'transparent', backgroundColor: 'rgba(245, 158, 11, 0.2)', borderWidth: 0, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false },
                { label: 'Bezug (Avg)', borderColor: 'transparent', backgroundColor: 'rgba(245, 158, 11, 0.5)', borderWidth: 0, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false },
                { label: 'Bezug (Min)', borderColor: '#f59e0b', backgroundColor: 'rgba(245, 158, 11, 1.0)', borderWidth: 2, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false },
                { label: 'Einspeisung (Max)', borderColor: 'transparent', backgroundColor: 'rgba(16, 185, 129, 0.2)', borderWidth: 0, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false },
                { label: 'Einspeisung (Avg)', borderColor: 'transparent', backgroundColor: 'rgba(16, 185, 129, 0.5)', borderWidth: 0, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false },
                { label: 'Einspeisung (Min)', borderColor: '#10b981', backgroundColor: 'rgba(16, 185, 129, 1.0)', borderWidth: 2, pointRadius: 0, fill: 'origin', data: [], tension: 0.3, spanGaps: false }
            ]
        },
        options: {
            ...baseChartOptions,
            plugins: {
                ...baseChartOptions.plugins,
                tooltip: { mode: 'index', intersect: false }
            },
            scales: {
                ...baseChartOptions.scales,
                x: { ...baseChartOptions.scales.x, min: '00:00', max: '23:55' }
            }
        }
    });

    dailyChart = new Chart(document.getElementById('dailyChart').getContext('2d'), {
        type: 'bar',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Verbrauch (kWh)',
                    backgroundColor: 'rgba(245, 158, 11, 0.75)',
                    borderColor: '#f59e0b',
                    borderWidth: 1,
                    borderRadius: 4,
                    data: []
                },
                {
                    label: 'Einspeisung (kWh)',
                    backgroundColor: 'rgba(16, 185, 129, 0.75)',
                    borderColor: '#10b981',
                    borderWidth: 1,
                    borderRadius: 4,
                    data: []
                }
            ]
        },
        options: {
            ...baseChartOptions,
            scales: {
                ...baseChartOptions.scales,
                x: { ...baseChartOptions.scales.x, grid: { display: false } }
            }
        }
    });
}

// Load historical data for Line Chart
function loadHistoryData() {
    const year = currentHistoryDate.getFullYear();
    const month = (currentHistoryDate.getMonth() + 1).toString().padStart(2, '0');
    const day = currentHistoryDate.getDate().toString().padStart(2, '0');
    const dateStr = `${year}-${month}-${day}`;

    fetch(`api.php?endpoint=history&date=${dateStr}`)
        .then(res => res.json())
        .then(data => {
            // Initialize arrays for 288 slots with null (for spanGaps: false)
            const impMax = Array(288).fill(null);
            const impAvg = Array(288).fill(null);
            const impMin = Array(288).fill(null);
            const expMax = Array(288).fill(null);
            const expAvg = Array(288).fill(null);
            const expMin = Array(288).fill(null);

            if (data && data.length > 0) {
                data.forEach(item => {
                    const date = new Date(item.timestamp);
                    const h = date.getHours();
                    const m = date.getMinutes();
                    const idx = (h * 12) + Math.floor(m / 5);

                    if (idx >= 0 && idx < 288) {
                        if (currentHistorySource === 'SMA') {
                            impMax[idx] = item.sma_power_import_w_max;
                            impAvg[idx] = item.sma_power_import_w_avg;
                            impMin[idx] = item.sma_power_import_w_min;
                            expMax[idx] = item.sma_power_export_w_max;
                            expAvg[idx] = item.sma_power_export_w_avg;
                            expMin[idx] = item.sma_power_export_w_min;
                        } else {
                            impMax[idx] = item.sml_power_import_w_max;
                            impAvg[idx] = item.sml_power_import_w_avg;
                            impMin[idx] = item.sml_power_import_w_min;
                            expMax[idx] = item.sml_power_export_w_max;
                            expAvg[idx] = item.sml_power_export_w_avg;
                            expMin[idx] = item.sml_power_export_w_min;
                        }
                    }
                });
            }

            historyChart.data.datasets[0].data = impMax;
            historyChart.data.datasets[1].data = impAvg;
            historyChart.data.datasets[2].data = impMin;
            historyChart.data.datasets[3].data = expMax;
            historyChart.data.datasets[4].data = expAvg;
            historyChart.data.datasets[5].data = expMin;
            
            historyChart.update();
            updateHistoryChartVisibility();
        })
        .catch(err => console.error("Failed to load history chart data", err));
}

let currentDailyScope = 'week';
let currentDailySource = 'SMA';
let currentDailyMode = 'both';
let currentDailyDate = new Date();

function setDailyScope(scope) {
    currentDailyScope = scope;
    ['day', 'week', '10days', 'month', 'year'].forEach(s => {
        document.getElementById(`btn-daily-scope-${s}`).classList.toggle('active', s === scope);
    });
    // Reset date to today when changing scope to avoid weird jumps
    currentDailyDate = new Date();
    loadDailyUsageData();
}

function setDailySource(src) {
    currentDailySource = src;
    document.getElementById('btn-daily-src-sma').classList.toggle('active', src === 'SMA');
    document.getElementById('btn-daily-src-sml').classList.toggle('active', src === 'SML');
    loadDailyUsageData(); // Data exists for both, but backend delivers both anyway. We just remap.
}

function setDailyMode(mode) {
    currentDailyMode = mode;
    document.getElementById('btn-daily-mode-both').classList.toggle('active', mode === 'both');
    document.getElementById('btn-daily-mode-import').classList.toggle('active', mode === 'import');
    document.getElementById('btn-daily-mode-export').classList.toggle('active', mode === 'export');
    updateDailyChartVisibility();
}

function navigateDaily(offset) {
    if (offset === 'today') {
        currentDailyDate = new Date();
    } else {
        if (currentDailyScope === 'day') currentDailyDate.setDate(currentDailyDate.getDate() + offset);
        else if (currentDailyScope === 'week') currentDailyDate.setDate(currentDailyDate.getDate() + (offset * 7));
        else if (currentDailyScope === '10days') currentDailyDate.setDate(currentDailyDate.getDate() + (offset * 10));
        else if (currentDailyScope === 'month') currentDailyDate.setMonth(currentDailyDate.getMonth() + offset);
        else if (currentDailyScope === 'year') currentDailyDate.setFullYear(currentDailyDate.getFullYear() + offset);
    }
    
    if (currentDailyDate > new Date()) currentDailyDate = new Date();
    loadDailyUsageData();
}

function updateDailyChartVisibility() {
    dailyChart.setDatasetVisibility(0, currentDailyMode === 'both' || currentDailyMode === 'import');
    dailyChart.setDatasetVisibility(1, currentDailyMode === 'both' || currentDailyMode === 'export');
    dailyChart.update();
}

let lastDailyRawData = [];

function loadDailyUsageData() {
    // 1. Calculate Date Range
    let startD = new Date(currentDailyDate);
    let endD = new Date(currentDailyDate);
    let url = '';
    let isToday = false;

    if (currentDailyScope === 'year') {
        const year = currentDailyDate.getFullYear();
        url = `api.php?endpoint=monthly&year=${year}`;
        isToday = year === new Date().getFullYear();
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${year})`;
    } else if (currentDailyScope === 'day') {
        const dateStr = `${startD.getFullYear()}-${(startD.getMonth()+1).toString().padStart(2,'0')}-${startD.getDate().toString().padStart(2,'0')}`;
        url = `api.php?endpoint=history&date=${dateStr}`;
        isToday = dateStr === new Date().toLocaleDateString('sv-SE'); // YYYY-MM-DD in local time
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${startD.toLocaleDateString([], { day: 'numeric', month: 'short', year: 'numeric' })})`;
    } else {
        if (currentDailyScope === 'week') {
            // Monday to Sunday of the currentDailyDate week
            const day = startD.getDay() || 7; // Get current day number, converting Sun(0) to 7
            startD.setDate(startD.getDate() - day + 1); // Monday
            endD = new Date(startD);
            endD.setDate(startD.getDate() + 6); // Sunday
        } else if (currentDailyScope === '10days') {
            startD.setDate(endD.getDate() - 9);
        } else if (currentDailyScope === 'month') {
            startD.setDate(1);
            endD = new Date(startD.getFullYear(), startD.getMonth() + 1, 0); // Last day of month
        }
        
        isToday = endD >= new Date() && startD <= new Date();

        const sStr = `${startD.getFullYear()}-${(startD.getMonth()+1).toString().padStart(2,'0')}-${startD.getDate().toString().padStart(2,'0')}`;
        const eStr = `${endD.getFullYear()}-${(endD.getMonth()+1).toString().padStart(2,'0')}-${endD.getDate().toString().padStart(2,'0')}`;
        url = `api.php?endpoint=daily&start=${sStr}&end=${eStr}`;

        const fmt = { day: 'numeric', month: 'short' };
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${startD.toLocaleDateString([], fmt)} - ${endD.toLocaleDateString([], fmt)})`;
    }

    document.getElementById('btn-daily-date-today').classList.toggle('active', isToday);

    // 2. Fetch Data
    fetch(url)
        .then(res => res.json())
        .then(data => {
            lastDailyRawData = data || [];
            renderDailyChart();
            
            // Update the Top Row values if we are looking at the current period
            if (isToday && lastDailyRawData.length > 0) {
                let smlConsumed = 0, smlDelivered = 0, smaConsumed = 0, smaDelivered = 0;
                if (currentDailyScope === 'day') {
                    const firstVal = lastDailyRawData[0];
                    const lastVal = lastDailyRawData[lastDailyRawData.length - 1];
                    smlConsumed = ((lastVal.sml_import_wh || 0) - (firstVal.sml_import_wh || 0)) / 1000.0;
                    smlDelivered = ((lastVal.sml_export_wh || 0) - (firstVal.sml_export_wh || 0)) / 1000.0;
                    smaConsumed = ((lastVal.sma_import_wh || 0) - (firstVal.sma_import_wh || 0)) / 1000.0;
                    smaDelivered = ((lastVal.sma_export_wh || 0) - (firstVal.sma_export_wh || 0)) / 1000.0;
                } else {
                    const todayVal = lastDailyRawData[lastDailyRawData.length - 1];
                    smlConsumed = (todayVal.sml_consumed_wh || 0) / 1000.0;
                    smlDelivered = (todayVal.sml_delivered_wh || 0) / 1000.0;
                    smaConsumed = (todayVal.sma_consumed_wh || 0) / 1000.0;
                    smaDelivered = (todayVal.sma_delivered_wh || 0) / 1000.0;
                }
                if (document.getElementById('sml-today-consumed')) {
                    document.getElementById('sml-today-consumed').innerText = `${smlConsumed.toFixed(2)} kWh`;
                    document.getElementById('sml-today-delivered').innerText = `${smlDelivered.toFixed(2)} kWh`;
                    document.getElementById('sma-today-consumed').innerText = `${smaConsumed.toFixed(2)} kWh`;
                    document.getElementById('sma-today-delivered').innerText = `${smaDelivered.toFixed(2)} kWh`;
                }
            }
        })
        .catch(err => console.error("Failed to load daily usage chart data", err));
}

function renderDailyChart() {
    const labels = [];
    const consumed = [];
    const delivered = [];

    // Pre-fill labels for full scope
    if (currentDailyScope === 'day') {
        for (let h = 0; h < 24; h++) {
            labels.push(`${h.toString().padStart(2, '0')}:00`);
            consumed.push(null);
            delivered.push(null);
        }
    } else if (currentDailyScope === 'week') {
        const d = new Date(currentDailyDate);
        const day = d.getDay() || 7;
        d.setDate(d.getDate() - day + 1);
        for (let i=0; i<7; i++) {
            labels.push(new Date(d).toLocaleDateString([], { weekday: 'short', day: 'numeric' }));
            consumed.push(null); delivered.push(null);
            d.setDate(d.getDate() + 1);
        }
    } else if (currentDailyScope === '10days') {
        const d = new Date(currentDailyDate);
        d.setDate(d.getDate() - 9);
        for (let i=0; i<10; i++) {
            labels.push(new Date(d).toLocaleDateString([], { weekday: 'short', day: 'numeric' }));
            consumed.push(null); delivered.push(null);
            d.setDate(d.getDate() + 1);
        }
    } else if (currentDailyScope === 'month') {
        const d = new Date(currentDailyDate.getFullYear(), currentDailyDate.getMonth(), 1);
        const daysInMonth = new Date(currentDailyDate.getFullYear(), currentDailyDate.getMonth() + 1, 0).getDate();
        for (let i=0; i<daysInMonth; i++) {
            labels.push(`${i+1}.`);
            consumed.push(null); delivered.push(null);
        }
    } else if (currentDailyScope === 'year') {
        const months = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez'];
        for (let i=0; i<12; i++) {
            labels.push(months[i]);
            consumed.push(null); delivered.push(null);
        }
    }

    let consumedBgColors = 'rgba(245, 158, 11, 0.75)';
    let consumedBorderColors = '#f59e0b';
    let deliveredBgColors = 'rgba(16, 185, 129, 0.75)';
    let deliveredBorderColors = '#10b981';

    if (currentDailyScope === 'day') {
        // Daily scope: aggregate 5-minute readings into 24 hours using delta math
        if (lastDailyRawData.length > 0) {
            // Find the first valid (non-zero) baseline readings of the day
            let prevCon = 0;
            let prevDel = 0;
            for (let i = 0; i < lastDailyRawData.length; i++) {
                const con = (currentDailySource === 'SMA' ? lastDailyRawData[i].sma_import_wh : lastDailyRawData[i].sml_import_wh) || 0;
                const del = (currentDailySource === 'SMA' ? lastDailyRawData[i].sma_export_wh : lastDailyRawData[i].sml_export_wh) || 0;
                if (con > 0 && prevCon === 0) prevCon = con;
                if (del > 0 && prevDel === 0) prevDel = del;
                if (prevCon > 0 && prevDel > 0) break;
            }

            const hourlyReadings = {};
            lastDailyRawData.forEach(item => {
                const date = new Date(item.timestamp);
                const hour = date.getHours();
                hourlyReadings[hour] = item; // Overwrite to get the latest reading of that hour
            });

            consumedBgColors = [];
            consumedBorderColors = [];
            deliveredBgColors = [];
            deliveredBorderColors = [];

            let hoursSinceLastReading = 0;

            for (let h = 0; h < 24; h++) {
                const currentItem = hourlyReadings[h];
                if (currentItem) {
                    const currentCon = (currentDailySource === 'SMA' ? currentItem.sma_import_wh : currentItem.sml_import_wh) || 0;
                    const currentDel = (currentDailySource === 'SMA' ? currentItem.sma_export_wh : currentItem.sml_export_wh) || 0;

                    // Consume delta
                    if (currentCon > 0 && prevCon > 0) {
                        const deltaCon = Math.max(0, currentCon - prevCon);
                        consumed[h] = deltaCon / 1000.0;
                        prevCon = currentCon;

                        if (hoursSinceLastReading > 0) {
                            consumedBgColors.push('rgba(156, 163, 175, 0.5)'); // Grey for accumulated
                            consumedBorderColors.push('#9ca3af');
                        } else {
                            consumedBgColors.push('rgba(245, 158, 11, 0.75)'); // Normal orange
                            consumedBorderColors.push('#f59e0b');
                        }
                    } else {
                        consumedBgColors.push('rgba(245, 158, 11, 0.75)');
                        consumedBorderColors.push('#f59e0b');
                    }
                    
                    // Deliver delta
                    if (currentDel > 0 && prevDel > 0) {
                        const deltaDel = Math.max(0, currentDel - prevDel);
                        delivered[h] = deltaDel / 1000.0;
                        prevDel = currentDel;

                        if (hoursSinceLastReading > 0) {
                            deliveredBgColors.push('rgba(156, 163, 175, 0.5)'); // Grey
                            deliveredBorderColors.push('#9ca3af');
                        } else {
                            deliveredBgColors.push('rgba(16, 185, 129, 0.75)'); // Normal green
                            deliveredBorderColors.push('#10b981');
                        }
                    } else {
                        deliveredBgColors.push('rgba(16, 185, 129, 0.75)');
                        deliveredBorderColors.push('#10b981');
                    }

                    hoursSinceLastReading = 0;
                } else {
                    // Hour is missing in DB
                    hoursSinceLastReading++;
                    consumedBgColors.push('rgba(245, 158, 11, 0.75)');
                    consumedBorderColors.push('#f59e0b');
                    deliveredBgColors.push('rgba(16, 185, 129, 0.75)');
                    deliveredBorderColors.push('#10b981');
                }
            }
        }
    } else {
        // Other scopes (week, 10days, month, year): read directly from pre-calculated dailyUsage API
        lastDailyRawData.forEach(item => {
            let valCon = (currentDailySource === 'SMA' ? item.sma_consumed_wh : item.sml_consumed_wh) || 0;
            let valDel = (currentDailySource === 'SMA' ? item.sma_delivered_wh : item.sml_delivered_wh) || 0;
            
            const d = new Date(item.date);
            let idx = -1;
            
            if (currentDailyScope === 'year') {
                idx = parseInt(item.date.split('-')[1]) - 1; // "YYYY-MM" -> MM
            } else if (currentDailyScope === 'month') {
                idx = d.getDate() - 1;
            } else if (currentDailyScope === 'week') {
                idx = (d.getDay() || 7) - 1;
            } else if (currentDailyScope === '10days') {
                const itemDate = new Date(item.date);
                const startD = new Date(currentDailyDate);
                startD.setDate(startD.getDate() - 9);
                startD.setHours(0,0,0,0);
                itemDate.setHours(0,0,0,0);
                idx = Math.round((itemDate - startD) / (1000 * 60 * 60 * 24));
            }

            if (idx >= 0 && idx < consumed.length) {
                consumed[idx] = valCon / 1000.0;
                delivered[idx] = valDel / 1000.0;
            }
        });
    }

    dailyChart.data.labels = labels;
    dailyChart.data.datasets[0].data = consumed;
    dailyChart.data.datasets[0].backgroundColor = consumedBgColors;
    dailyChart.data.datasets[0].borderColor = consumedBorderColors;
    dailyChart.data.datasets[1].data = delivered;
    dailyChart.data.datasets[1].backgroundColor = deliveredBgColors;
    dailyChart.data.datasets[1].borderColor = deliveredBorderColors;
    dailyChart.update();
    updateDailyChartVisibility();
}

// Fullscreen Toggle
function toggleFullscreen(sectionId) {
    const el = document.getElementById(sectionId);
    if (!el) return;

    if (el.classList.contains('fullscreen-card')) {
        el.classList.remove('fullscreen-card');
        document.body.classList.remove('fullscreen-mode');
    } else {
        el.classList.add('fullscreen-card');
        document.body.classList.add('fullscreen-mode');
    }

    // Trigger chart resize
    if (sectionId === 'history-section' && historyChart) historyChart.resize();
    if (sectionId === 'daily-section' && dailyChart) dailyChart.resize();
}

// Lade aktuelle Zählerstände und letzten Backup-Zeitstempel von api.php
function loadLatestTotals() {
    fetch('api.php?endpoint=latest')
        .then(res => res.json())
        .then(data => {
            if (data) {
                if (data.sml_import_wh !== undefined) {
                    document.getElementById('sml-import-val').innerText = `${(data.sml_import_wh / 1000).toFixed(2)} kWh`;
                }
                if (data.sml_export_wh !== undefined) {
                    document.getElementById('sml-export-val').innerText = `${(data.sml_export_wh / 1000).toFixed(2)} kWh`;
                }
                if (data.sma_import_wh !== undefined) {
                    document.getElementById('sma-import-val').innerText = `${(data.sma_import_wh / 1000).toFixed(2)} kWh`;
                }
                if (data.sma_export_wh !== undefined) {
                    document.getElementById('sma-export-val').innerText = `${(data.sma_export_wh / 1000).toFixed(2)} kWh`;
                }
                if (data.timestamp) {
                    const lastUpdate = new Date(data.timestamp);
                    document.getElementById('last-update-time').innerText = lastUpdate.toLocaleString('de-DE');
                }
            }
        })
        .catch(err => console.error("Failed to load latest cumulative totals", err));
}
