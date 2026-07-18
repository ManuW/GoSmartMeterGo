export function formatNum(value, fractionDigits = 2) {
    if (value === undefined || value === null || isNaN(value)) return "0,00";
    return value.toLocaleString('de-DE', {
        minimumFractionDigits: fractionDigits,
        maximumFractionDigits: fractionDigits
    });
}

export function updateBadge(id, active, prefix) {
    const badge = document.getElementById(id);
    if (badge) {
        if (active) {
            badge.className = "badge badge-success";
            badge.innerText = `${prefix}: verbunden`;
        } else {
            badge.className = "badge badge-error";
            badge.innerText = `${prefix}: inaktiv`;
        }
    }
}

export function updateMeterValues(idPrefix, data, keyPrefix, phaseOrder) {
    document.getElementById(`${idPrefix}-import-val`).innerText = `${formatNum(data[`${keyPrefix}_energy_import`] / 1000)} kWh`;
    document.getElementById(`${idPrefix}-export-val`).innerText = `${formatNum(data[`${keyPrefix}_energy_export`] / 1000)} kWh`;

    const totalPower = data[`${keyPrefix}_power_import`] > 0 ? data[`${keyPrefix}_power_import`] : -data[`${keyPrefix}_power_export`];
    document.getElementById(`${idPrefix}-power-total-val`).innerText = `${totalPower > 0 ? "+" : ""}${formatNum(totalPower, 0)} W`;

    const [p1, p2, p3] = phaseOrder;
    document.getElementById(`${idPrefix}-power-phases-val`).innerText = `${formatNum(data[`${keyPrefix}_power_${p1}`], 0)} / ${formatNum(data[`${keyPrefix}_power_${p2}`], 0)} / ${formatNum(data[`${keyPrefix}_power_${p3}`], 0)} W`;
    document.getElementById(`${idPrefix}-voltage-phases-val`).innerText = `${formatNum(data[`${keyPrefix}_voltage_${p1}`], 1)} / ${formatNum(data[`${keyPrefix}_voltage_${p2}`], 1)} / ${formatNum(data[`${keyPrefix}_voltage_${p3}`], 1)} V`;
    document.getElementById(`${idPrefix}-current-phases-val`).innerText = `${formatNum(data[`${keyPrefix}_current_${p1}`], 2)} / ${formatNum(data[`${keyPrefix}_current_${p2}`], 2)} / ${formatNum(data[`${keyPrefix}_current_${p3}`], 2)} A`;
    document.getElementById(`${idPrefix}-freq-val`).innerText = `${formatNum(data[`${keyPrefix}_frequency`], 2)} Hz`;
    document.getElementById(`${idPrefix}-interval-val`).innerText = `${data[`${keyPrefix}_interval_ms`] || 0} ms`;
}

export function getPowerState(data, keyPrefix) {
    if (data[`${keyPrefix}_power_import`] > 0) {
        return { value: data[`${keyPrefix}_power_import`], isImport: true };
    } else if (data[`${keyPrefix}_power_export`] > 0) {
        return { value: data[`${keyPrefix}_power_export`], isImport: false };
    }
    return { value: 0, isImport: true };
}

export function applyPowerState(elements, isImport) {
    const mode = isImport ? 'import' : 'export';
    const label = isImport ? 'Netzbezug' : 'Netzeinspeisung';
    const colorClass = isImport ? 'import-text' : 'export-text';

    if (elements.pulse) elements.pulse.className = `pulse-dot active-${mode}`;
    if (elements.label) {
        elements.label.innerText = label;
        elements.label.className = `${elements.labelBase} ${colorClass}`;
    }
    if (elements.val) elements.val.className = `${elements.valBase} ${colorClass}`;
    if (elements.card) elements.card.className = `card live-card ${isImport ? 'import' : 'surplus'}`;
    if (elements.bar) elements.bar.style.backgroundColor = isImport ? '#f59e0b' : '#10b981';
}

export function resetPowerState(elements) {
    if (elements.pulse) elements.pulse.className = "pulse-dot";
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
    if (elements.card) elements.card.className = "card live-card";
    if (elements.bar) elements.bar.style.width = "0%";
}

export function toggleFullscreen(sectionId) {
    const section = document.getElementById(sectionId);
    if (!section) return;
    if (!document.fullscreenElement) {
        section.requestFullscreen().catch(err => {
            console.error(`Error attempting to enable fullscreen: ${err.message}`);
        });
    } else {
        document.exitFullscreen();
    }
}
