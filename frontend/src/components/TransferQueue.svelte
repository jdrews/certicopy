<script lang="ts">
    import { formatBytes } from "../utils/formatters";
    import type { TransferJob } from "../lib/types";

    const { queue, activeTransfer, onselect } = $props<{
        queue: TransferJob[];
        activeTransfer: TransferJob | null;
        onselect?: (transfer: TransferJob) => void;
    }>();

    function selectTransfer(transfer: TransferJob) {
        onselect?.(transfer);
    }
</script>

<div class="sidebar">
    <div class="section-title">Transfers</div>

    <div class="transfer-list">
        {#each queue as transfer}
            <button
                type="button"
                class="transfer-item {activeTransfer &&
                activeTransfer.id === transfer.id
                    ? 'selected'
                    : ''}"
                onclick={() => selectTransfer(transfer)}
                onkeydown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        selectTransfer(transfer);
                    }
                }}
            >
                <div class="transfer-icon">
                    {#if transfer.status === "in_progress"}
                        <span class="icon-active">⟳</span>
                    {:else if transfer.status === "paused"}
                        <span class="icon-paused">⏸</span>
                    {:else if transfer.status === "success"}
                        <span class="icon-success">✓</span>
                    {:else if transfer.status === "failed"}
                        <span class="icon-failed">⚠</span>
                    {:else}
                        <span class="icon-pending">⋯</span>
                    {/if}
                </div>
                <div class="transfer-details">
                    <div
                        class="transfer-source"
                        title={"Src: " + (transfer.sources[0] || "Unknown")}
                    >
                        {transfer.sources[0] || "Unknown Source"}
                    </div>
                    <div
                        class="transfer-dest"
                        title={"Dst: " + transfer.destination}
                    >
                        {transfer.destination}
                    </div>
                    <div class="transfer-info">
                        <span class="size"
                            >{formatBytes(transfer.totalBytes)}</span
                        >
                        {#if transfer.status === "in_progress"}
                            • <span class="percent"
                                >{transfer.totalBytes > 0
                                    ? Math.round(
                                          (transfer.bytesCopied /
                                              transfer.totalBytes) *
                                              100,
                                      )
                                    : 0}%</span
                            >
                        {/if}
                    </div>
                    {#if transfer.error}
                        <div class="transfer-error" title={transfer.error}>
                            <span class="error-badge"
                                >{transfer.errorCode || "Error"}</span
                            >
                            {transfer.error}
                        </div>
                    {/if}
                </div>
            </button>
        {/each}
        {#if queue.length === 0}
            <div class="empty-state">No transfers</div>
        {/if}
    </div>
</div>

<style>
    .sidebar {
        width: 250px;
        background-color: var(--bg-secondary); /* SPEC: Surface #1E1E1E */
        border-right: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        flex: 1;
        min-height: 0;
    }

    .section-title {
        padding: 10px 15px;
        font-size: var(--font-size-sm);
        text-transform: uppercase;
        color: var(--text-secondary); /* SPEC: #969696 */
        font-weight: bold;
        border-bottom: 1px solid var(--border-color);
    }

    .transfer-list {
        flex: 1;
        overflow-y: auto;
    }

    .transfer-item {
        display: flex;
        padding: 12px 15px; /* Increased padding */
        border: none;
        border-bottom: 1px solid var(--border-color);
        background: transparent;
        width: 100%;
        text-align: left;
        font: inherit;
        color: inherit;
        cursor: pointer;
        transition: background-color 0.1s;
        border-left: 3px solid transparent;
    }

    .transfer-item:hover {
        background-color: var(--bg-hover); /* SPEC: #2A2D2E */
    }

    .transfer-item.selected {
        background-color: var(--bg-hover);
        border-left: 3px solid var(--accent-color); /* SPEC: #0078D4 */
    }

    .transfer-icon {
        margin-right: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 16px;
    }

    .icon-active {
        color: var(--accent-color);
        animation: spin 2s linear infinite;
    }
    .icon-paused {
        color: var(--warning-color);
    }
    .icon-success {
        color: var(--success-color); /* SPEC: #10B981 */
    }
    .icon-failed {
        color: var(--error-color);
    }
    .icon-pending {
        color: var(--text-tertiary);
    }

    .transfer-details {
        flex: 1;
        overflow: hidden;
    }

    .transfer-source {
        font-weight: 600;
        font-size: 13px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-bottom: 2px;
        color: var(--text-primary);
    }

    .transfer-dest {
        font-size: 11px;
        color: var(--text-secondary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-bottom: 4px;
    }

    .transfer-info {
        font-size: 11px;
        color: var(--text-tertiary);
    }

    .empty-state {
        padding: 20px;
        text-align: center;
        color: var(--text-tertiary);
        font-size: var(--font-size-sm);
    }

    .transfer-error {
        margin-top: 4px;
        font-size: 10px;
        color: var(--error-color);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        display: flex;
        align-items: center;
        gap: 4px;
    }

    .error-badge {
        background-color: rgba(244, 67, 54, 0.1);
        color: var(--error-color);
        padding: 1px 4px;
        border-radius: 2px;
        font-size: 9px;
        font-weight: bold;
        border: 1px solid rgba(244, 67, 54, 0.2);
    }

    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
</style>
