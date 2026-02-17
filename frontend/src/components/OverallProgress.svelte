<script lang="ts">
    import {
        formatBytes,
        formatDuration,
        formatSpeed,
    } from "../utils/formatters";

    export let transfer: any = null;

    $: percentage =
        transfer && transfer.totalBytes > 0
            ? (transfer.bytesCopied / transfer.totalBytes) * 100
            : 0;

    // Prevent NaN
    $: if (isNaN(percentage)) percentage = 0;

    // Mock data for speed/eta if not present in transfer object yet
    $: speed = transfer?.speed || 0;
    $: eta = transfer?.eta || 0;
</script>

<div class="progress-panel">
    <div class="info-row">
        <div class="label-col">
            <div class="path" title={transfer?.destPath || ""}>
                {transfer?.destPath || "No destination"}
            </div>
        </div>
        <div class="stats-col">
            {#if transfer}
                <span class="speed">{formatSpeed(speed)}</span>
                {#if eta > 0}
                    <span class="eta">- {formatDuration(eta)}</span>
                {/if}
                <span class="size-progress">
                    {formatBytes(transfer.bytesCopied)} / {formatBytes(
                        transfer.totalSize,
                    )}
                </span>
                <span class="percentage">{percentage.toFixed(1)}%</span>
            {/if}
        </div>
    </div>

    <div class="bar-container">
        <div class="progress-bar" style="width: {percentage}%"></div>
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
        background-color: #333;
        border-radius: 3px;
        overflow: hidden;
    }

    .progress-bar {
        height: 100%;
        background-color: var(--accent-color);
        transition: width 0.2s ease-out;
    }
</style>
