import { writable } from 'svelte/store';

export interface DataPoint {
    bytesCopied: number;
    speed: number;
}

export interface TransferMetrics {
    dataPoints: DataPoint[];
    maxSpeed: number;
    lastSpeed: number;
    currentFile: Record<string, any> | null;
}

// Map of transfer ID to graph data and other metrics
export const transferMetricsStore = writable<Map<string, TransferMetrics>>(new Map());

export function updateGraphData(transferId: string, bytesCopied: number, speed: number) {
    transferMetricsStore.update(map => {
        const current = map.get(transferId) || { dataPoints: [], maxSpeed: 0, lastSpeed: 0, currentFile: null };

        const lastPoint = current.dataPoints.at(-1);

        // Handle potential reset (e.g. retry or new job with same ID)
        if (lastPoint && bytesCopied < lastPoint.bytesCopied - 1024 * 1024) { // 1MB threshold for "significant reset"
            map.set(transferId, {
                ...current,
                dataPoints: [{ bytesCopied, speed }],
                maxSpeed: speed,
                lastSpeed: speed
            });
            return map;
        }

        // Only add if we have new forward progress
        if (current.dataPoints.length === 0 || (bytesCopied > (lastPoint?.bytesCopied || 0))) {
            const newDataPoints = [...current.dataPoints, { bytesCopied, speed }];
            const newMaxSpeed = Math.max(current.maxSpeed, speed);

            map.set(transferId, {
                ...current,
                dataPoints: newDataPoints,
                maxSpeed: newMaxSpeed,
                lastSpeed: speed
            });
        } else {
            // Just update current speed if bytes haven't changed (or to keep it fresh)
            map.set(transferId, {
                ...current,
                lastSpeed: speed
            });
        }
        return map;
    });
}

export function setTransferMetrics(transferId: string, metrics: Partial<TransferMetrics>) {
    transferMetricsStore.update(map => {
        const current = map.get(transferId) || { dataPoints: [], maxSpeed: 0, lastSpeed: 0, currentFile: null };
        map.set(transferId, { ...current, ...metrics });
        return map;
    });
}
