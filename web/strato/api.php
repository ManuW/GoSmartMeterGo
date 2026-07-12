<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

// Finde alle Backup-Datenbanken im backups-Verzeichnis
$files = glob(__DIR__ . '/backups/*/smartmeter_sync.db');

if (empty($files)) {
    http_response_code(404);
    echo json_encode(["error" => "No database backups found"]);
    exit;
}

// Sortiere absteigend (die neuste Verzeichnis-Zeitstempel-Kombination zuerst)
rsort($files);
$dbFile = $files[0];

if (!file_exists($dbFile)) {
    http_response_code(404);
    echo json_encode(["error" => "Database file not found"]);
    exit;
}

try {
    // Verbindung zur SQLite-Datenbankdatei herstellen
    $db = new PDO("sqlite:" . $dbFile);
    $db->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    
    $endpoint = $_GET['endpoint'] ?? '';
    
    if ($endpoint === 'latest') {
        $stmt = $db->query("
            SELECT timestamp, sml_import_wh, sml_export_wh, sma_import_wh, sma_export_wh
            FROM metrics_summary
            ORDER BY timestamp DESC
            LIMIT 1
        ");
        echo json_encode($stmt->fetch(PDO::FETCH_ASSOC));
        
    } elseif ($endpoint === 'history') {
        $date = $_GET['date'] ?? '';
        if (empty($date)) {
            $date = date('Y-m-d');
        }
        
        // Suche passend für ISO-Zeitstempel-Strings
        $start = $date . 'T00:00:00';
        $end = $date . 'T23:59:59.999999999';
        
        $stmt = $db->prepare("
            SELECT 
                timestamp, sml_import_wh, sml_export_wh, sma_import_wh, sma_export_wh,
                sml_power_import_w_min, sml_power_import_w_max, sml_power_import_w_avg,
                sml_power_export_w_min, sml_power_export_w_max, sml_power_export_w_avg,
                sma_power_import_w_min, sma_power_import_w_max, sma_power_import_w_avg,
                sma_power_export_w_min, sma_power_export_w_max, sma_power_export_w_avg
            FROM metrics_summary
            WHERE timestamp >= :start AND timestamp <= :end
            ORDER BY timestamp ASC
        ");
        $stmt->execute([':start' => $start, ':end' => $end]);
        echo json_encode($stmt->fetchAll(PDO::FETCH_ASSOC));
        
    } elseif ($endpoint === 'daily') {
        $start = $_GET['start'] ?? '';
        $end = $_GET['end'] ?? '';
        
        if (empty($start) || empty($end)) {
            http_response_code(400);
            echo json_encode(["error" => "Missing start or end parameters"]);
            exit;
        }

        $start = $start . 'T00:00:00';
        $end = $end . 'T23:59:59.999999999';

        // Berechnet über LAG() für lückenlose Übergänge
        $stmt = $db->prepare("
            WITH daily_max AS (
                SELECT 
                    strftime('%Y-%m-%d', timestamp) as date,
                    MAX(sml_import_wh) as max_sml_in,
                    MIN(CASE WHEN sml_import_wh > 0 THEN sml_import_wh END) as min_sml_in,
                    MAX(sml_export_wh) as max_sml_out,
                    MIN(CASE WHEN sml_export_wh > 0 THEN sml_export_wh END) as min_sml_out,
                    MAX(sma_import_wh) as max_sma_in,
                    MIN(CASE WHEN sma_import_wh > 0 THEN sma_import_wh END) as min_sma_in,
                    MAX(sma_export_wh) as max_sma_out,
                    MIN(CASE WHEN sma_export_wh > 0 THEN sma_export_wh END) as min_sma_out
                FROM metrics_summary
                WHERE timestamp >= :start AND timestamp <= :end
                GROUP BY date
            )
            SELECT 
                date,
                COALESCE(max_sml_in - LAG(max_sml_in, 1, min_sml_in) OVER (ORDER BY date), 0) as sml_consumed_wh,
                COALESCE(max_sml_out - LAG(max_sml_out, 1, min_sml_out) OVER (ORDER BY date), 0) as sml_delivered_wh,
                COALESCE(max_sma_in - LAG(max_sma_in, 1, min_sma_in) OVER (ORDER BY date), 0) as sma_consumed_wh,
                COALESCE(max_sma_out - LAG(max_sma_out, 1, min_sma_out) OVER (ORDER BY date), 0) as sma_delivered_wh
            FROM daily_max
            ORDER BY date ASC
        ");
        $stmt->execute([':start' => $start, ':end' => $end]);
        echo json_encode($stmt->fetchAll(PDO::FETCH_ASSOC));
        
    } elseif ($endpoint === 'monthly') {
        $year = intval($_GET['year'] ?? date('Y'));
        $start = "$year-01-01T00:00:00";
        $end = "$year-12-31T23:59:59.999999999";
        
        $stmt = $db->prepare("
            WITH monthly_max AS (
                SELECT 
                    strftime('%Y-%m', timestamp) as date,
                    MAX(sml_import_wh) as max_sml_in,
                    MIN(CASE WHEN sml_import_wh > 0 THEN sml_import_wh END) as min_sml_in,
                    MAX(sml_export_wh) as max_sml_out,
                    MIN(CASE WHEN sml_export_wh > 0 THEN sml_export_wh END) as min_sml_out,
                    MAX(sma_import_wh) as max_sma_in,
                    MIN(CASE WHEN sma_import_wh > 0 THEN sma_import_wh END) as min_sma_in,
                    MAX(sma_export_wh) as max_sma_out,
                    MIN(CASE WHEN sma_export_wh > 0 THEN sma_export_wh END) as min_sma_out
                FROM metrics_summary
                WHERE timestamp >= :start AND timestamp <= :end
                GROUP BY date
            )
            SELECT 
                date,
                COALESCE(max_sml_in - LAG(max_sml_in, 1, min_sml_in) OVER (ORDER BY date), 0) as sml_consumed_wh,
                COALESCE(max_sml_out - LAG(max_sml_out, 1, min_sml_out) OVER (ORDER BY date), 0) as sml_delivered_wh,
                COALESCE(max_sma_in - LAG(max_sma_in, 1, min_sma_in) OVER (ORDER BY date), 0) as sma_consumed_wh,
                COALESCE(max_sma_out - LAG(max_sma_out, 1, min_sma_out) OVER (ORDER BY date), 0) as sma_delivered_wh
            FROM monthly_max
            ORDER BY date ASC
        ");
        $stmt->execute([':start' => $start, ':end' => $end]);
        echo json_encode($stmt->fetchAll(PDO::FETCH_ASSOC));
        
    } else {
        http_response_code(400);
        echo json_encode(["error" => "Invalid endpoint"]);
    }
    
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(["error" => "Database connection or query failed: " . $e->getMessage()]);
}
?>
