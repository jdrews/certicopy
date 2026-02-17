<script lang="ts">
    import { onMount } from "svelte";
    import { formatBytes } from "../utils/formatters";

    export let transfer: any = null;
    export let currentSpeed: number = 0;

    let history: number[] = [];
    let maxSpeed = 0;
    const maxPoints = 100; // Keep last 100 points for the graph

    $: if (currentSpeed > 0) {
        history.push(currentSpeed);
        if (history.length > maxPoints) {
            history.shift();
        }
        history = history; // Trigger update

        // Update max speed for scaling, but slowly decay if speed drops
        if (currentSpeed > maxSpeed) {
            maxSpeed = currentSpeed;
        } else if (history.length > 0) {
            // Slowly adjust max speed to local maximum
            const localMax = Math.max(...history);
            if (localMax < maxSpeed) {
                maxSpeed = localMax;
            }
        }
    } else if (!transfer || transfer.status !== "in_progress") {
        // Reset or pause?
        // history = [];
    }

    // Generate SVG path
    $: pathD = generatePath(history, maxSpeed);

    function generatePath(data: number[], max: number): string {
        if (data.length < 2) return "";

        const width = 100; // SVG viewBox percentage
        const height = 100;

        // Scale factors
        const xStep = width / (maxPoints - 1);
        const yScale = max > 0 ? height / (max * 1.1) : 0; // Leave 10% headroom

        let d = `M 0 ${height - data[0] * yScale}`;

        for (let i = 1; i < data.length; i++) {
            const x = i * xStep;
            const y = height - data[i] * yScale;
            d += ` L ${x} ${y}`;
        }

        // Fill area
        d += ` L ${(data.length - 1) * xStep} ${height} L 0 ${height} Z`;

        return d;
    }
</script>

<div class="graph-container">
    {#if history.length > 2}
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
            Speed: {formatBytes(currentSpeed)}/s
        </div>
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
