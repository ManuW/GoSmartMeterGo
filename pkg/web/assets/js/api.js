export async function fetchHistory(dateStr) {
    try {
        const res = await fetch(`/api/history?date=${dateStr}`);
        return await res.json();
    } catch (err) {
        console.error("Failed to load history data", err);
        return [];
    }
}

export async function fetchMonthly(year) {
    try {
        const res = await fetch(`/api/monthly?year=${year}`);
        return await res.json();
    } catch (err) {
        console.error("Failed to load monthly data", err);
        return [];
    }
}

export async function fetchLatest() {
    try {
        const res = await fetch('api.php?endpoint=latest');
        return await res.json();
    } catch (err) {
        console.error("Failed to load latest cumulative totals", err);
        return null;
    }
}
