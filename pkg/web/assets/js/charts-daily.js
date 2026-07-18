import { baseChartOptions } from './charts-config.js';
import { fetchHistory, fetchMonthly } from './api.js';
import { formatNum } from './utils.js';

export let dailyChart = null;

export let currentDailyScope = 'week';
export let currentDailySource = 'SMA';
export let currentDailyMode = 'both';
export let currentDailyDate = new Date();
export let lastDailyRawData = [];

export function initDailyChart(canvasId) {
    dailyChart = new Chart(document.getElementById(canvasId).getContext('2d'), {
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
    return dailyChart;
}

export function updateDailyChartVisibility() {
    if (!dailyChart) return;
    dailyChart.setDatasetVisibility(0, currentDailyMode === 'both' || currentDailyMode === 'import');
    dailyChart.setDatasetVisibility(1, currentDailyMode === 'both' || currentDailyMode === 'export');
    dailyChart.update();
}

export async function loadDailyUsageData() {
    if (!dailyChart) return;
    
    let startD = new Date(currentDailyDate);
    let endD = new Date(currentDailyDate);
    let isToday = false;
    
    if (currentDailyScope === 'year') {
        const year = currentDailyDate.getFullYear();
        isToday = year === new Date().getFullYear();
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${year})`;
        lastDailyRawData = await fetchMonthly(year);
    } else if (currentDailyScope === 'day') {
        const dateStr = `${startD.getFullYear()}-${(startD.getMonth()+1).toString().padStart(2,'0')}-${startD.getDate().toString().padStart(2,'0')}`;
        isToday = dateStr === new Date().toLocaleDateString('sv-SE');
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${startD.toLocaleDateString([], { day: 'numeric', month: 'short', year: 'numeric' })})`;
        lastDailyRawData = await fetchHistory(dateStr);
    } else {
        if (currentDailyScope === 'week') {
            const day = startD.getDay() || 7;
            startD.setDate(startD.getDate() - day + 1);
            endD = new Date(startD);
            endD.setDate(startD.getDate() + 6);
        } else if (currentDailyScope === '10days') {
            startD.setDate(endD.getDate() - 9);
        } else if (currentDailyScope === 'month') {
            startD.setDate(1);
            endD = new Date(startD.getFullYear(), startD.getMonth() + 1, 0);
        }
        
        startD.setHours(0, 0, 0, 0);
        endD.setHours(23, 59, 59, 999);
        
        isToday = (new Date() >= startD && new Date() <= endD);
        
        const fmt = { day: 'numeric', month: 'short', year: 'numeric' };
        document.getElementById('daily-chart-title').innerText = `Tagesübersicht (${startD.toLocaleDateString([], fmt)} - ${endD.toLocaleDateString([], fmt)})`;
        
        const startStr = `${startD.getFullYear()}-${(startD.getMonth()+1).toString().padStart(2,'0')}-${startD.getDate().toString().padStart(2,'0')}`;
        const endStr = `${endD.getFullYear()}-${(endD.getMonth()+1).toString().padStart(2,'0')}-${endD.getDate().toString().padStart(2,'0')}`;
        
        try {
            const res = await fetch(`/api/daily?start=${startStr}&end=${endStr}`);
            lastDailyRawData = await res.json();
        } catch (err) {
            console.error("Failed to load daily usage data", err);
            lastDailyRawData = [];
        }
    }
    
    document.getElementById('btn-daily-date-today').classList.toggle('active', isToday);
    renderDailyChart();
    
    if (lastDailyRawData && lastDailyRawData.length > 0) {
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
        
        const elemSmlConsumed = document.getElementById('sml-today-consumed');
        if (elemSmlConsumed) {
            elemSmlConsumed.innerText = `${formatNum(smlConsumed)} kWh`;
            document.getElementById('sml-today-delivered').innerText = `${formatNum(smlDelivered)} kWh`;
            document.getElementById('sma-today-consumed').innerText = `${formatNum(smaConsumed)} kWh`;
            document.getElementById('sma-today-delivered').innerText = `${formatNum(smaDelivered)} kWh`;
        }
    }
}

export function renderDailyChart() {
    if (!dailyChart) return;
    
    const labels = [];
    const consumed = [];
    const delivered = [];

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
        if (lastDailyRawData.length > 0) {
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
                hourlyReadings[hour] = item;
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

                    if (currentCon > 0 && prevCon > 0) {
                        const deltaCon = Math.max(0, currentCon - prevCon);
                        consumed[h] = deltaCon / 1000.0;
                        prevCon = currentCon;

                        if (hoursSinceLastReading > 0) {
                            consumedBgColors.push('rgba(156, 163, 175, 0.5)');
                            consumedBorderColors.push('#9ca3af');
                        } else {
                            consumedBgColors.push('rgba(245, 158, 11, 0.75)');
                            consumedBorderColors.push('#f59e0b');
                        }
                    } else {
                        consumedBgColors.push('rgba(245, 158, 11, 0.75)');
                        consumedBorderColors.push('#f59e0b');
                    }
                    
                    if (currentDel > 0 && prevDel > 0) {
                        const deltaDel = Math.max(0, currentDel - prevDel);
                        delivered[h] = deltaDel / 1000.0;
                        prevDel = currentDel;

                        if (hoursSinceLastReading > 0) {
                            deliveredBgColors.push('rgba(156, 163, 175, 0.5)');
                            deliveredBorderColors.push('#9ca3af');
                        } else {
                            deliveredBgColors.push('rgba(16, 185, 129, 0.75)');
                            deliveredBorderColors.push('#10b981');
                        }
                    } else {
                        deliveredBgColors.push('rgba(16, 185, 129, 0.75)');
                        deliveredBorderColors.push('#10b981');
                    }

                    hoursSinceLastReading = 0;
                } else {
                    hoursSinceLastReading++;
                    consumedBgColors.push('rgba(245, 158, 11, 0.75)');
                    consumedBorderColors.push('#f59e0b');
                    deliveredBgColors.push('rgba(16, 185, 129, 0.75)');
                    deliveredBorderColors.push('#10b981');
                }
            }
        }
    } else {
        lastDailyRawData.forEach(item => {
            let valCon = (currentDailySource === 'SMA' ? item.sma_consumed_wh : item.sml_consumed_wh) || 0;
            let valDel = (currentDailySource === 'SMA' ? item.sma_delivered_wh : item.sml_delivered_wh) || 0;
            
            const d = new Date(item.date);
            let idx = -1;
            
            if (currentDailyScope === 'year') {
                idx = parseInt(item.date.split('-')[1]) - 1;
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

export function setDailyScope(scope) {
    currentDailyScope = scope;
    ['day', 'week', '10days', 'month', 'year'].forEach(s => {
        const btn = document.getElementById(`btn-daily-scope-${s}`);
        if(btn) btn.classList.toggle('active', s === scope);
    });
    currentDailyDate = new Date();
    loadDailyUsageData();
}

export function setDailySource(src) {
    currentDailySource = src;
    document.getElementById('btn-daily-src-sma').classList.toggle('active', src === 'SMA');
    document.getElementById('btn-daily-src-sml').classList.toggle('active', src === 'SML');
    loadDailyUsageData();
}

export function setDailyMode(mode) {
    currentDailyMode = mode;
    document.getElementById('btn-daily-mode-both').classList.toggle('active', mode === 'both');
    document.getElementById('btn-daily-mode-import').classList.toggle('active', mode === 'import');
    document.getElementById('btn-daily-mode-export').classList.toggle('active', mode === 'export');
    updateDailyChartVisibility();
}

export function navigateDaily(offset) {
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

export function forceDailyDate(newDate) {
    currentDailyDate = newDate;
}
