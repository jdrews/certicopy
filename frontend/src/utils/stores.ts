import { writable } from 'svelte/store';

export interface DataPoint {
    bytesCopied: number;
    speed: number;
}

interface GraphData {
    dataPoints: DataPoint[];
    maxSpeed: number;
}

// Map of transfer ID to graph data
export const graphDataStore = writable<Map<string, GraphData>>(new Map());

export function updateGraphData(transferId: string, bytesCopied: number, speed: number) {
    graphDataStore.update(map => {
        const current = map.get(transferId) || { dataPoints: [], maxSpeed: 0 };

        // Only add if we have new progress
        if (current.dataPoints.length === 0 || (current.dataPoints.at(-1)?.bytesCopied !== bytesCopied)) {
            const newDataPoints = [...current.dataPoints, { bytesCopied, speed }];
            const newMaxSpeed = Math.max(current.maxSpeed, speed);

            map.set(transferId, {
                dataPoints: newDataPoints,
                maxSpeed: newMaxSpeed
            });
        }
        return map;
    });
}
