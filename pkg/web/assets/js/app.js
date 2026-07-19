import { initTheme } from './theme.js';
import { updateChartsForTheme } from './charts-config.js';
import { 
    initHistoryChart, loadHistoryData, setHistorySource, 
    setHistoryMode, navigateHistory, forceHistoryDate, historyChart, currentHistoryDate 
} from './charts-history.js';
import { 
    initDailyChart, loadDailyUsageData, setDailyScope, 
    setDailySource, setDailyMode, setDailyUnit, navigateDaily, dailyChart
} from './charts-daily.js';
import { updateBadge, updateMeterValues, getPowerState, applyPowerState, resetPowerState, formatNum, toggleFullscreen } from './utils.js';

let lastHistoryUpdateMinute = -1;

let liveSparklineChart = null;
const sparklinePoints = 120; // 2 minutes of data at 1 sec intervals
const liveSparklineData = Array(sparklinePoints).fill(0);

document.addEventListener('DOMContentLoaded', () => {
    initTheme((theme) => {
        updateChartsForTheme(theme, historyChart, dailyChart);
    });
    
    initHistoryChart('historyChart');
    initDailyChart('dailyChart');
    
    startLiveStream();
    loadHistoryData();
    loadDailyUsageData();
    initSparkline();
    initBackupTrigger();
    
    // Attach navigation logic
    document.getElementById('nav-overview')?.addEventListener('click', (e) => { e.preventDefault(); showSection('live-section'); });
    document.getElementById('nav-history')?.addEventListener('click', (e) => { e.preventDefault(); showSection('history-section'); });
    document.getElementById('nav-daily')?.addEventListener('click', (e) => { e.preventDefault(); showSection('daily-section'); });
    document.getElementById('nav-settings')?.addEventListener('click', (e) => { e.preventDefault(); showSection('settings-section'); });

    // Attach event listeners for History Chart UI
    document.getElementById('btn-src-sma')?.addEventListener('click', () => setHistorySource('SMA'));
    document.getElementById('btn-src-sml')?.addEventListener('click', () => setHistorySource('SML'));
    document.getElementById('btn-mode-both')?.addEventListener('click', () => setHistoryMode('both'));
    document.getElementById('btn-mode-import')?.addEventListener('click', () => setHistoryMode('import'));
    document.getElementById('btn-mode-export')?.addEventListener('click', () => setHistoryMode('export'));
    
    document.getElementById('btn-history-prev-7')?.addEventListener('click', () => navigateHistory(-7));
    document.getElementById('btn-history-prev-1')?.addEventListener('click', () => navigateHistory(-1));
    document.getElementById('btn-date-today')?.addEventListener('click', () => navigateHistory('today'));
    document.getElementById('btn-history-next-1')?.addEventListener('click', () => navigateHistory(1));
    document.getElementById('btn-history-next-7')?.addEventListener('click', () => navigateHistory(7));
    
    document.getElementById('btn-history-fullscreen')?.addEventListener('click', () => toggleFullscreen('history-section'));

    // Attach event listeners for Daily Chart UI
    document.getElementById('btn-daily-scope-day')?.addEventListener('click', () => setDailyScope('day'));
    document.getElementById('btn-daily-scope-week')?.addEventListener('click', () => setDailyScope('week'));
    document.getElementById('btn-daily-scope-10days')?.addEventListener('click', () => setDailyScope('10days'));
    document.getElementById('btn-daily-scope-month')?.addEventListener('click', () => setDailyScope('month'));
    document.getElementById('btn-daily-scope-year')?.addEventListener('click', () => setDailyScope('year'));
    
    document.getElementById('btn-daily-src-sma')?.addEventListener('click', () => setDailySource('SMA'));
    document.getElementById('btn-daily-src-sml')?.addEventListener('click', () => setDailySource('SML'));
    document.getElementById('btn-daily-mode-both')?.addEventListener('click', () => setDailyMode('both'));
    document.getElementById('btn-daily-mode-import')?.addEventListener('click', () => setDailyMode('import'));
    document.getElementById('btn-daily-mode-export')?.addEventListener('click', () => setDailyMode('export'));
    
    document.getElementById('btn-daily-prev-1')?.addEventListener('click', () => navigateDaily(-1));
    document.getElementById('btn-daily-date-today')?.addEventListener('click', () => navigateDaily('today'));
    document.getElementById('btn-daily-next-1')?.addEventListener('click', () => navigateDaily(1));
    
    document.getElementById('btn-daily-unit-auto')?.addEventListener('click', () => setDailyUnit('auto'));
    document.getElementById('btn-daily-unit-kwh')?.addEventListener('click', () => setDailyUnit('kwh'));
    document.getElementById('btn-daily-unit-wh')?.addEventListener('click', () => setDailyUnit('wh'));
    
    document.getElementById('btn-daily-fullscreen')?.addEventListener('click', () => toggleFullscreen('daily-section'));
});

function initSparkline() {
    const sparkCtx = document.getElementById('live-sparkline');
    if (sparkCtx && window.Chart) {
        liveSparklineChart = new Chart(sparkCtx.getContext('2d'), {
            type: 'line',
            data: {
                labels: Array(sparklinePoints).fill(''),
                datasets: [{
                    data: liveSparklineData,
                    borderWidth: 2,
                    pointRadius: 0,
                    tension: 0.2,
                    fill: {
                        target: 'origin',
                        above: 'rgba(245, 158, 11, 0.2)', // Light orange for import
                        below: 'rgba(16, 185, 129, 0.2)'  // Light green for export
                    },
                    segment: {
                        borderColor: ctx => (ctx.p1 && ctx.p1.parsed.y < 0) ? '#10b981' : '#f59e0b'
                    }
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                plugins: {
                    legend: { display: false },
                    tooltip: { enabled: false }
                },
                scales: {
                    x: { display: false },
                    y: { display: false }
                }
            }
        });
    }
}

function updateSparkline(newValue) {
    if (!liveSparklineChart) return;
    liveSparklineData.push(newValue);
    liveSparklineData.shift();
    liveSparklineChart.update('none'); // Update without animation for snappiness
}

function startLiveStream() {
    const eventSource = new EventSource('/api/live/stream');

    eventSource.onmessage = (event) => {
        const data = JSON.parse(event.data);
        updateLiveDashboard(data);
    };

    eventSource.onerror = (err) => {
        console.error("SSE stream error, retrying...", err);
        const smlBadge = document.getElementById('sml-status');
        if (smlBadge) {
            smlBadge.className = "badge badge-error";
            smlBadge.innerText = "SML: getrennt";
        }
        const smaBadge = document.getElementById('sma-status');
        if (smaBadge) {
            smaBadge.className = "badge badge-error";
            smaBadge.innerText = "SMA: getrennt";
        }
    };
}

function initBackupTrigger() {
    const btn = document.getElementById('btn-trigger-backup');
    if (!btn) return;
    
    btn.addEventListener('click', () => {
        if (!confirm("Soll das Datenbank-Backup jetzt angestoßen und auf den Strato-Server hochgeladen werden?")) {
            return;
        }

        btn.disabled = true;
        btn.innerText = "⏳ Backup läuft...";

        fetch('/api/backup/upload', { method: 'POST' })
            .then(res => res.json().then(data => ({ status: res.status, data })))
            .then(({ status, data }) => {
                if (status === 200) {
                    alert("Erfolg: " + (data.message || "Online-Backup erfolgreich durchgeführt!"));
                } else {
                    alert("Fehler beim Backup: " + (data.error || "Unbekannter Fehler"));
                }
            })
            .catch(err => {
                alert("Netzwerkfehler beim Backup: " + err.message);
            })
            .finally(() => {
                btn.disabled = false;
                btn.innerText = "☁️ Online-Backup anstoßen";
            });
    });
}

export function updateLiveDashboard(data) {
    const sml = data.sml || {};
    const sma = data.sma || {};

    const now = new Date();
    document.getElementById('last-update-time').innerText = now.toLocaleTimeString('de-DE');

    const currentMinute = now.getMinutes();
    if (currentMinute % 5 === 0 && currentMinute !== lastHistoryUpdateMinute) {
        lastHistoryUpdateMinute = currentMinute;
        
        const isTrackingToday = document.getElementById('btn-date-today').classList.contains('active');
        if (isTrackingToday) {
            if (currentHistoryDate.toDateString() !== now.toDateString()) {
                forceHistoryDate(new Date());
                document.getElementById('history-chart-title').innerText = 'Leistungsverlauf (Heute)';
            }
            loadHistoryData();
        }
    }

    const smlActive = sml.sml_active;
    const smaActive = sma.sma_active;

    updateBadge('sml-status', smlActive, 'SML');
    updateBadge('sma-status', smaActive, 'SMA');

    if (smlActive) updateMeterValues('sml', sml, 'sml', ['l1', 'l2', 'l3']);
    if (smaActive) updateMeterValues('sma', sma, 'sma', ['l1', 'l3', 'l2']);

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
        if(smlElements.val) smlElements.val.innerText = `${formatNum(value, 0)} W`;
        if(smlElements.bar) smlElements.bar.style.width = `${Math.min((value / 6000) * 100, 100)}%`;
        applyPowerState(smlElements, isImport);
        
        // Update Sparkline
        const realValue = isImport ? value : -value;
        updateSparkline(realValue);
    } else {
        resetPowerState(smlElements);
        updateSparkline(0);
    }

    const smaElements = {
        pulse: document.getElementById('sma-power-pulse'),
        val: document.getElementById('sma-power-val'),
        label: document.getElementById('sma-power-label'),
        valBase: 'sma-value',
        labelBase: 'sma-status-text',
    };

    if (smaActive) {
        const { value, isImport } = getPowerState(sma, 'sma');
        if(smaElements.val) smaElements.val.innerText = `${formatNum(value, 0)} W`;
        applyPowerState(smaElements, isImport);
    } else {
        resetPowerState(smaElements);
    }
}
// Attach to window so external systems (if any) can call it
window.updateLiveDashboard = updateLiveDashboard;
