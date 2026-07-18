import { formatNum } from './utils.js';

export const baseChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
        legend: {
            labels: { color: '#9ca3af', font: { family: 'Inter' } }
        },
        tooltip: {
            callbacks: {
                label: function(context) {
                    let label = context.dataset.label || '';
                    if (label) label += ': ';
                    if (context.parsed.y !== null) {
                        label += formatNum(context.parsed.y, 2);
                    }
                    return label;
                }
            }
        }
    },
    scales: {
        x: {
            grid: { color: 'rgba(255, 255, 255, 0.03)' },
            ticks: { color: '#9ca3af', font: { family: 'Inter', size: 10 } }
        },
        y: {
            grid: { color: 'rgba(255, 255, 255, 0.03)' },
            ticks: { 
                color: '#9ca3af', 
                font: { family: 'Inter', size: 10 },
                callback: function(value) {
                    return formatNum(value, 0);
                }
            }
        }
    }
};

export function updateChartsForTheme(theme, historyChart, dailyChart) {
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
