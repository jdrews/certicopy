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
            <div
                class="transfer-item {activeTransfer &&
                activeTransfer.id === transfer.id
                    ? 'selected'
                    : ''}"
                onclick={() => selectTransfer(transfer)}
            >
                <div class="transfer-icon">
                    {#if transfer.status === "in_progress"}
                        <span class="icon-active">⟳</span>
                    {:else if transfer.status === "success"}
                        <span class="icon-success">✓</span>
                    {:else if transfer.status === "failed"}
                        <span class="icon-failed">⚠</span>
                    {:else}
                        <span class="icon-pending">⋯</span>
                    {/if}
                </div>
                <div class="transfer-details">
                    <div class="transfer-name">
                        {transfer.name || "Untitled Transfer"}
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
                </div>
            </div>
        {/each}
        {#if queue.length === 0}
            <div class="empty-state">No transfers</div>
        {/if}
    </div>
</div>

<style>
    .sidebar {
        width: 250px;
        background-color: var(--bg-secondary);
        border-right: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        flex: 1; /* Fill available space */
        min-height: 0; /* Allow scrolling */
    }

    .section-title {
        padding: 10px 15px;
        font-size: var(--font-size-sm);
        text-transform: uppercase;
        color: var(--text-secondary);
        font-weight: bold;
        border-bottom: 1px solid var(--border-color);
    }

    .transfer-list {
        flex: 1;
        overflow-y: auto;
    }

    .transfer-item {
        display: flex;
        padding: 10px 15px;
        border-bottom: 1px solid var(--border-color);
        cursor: pointer;
        transition: background-color 0.2s;
    }

    .transfer-item:hover {
        background-color: var(--bg-tertiary);
    }

    .transfer-item.selected {
        background-color: var(--bg-tertiary);
        border-left: 3px solid var(--accent-color);
    }

    .transfer-icon {
        margin-right: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 16px;
    }

    .icon-active {
        color: var(--accent-color);
        animation: spin 2s linear infinite;
    }
    .icon-success {
        color: var(--success-color);
    }
    .icon-failed {
        color: var(--error-color);
    }
    .icon-pending {
        color: var(--text-secondary);
    }

    .transfer-details {
        flex: 1;
        overflow: hidden;
    }

    .transfer-name {
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        margin-bottom: 2px;
    }

    .transfer-info {
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
    }

    .empty-state {
        padding: 20px;
        text-align: center;
        color: var(--text-secondary);
        font-size: var(--font-size-sm);
    }

    @keyframes spin {
        0% {
            transform: rotate(0deg);
        }
        100% {
            transform: rotate(360deg);
        }
    }
</style>
