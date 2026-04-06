<script lang="ts">
    import { formatBytes } from "../utils/formatters";
    import { appState } from "../lib/state.svelte";
    import type { DataPoint } from "../lib/types";

    const { transfer, currentSpeed } = $props<{
        transfer: any;
        currentSpeed: number;
    }>();

    const metrics = $derived(
        transfer?.id
            ? appState.metrics[transfer.id] || {
                  dataPoints: [],
                  maxSpeed: 0,
              }
            : { dataPoints: [], maxSpeed: 0 },
    );

    // Generate SVG path with bytes-based x-axis
    const pathD = $derived(
        generatePath(
            metrics.dataPoints,
            metrics.maxSpeed,
            transfer?.totalBytes || 0,
        ),
    );

    function generatePath(
        data: DataPoint[],
        max: number,
        totalBytes: number,
    ): string {
        if (data.length < 2 || totalBytes <= 0) return "";

        const width = 100;
        const height = 100;
        const yScale = max > 0 ? height / (max * 1.1) : 0;

        let d = `M 0 ${height - data[0].speed * yScale}`;

        for (let i = 1; i < data.length; i++) {
            const x = (data[i].bytesCopied / totalBytes) * width;
            const y = height - data[i].speed * yScale;
            d += ` L ${x} ${y}`;
        }

        const lastX = (data[data.length - 1].bytesCopied / totalBytes) * width;
        d += ` L ${lastX} ${height} L 0 ${height} Z`;

        return d;
    }

    const progressPercent = $derived(
        transfer?.totalBytes > 0
            ? Math.round((transfer.bytesCopied / transfer.totalBytes) * 100)
            : 0,
    );

    const isCompleteWithData = $derived(
        (transfer?.status === "success" || transfer?.status === "completed") &&
            metrics.dataPoints.length > 2,
    );

    const isHashing = $derived(transfer?.status === "hashing");
    const strokeColor = $derived("var(--accent-color)");
</script>

<div class="graph-container">
    {#if metrics.dataPoints.length > 2}
        <svg viewBox="0 0 100 100" preserveAspectRatio="none">
            <defs>
                <linearGradient id="grad-{transfer?.id || 'default'}" x1="0%" y1="0%" x2="0%" y2="100%">
                    <stop
                        offset="0%"
                        style="stop-color:{strokeColor};stop-opacity:0.15"
                    />
                    <stop
                        offset="100%"
                        style="stop-color:transparent;stop-opacity:0"
                    />
                </linearGradient>
            </defs>

            <!-- SPEC: Horizontal Grid Lines #252525 at 50% opacity -->
            <g class="grid-lines">
                <line x1="0" y1="25" x2="100" y2="25" stroke="#252525" stroke-opacity="0.5" stroke-width="0.5" />
                <line x1="0" y1="50" x2="100" y2="50" stroke="#252525" stroke-opacity="0.5" stroke-width="0.5" />
                <line x1="0" y1="75" x2="100" y2="75" stroke="#252525" stroke-opacity="0.5" stroke-width="0.5" />
            </g>

            <path
                d={pathD}
                fill="url(#grad-{transfer?.id || 'default'})"
                stroke={strokeColor}
                stroke-width="1.5"
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
