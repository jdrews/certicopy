<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { formatBytes } from "../utils/formatters";
    import { appState } from "../lib/state.svelte";
    import type { FileStatus } from "../lib/types";

    let { files = [], filter = $bindable("all") } = $props<{
        files: any[];
        filter?: "all" | "success" | "failed";
    }>();

    const filteredFiles = $derived(
        files.filter((f: any) => {
            if (filter === "all") return true;
            if (filter === "success") return f.status === "success";
            if (filter === "failed")
                return f.status === "failed" || f.status === "paused";
            return true;
        }),
    );

    function getStatusIcon(status: string) {
        switch (status) {
            case "success":
                return "✓";
            case "failed":
                return "⚠";
            case "in_progress":
                return "⟳";
            case "pending":
                return "⋯";
            case "skipped":
                return "↷";
            case "paused":
                return "⏸";
            default:
                return "";
        }
    }

    function getStatusClass(status: string) {
        switch (status) {
            case "success":
                return "status-success";
            case "failed":
                return "status-failed";
            case "in_progress":
                return "status-active";
            case "paused":
                return "status-paused";
            default:
                return "status-pending";
        }
    }

    // Helper to format hash for display (shortened)
    function formatHash(hash: string): string {
        if (!hash) return "....";
        if (hash.length > 8) {
            return `${hash.substring(0, 4)}:${hash.substring(hash.length - 4)}`;
        }
        return hash;
    }
</script>

<div class="file-list-container">
    <div class="list-tabs">
        <button
            class="tab-btn {filter === 'all' ? 'active' : ''}"
            onclick={() => (filter = "all")}>All</button
        >
        <button
            class="tab-btn {filter === 'success' ? 'active' : ''}"
            onclick={() => (filter = "success")}>Succeeded</button
        >
        <button
            class="tab-btn {filter === 'failed' ? 'active' : ''}"
            onclick={() => (filter = "failed")}>Failed</button
        >
    </div>

    <div class="file-rows">
        <div class="header-row">
            <div class="col-icon"></div>
            <div class="col-name">Name</div>
            <div class="col-hash">Source Hash</div>
            <div class="col-hash">Dest Hash</div>
            <div class="col-size">Size</div>
            <div class="col-msg">Message</div>
        </div>

        {#each filteredFiles as file (file.sourcePath)}
            <div
                class="file-row {file.status === 'in_progress'
                    ? 'row-active'
                    : ''} {file.status === 'failed'
                    ? 'row-failed'
                    : ''} {file.status === 'paused'
                    ? 'row-paused'
                    : ''} {file.status === 'success' ? 'row-success' : ''}"
            >
                <div class="col-icon {getStatusClass(file.status)}">
                    {getStatusIcon(file.status)}
                </div>
                <div class="col-name" title={file.sourcePath}>
                    {file.name}
                </div>
                <div class="col-hash monospace" title={file.sourceHash}>
                    {formatHash(file.sourceHash || "")}
                </div>
                <div class="col-hash monospace" title={file.destHash}>
                    {formatHash(file.destHash || "")}
                    {#if file.destHash && file.sourceHash && file.destHash !== file.sourceHash}
                        <span class="hash-mismatch" title="Hash Mismatch"
                            >⚠</span
                        >
                    {/if}
                </div>
                <div class="col-size">{formatBytes(file.size)}</div>
                <div class="col-msg">{file.errorMessage || file.status}</div>
            </div>
        {/each}
        {#if filteredFiles.length === 0}
            <div class="empty-state">No files to display</div>
        {/if}
    </div>
</div>

<style>
    .file-list-container {
        display: flex;
        flex-direction: column;
        height: 100%;
        background-color: var(--bg-secondary);
        font-size: 13px;
    }

    .list-tabs {
        display: flex;
        background-color: var(--bg-primary);
        border-bottom: 1px solid var(--border-color);
    }

    .tab-btn {
        padding: 6px 12px;
        background: none;
        border: none;
        color: var(--text-secondary);
        cursor: pointer;
        font-size: 12px;
    }

    .tab-btn:hover {
        color: var(--text-primary);
        background-color: var(--bg-tertiary);
    }

    .tab-btn.active {
        color: var(--text-primary);
        background-color: var(--bg-secondary);
        border-top: 2px solid var(--accent-color);
        font-weight: 500;
    }

    .header-row {
        display: grid;
        grid-template-columns: 30px 2fr 100px 100px 90px 1fr;
        grid-column-gap: 12px;
        background-color: var(--bg-tertiary);
        padding: 8px 15px;
        font-weight: 600;
        font-size: 12px;
        color: var(--text-secondary);
        border-bottom: 1px solid var(--border-color);
        position: sticky;
        top: 0;
        z-index: 10;
    }

    .file-rows {
        flex: 1;
        overflow-y: auto;
    }

    .file-row {
        display: grid;
        grid-template-columns: 30px 2fr 100px 100px 90px 1fr;
        grid-column-gap: 12px;
        padding: 6px 15px;
        border-bottom: 1px solid #333;
        align-items: center;
    }

    .file-row:hover {
        background-color: rgba(255, 255, 255, 0.05);
    }

    .row-active {
        background-color: rgba(0, 122, 204, 0.2);
    }

    .row-failed {
        background-color: rgba(244, 71, 71, 0.1);
    }
    .row-paused {
        background-color: rgba(245, 158, 11, 0.15);
    }
    .row-success {
        background-color: rgba(16, 185, 129, 0.15);
    }

    .col-icon {
        text-align: center;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .col-name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .col-hash {
        font-family: monospace;
        color: #aaa;
        display: flex;
        align-items: center;
        gap: 5px;
        overflow: hidden;
    }
    .col-size {
        text-align: right;
        color: #ccc;
    }
    .col-msg {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: #888;
    }

    .status-success {
        color: var(--success-color);
    }
    .status-failed {
        color: var(--error-color);
    }
    .status-active {
        color: var(--accent-color);
    }
    .status-paused {
        color: #f59e0b;
    }
    .status-pending {
        color: var(--text-secondary);
    }

    .hash-mismatch {
        color: var(--warning-color);
        font-weight: bold;
    }
    .monospace {
        font-family: "Consolas", "Monaco", monospace;
        font-size: 12px;
    }

    .empty-state {
        padding: 20px;
        text-align: center;
        color: var(--text-secondary);
        font-style: italic;
    }
</style>
