import { baseChartOptions } from './charts-config.js';
import { fetchHistory } from './api.js';

export let historyChart = null;

export let currentHistoryDate = new Date();
export let currentHistorySource = 'SMA';
export let currentHistoryMode = 'both';

function generateDayLabels() {
    const labels = [];
    for (let h = 0; h < 24; h++) {
        for (let m = 0; m < 60; m += 5) {
            labels.push(`${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`);
        }
    }
    return labels;
}

export function initHistoryChart(canvasId) {
    const dayLabels = generateDayLabels();
    historyChart = new Chart(document.getElementById(canvasId).getContext('2d'), {
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
    return historyChart;
}

export function updateHistoryChartVisibility() {
    if (!historyChart) return;
    const mode = currentHistoryMode;
    const showImport = mode === 'both' || mode === 'import';
    const showExport = mode === 'both' || mode === 'export';
    
    for (let i = 0; i < 3; i++) {
        historyChart.setDatasetVisibility(i, showImport);
        historyChart.setDatasetVisibility(i + 3, showExport);
    }
    historyChart.update();
}

export async function loadHistoryData() {
    if (!historyChart) return;
    
    const year = currentHistoryDate.getFullYear();
    const month = (currentHistoryDate.getMonth() + 1).toString().padStart(2, '0');
    const day = currentHistoryDate.getDate().toString().padStart(2, '0');
    const dateStr = `${year}-${month}-${day}`;

    const data = await fetchHistory(dateStr);
    
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
}

export function setHistorySource(src) {
    currentHistorySource = src;
    document.getElementById('btn-src-sma').classList.toggle('active', src === 'SMA');
    document.getElementById('btn-src-sml').classList.toggle('active', src === 'SML');
    loadHistoryData();
}

export function setHistoryMode(mode) {
    currentHistoryMode = mode;
    document.getElementById('btn-mode-both').classList.toggle('active', mode === 'both');
    document.getElementById('btn-mode-import').classList.toggle('active', mode === 'import');
    document.getElementById('btn-mode-export').classList.toggle('active', mode === 'export');
    updateHistoryChartVisibility();
}

export function navigateHistory(offset) {
    if (offset === 'today') {
        currentHistoryDate = new Date();
    } else {
        currentHistoryDate.setDate(currentHistoryDate.getDate() + offset);
    }
    
    if (currentHistoryDate > new Date()) {
        currentHistoryDate = new Date();
    }
    
    const today = new Date();
    const isToday = currentHistoryDate.toDateString() === today.toDateString();
    document.getElementById('btn-date-today').classList.toggle('active', isToday);
    
    const titleDate = isToday ? "Heute" : currentHistoryDate.toLocaleDateString();
    document.getElementById('history-chart-title').innerText = `Leistungsverlauf (${titleDate})`;
    
    loadHistoryData();
}

export function forceHistoryDate(newDate) {
    currentHistoryDate = newDate;
}
