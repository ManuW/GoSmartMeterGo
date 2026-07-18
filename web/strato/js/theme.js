export function getCurrentTheme() {
    return localStorage.getItem('theme') || 'dark';
}

export function initTheme(updateChartsCallback) {
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
            if (typeof updateChartsCallback === 'function') {
                updateChartsCallback(next);
            }
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
