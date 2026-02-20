<script lang="ts">
    import { formatBytes } from "../utils/formatters";

    import {
        transferMetricsStore,
        updateGraphData,
        type DataPoint,
    } from "../utils/stores";

    export let transfer: any = null;
    export let currentSpeed: number = 0;

    let dataPoints: DataPoint[] = [];
    let maxSpeed = 0;

    // Update view when transfer changes or store updates
    $: {
        if (transfer?.id) {
            const metrics = $transferMetricsStore.get(transfer.id);
            if (metrics) {
                dataPoints = metrics.dataPoints;
                maxSpeed = metrics.maxSpeed;
            } else {
                dataPoints = [];
                maxSpeed = 0;
            }
        }
    }

    // Generate SVG path with bytes-based x-axis
    $: pathD = generatePath(dataPoints, maxSpeed, transfer?.totalBytes || 0);

    function generatePath(
        data: DataPoint[],
        max: number,
        totalBytes: number,
    ): string {
        if (data.length < 2 || totalBytes <= 0) return "";

        const width = 100; // SVG viewBox percentage
        const height = 100;
        const yScale = max > 0 ? height / (max * 1.1) : 0; // Leave 10% headroom

        // Start at 0,0 (0 bytes transferred)
        let d = `M 0 ${height - data[0].speed * yScale}`;

        for (let i = 1; i < data.length; i++) {
            // X position based on percentage of total bytes transferred
            const x = (data[i].bytesCopied / totalBytes) * width;
            const y = height - data[i].speed * yScale;
            d += ` L ${x} ${y}`;
        }

        // Fill area - close the path at the bottom
        const lastX = (data[data.length - 1].bytesCopied / totalBytes) * width;
        d += ` L ${lastX} ${height} L 0 ${height} Z`;

        return d;
    }

    // Calculate progress percentage for display
    $: progressPercent =
        transfer?.totalBytes > 0
            ? Math.round((transfer.bytesCopied / transfer.totalBytes) * 100)
            : 0;

    // Check if transfer is complete with data
    $: isCompleteWithData =
        (transfer?.status === "success" || transfer?.status === "completed") &&
        dataPoints.length > 2;
</script>

<div class="graph-container">
    {#if dataPoints.length > 2}
        <svg viewBox="0 0 100 100" preserveAspectRatio="none">
            <defs>
                <linearGradient id="grad1" x1="0%" y1="0%" x2="0%" y2="100%">
                    <stop
                        offset="0%"
                        style="stop-color:var(--accent-color);stop-opacity:0.5"
                    />
                    <stop
                        offset="100%"
                        style="stop-color:var(--accent-color);stop-opacity:0.1"
                    />
                </linearGradient>
            </defs>
            <path
                d={pathD}
                fill="url(#grad1)"
                stroke="var(--accent-color)"
                stroke-width="1"
                vector-effect="non-scaling-stroke"
            />
        </svg>
        <div class="graph-label">
            {#if isCompleteWithData}
                Complete: {progressPercent}%
            {:else}
                Speed: {formatBytes(currentSpeed)}/s | {progressPercent}%
            {/if}
        </div>
    {:else if transfer?.status === "completed"}
        <div class="placeholder">Transfer completed</div>
    {:else}
        <div class="placeholder">Waiting for data...</div>
    {/if}
</div>

<style>
    .graph-container {
        height: 60px;
        background-color: var(--bg-primary);
        border-bottom: 1px solid var(--border-color);
        position: relative;
        overflow: hidden;
    }

    svg {
        width: 100%;
        height: 100%;
        display: block;
    }

    .graph-label {
        position: absolute;
        top: 5px;
        right: 10px;
        font-size: 11px;
        color: var(--text-secondary);
        background: rgba(0, 0, 0, 0.5);
        padding: 2px 5px;
        border-radius: 3px;
    }

    .placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        font-size: 12px;
        color: var(--text-tertiary);
        font-style: italic;
    }
</style>
