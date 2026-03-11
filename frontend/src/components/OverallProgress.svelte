<script lang="ts">
    import {
        formatBytes,
        formatDuration,
        formatSpeed,
    } from "../utils/formatters";

    const { transfer, currentSpeed } = $props<{ transfer: any; currentSpeed: number }>();

    let percentage = $derived(
        transfer && transfer.totalBytes > 0
            ? (transfer.bytesCopied / transfer.totalBytes) * 100
            : 0
    );

    const eta = $derived(
        currentSpeed > 0 && transfer
            ? (transfer.totalBytes - transfer.bytesCopied) / currentSpeed
            : 0
    );
</script>

<div class="progress-panel">
    <div class="info-row">
        <div class="label-col">
            <div class="path" title={transfer?.destination || ""}>
                {transfer?.destination || "No destination"}
            </div>
        </div>
        <div class="stats-col">
            {#if transfer}
                <span class="speed">{formatSpeed(currentSpeed)}</span>
                {#if eta > 0}
                    <span class="eta">- {formatDuration(eta)}</span>
                {/if}
                <span class="size-progress">
                    {formatBytes(transfer.bytesCopied)} / {formatBytes(
                        transfer.totalBytes,
                    )}
                </span>
                <span class="percentage">{percentage.toFixed(1)}%</span>
            {/if}
        </div>
    </div>

    <div class="bar-container">
        <div 
            class="progress-bar" 
            style="width: {percentage}%"
        ></div>
    </div>
</div>

<style>
    .progress-panel {
        background-color: var(--bg-secondary);
        padding: 12px 20px;
        border-bottom: 1px solid var(--border-color);
    }

    .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
        font-size: 13px;
    }

    .label-col {
        flex: 1;
        overflow: hidden;
        margin-right: 20px;
    }

    .path {
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        font-weight: 500;
    }

    .stats-col {
        display: flex;
        gap: 15px;
        color: var(--text-secondary);
        white-space: nowrap;
    }

    .percentage {
        color: var(--accent-color);
        font-weight: bold;
        min-width: 45px;
        text-align: right;
    }

    .bar-container {
        height: 12px;
        background-color: var(--track-color); /* SPEC: Track #2D2D2D */
        border-radius: 2px;
        overflow: hidden;
    }

    .progress-bar {
        height: 100%;
        background-color: var(--accent-color); /* SPEC: Fill #0078D4 */
        transition: width 0.2s ease-out;
    }
</style>
