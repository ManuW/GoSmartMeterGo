let currentDayData = [];

document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    
    const datePicker = document.getElementById('date-picker');
    const btnExport = document.getElementById('btn-export-csv');
    const today = new Date();
    // Default to today
    datePicker.value = today.toLocaleDateString('sv-SE'); // YYYY-MM-DD
    
    datePicker.addEventListener('change', () => {
        loadData(datePicker.value);
    });
    
    if (btnExport) {
        btnExport.addEventListener('click', () => {
            exportToCSV(datePicker.value);
        });
    }
    
    loadData(datePicker.value);
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

function loadData(dateStr) {
    const tbody = document.getElementById('table-body');
    const btnExport = document.getElementById('btn-export-csv');
    tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; padding: 2rem;">Lade Daten...</td></tr>';
    if (btnExport) btnExport.disabled = true;
    currentDayData = [];
    
    fetch(`/api/history?date=${dateStr}`)
        .then(res => res.json())
        .then(data => {
            if (!data || data.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; padding: 2rem; color: var(--text-muted);">Keine Daten für diesen Tag gefunden.</td></tr>';
                return;
            }
            
            currentDayData = data;
            if (btnExport) btnExport.disabled = false;
            tbody.innerHTML = '';
            
            data.forEach(row => {
                const tr = document.createElement('tr');
                
                // Format time (HH:MM)
                const d = new Date(row.timestamp);
                const timeStr = `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
                
                // Format Wh to kWh with exactly 3 decimals using German locale (comma)
                const formatKwh = (wh) => {
                    if (wh === 0 || wh == null) return '-';
                    return (wh / 1000).toLocaleString('de-DE', { minimumFractionDigits: 3, maximumFractionDigits: 3 }) + ' kWh';
                };
                
                tr.innerHTML = `
                    <td>${timeStr}</td>
                    <td>${formatKwh(row.sml_import_wh)}</td>
                    <td>${formatKwh(row.sml_export_wh)}</td>
                    <td>${formatKwh(row.sma_import_wh)}</td>
                    <td>${formatKwh(row.sma_export_wh)}</td>
                `;
                
                tbody.appendChild(tr);
            });
        })
        .catch(err => {
            console.error(err);
            tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; padding: 2rem; color: #ef4444;">Fehler beim Laden der Daten.</td></tr>';
        });
}

function exportToCSV(dateStr) {
    if (!currentDayData || currentDayData.length === 0) return;

    const headers = [
        "Timestamp",
        "SML_Import_Wh", "SML_Export_Wh",
        "SMA_Import_Wh", "SMA_Export_Wh",
        "SML_Power_Import_W_Min", "SML_Power_Import_W_Max", "SML_Power_Import_W_Avg",
        "SML_Power_Export_W_Min", "SML_Power_Export_W_Max", "SML_Power_Export_W_Avg",
        "SMA_Power_Import_W_Min", "SMA_Power_Import_W_Max", "SMA_Power_Import_W_Avg",
        "SMA_Power_Export_W_Min", "SMA_Power_Export_W_Max", "SMA_Power_Export_W_Avg"
    ];

    const formatCSVValue = (val) => {
        if (val == null) return "";
        return val.toString().replace('.', ',');
    };

    const rows = currentDayData.map(row => {
        return [
            row.timestamp,
            formatCSVValue(row.sml_import_wh), formatCSVValue(row.sml_export_wh),
            formatCSVValue(row.sma_import_wh), formatCSVValue(row.sma_export_wh),
            formatCSVValue(row.sml_power_import_w_min), formatCSVValue(row.sml_power_import_w_max), formatCSVValue(row.sml_power_import_w_avg),
            formatCSVValue(row.sml_power_export_w_min), formatCSVValue(row.sml_power_export_w_max), formatCSVValue(row.sml_power_export_w_avg),
            formatCSVValue(row.sma_power_import_w_min), formatCSVValue(row.sma_power_import_w_max), formatCSVValue(row.sma_power_import_w_avg),
            formatCSVValue(row.sma_power_export_w_min), formatCSVValue(row.sma_power_export_w_max), formatCSVValue(row.sma_power_export_w_avg)
        ].join(";");
    });

    const csvContent = headers.join(";") + "\n" + rows.join("\n");
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    
    const link = document.createElement("a");
    link.setAttribute("href", url);
    link.setAttribute("download", `smartmeter_export_${dateStr}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}
