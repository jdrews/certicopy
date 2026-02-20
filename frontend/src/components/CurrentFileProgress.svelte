<script lang="ts">
    import { formatBytes } from "../utils/formatters";

    export let currentFile: any = null;
    export let currentFileIndex: number = 0;
    export let totalFiles: number = 0;
    export let transferStatus: string = "";

    // Determine what to show in the path field - empty for success/failure
    $: displayPath =
        currentFile?.sourcePath ||
        (transferStatus === "success" || transferStatus === "failed"
            ? ""
            : "Waiting...");
    $: percentage =
        currentFile && currentFile.size > 0
            ? (currentFile.bytesCopied / currentFile.size) * 100
            : 0;
</script>

<div class="progress-panel">
    <div class="info-row">
        <div class="label-col">
            <div class="path" title={currentFile?.sourcePath || ""}>
                {displayPath}
            </div>
        </div>
        <div class="stats-col">
            {#if currentFile}
                <span class="file-count">{currentFileIndex} / {totalFiles}</span
                >
                <span class="size-progress">
                    {formatBytes(currentFile.bytesCopied)} / {formatBytes(
                        currentFile.size,
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
        height: 6px;
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
