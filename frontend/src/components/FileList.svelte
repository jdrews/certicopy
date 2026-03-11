<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { formatBytes } from "../utils/formatters";
    import { appState } from "../lib/state.svelte";
    import type { FileStatus } from "../lib/types";
    import {
        RemoveFileFromJob,
        ResumeTransfer,
    } from "../../wailsjs/go/main/App";

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

    async function removeFile(jobId: string, sourcePath: string) {
        try {
            await RemoveFileFromJob(jobId, sourcePath);
        } catch (e) {
            console.error("Failed to remove file:", e);
        }
    }

    async function retryFailedFiles() {
        if (filteredFiles.length > 0 && filter === "failed") {
            // All files in a given view typically share the same jobId, grab the first one
            const jobId = filteredFiles[0].jobId;
            await ResumeTransfer(jobId);
        }
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
        {#if filter === "failed" && filteredFiles.length > 0}
            <button
                class="retry-btn"
                onclick={retryFailedFiles}
                title="Retry all failed files"
            >
                ⟳ Retry All
            </button>
        {/if}
    </div>

    <div class="file-rows">
        <div class="header-row">
            <div class="col-icon"></div>
            <div class="col-name">Name</div>
            <div class="col-hash">Source Hash</div>
            <div class="col-hash">Dest Hash</div>
            <div class="col-size">Size</div>
            <div class="col-msg">Message</div>
            <div class="col-action"></div>
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
                <div class="col-msg" title={file.errorMessage || file.status}>
                    {#if file.errorCode}
                        <span class="error-badge" title={file.errorCode}
                            >{file.errorCode}</span
                        >
                    {/if}
                    {file.errorMessage || file.status}
                </div>
                <div class="col-action">
                    {#if file.status === "failed" || file.status === "paused"}
                        <button
                            class="action-btn remove-btn"
                            onclick={() =>
                                removeFile(file.jobId, file.sourcePath)}
                            title="Remove file from transfer job"
                        >
                            ✕
                        </button>
                    {/if}
                </div>
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
        background-color: var(--bg-primary); /* SPEC: Main Background #121212 */
        font-size: 13px;
    }

    .list-tabs {
        display: flex;
        background-color: var(--bg-secondary); /* SPEC: Surface/Sidebar #1E1E1E */
        border-bottom: 1px solid var(--border-color);
    }

    .tab-btn {
        padding: 8px 16px;
        background: none;
        border: none;
        color: var(--text-secondary);
        cursor: pointer;
        font-size: 12px;
        transition: color 0.2s, background-color 0.2s;
    }

    .tab-btn:hover {
        color: var(--text-primary);
        background-color: var(--bg-hover);
    }

    .tab-btn.active {
        color: var(--text-primary);
        background-color: var(--bg-primary);
        border-bottom: 2px solid var(--accent-color);
        font-weight: 600;
    }

    .header-row {
        display: grid;
        grid-template-columns: 35px 2fr 100px 100px 90px 1fr 40px;
        grid-column-gap: 12px;
        background-color: var(--bg-secondary);
        padding: 10px 15px;
        font-weight: 600;
        font-size: 13px; /* Standardized to match content feel */
        color: var(--text-secondary);
        border-bottom: 1px solid var(--border-color);
        position: sticky;
        top: 0;
        z-index: 10;
        text-transform: none; /* Ensure no unintended casing */
    }

    .file-rows {
        flex: 1;
        overflow-y: auto;
    }

    .file-row {
        display: grid;
        grid-template-columns: 35px 2fr 100px 100px 90px 1fr 40px;
        grid-column-gap: 12px;
        padding: 10px 15px; /* SPEC: Increased vertical padding to 10px */
        border-bottom: 1px solid var(--border-color);
        align-items: center;
        transition: background-color 0.1s;
        border-left: 3px solid transparent;
    }

    .file-row:hover {
        background-color: var(--bg-hover); /* SPEC: Hover Highlight #2A2D2E */
    }

    .row-active {
        border-left: 3px solid var(--accent-color); /* SPEC: In Progress left-border accent */
        background-color: rgba(0, 120, 212, 0.05);
    }

    .row-failed {
        background-color: rgba(244, 67, 54, 0.05);
    }
    .row-paused {
        background-color: rgba(245, 158, 11, 0.05);
    }
    .row-success {
        border-left: 3px solid var(--success-color);
    }

    .col-icon {
        text-align: center;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 14px;
    }

    .col-name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .file-row .col-name {
        color: #FFFFFF; /* SPEC: File Names White */
        font-weight: 600; /* SPEC: Semi-bold */
    }

    .col-hash {
        display: flex;
        align-items: center;
        gap: 5px;
        overflow: hidden;
    }

    .file-row .col-hash {
        font-family: var(--font-mono); /* SPEC: JetBrains Mono */
        color: var(--text-secondary); /* SPEC: Mid-gray #969696 */
        font-size: 11px; /* SPEC: 0.9em size */
    }

    .col-size {
        text-align: right;
        color: var(--text-secondary);
    }

    .col-msg {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .file-row .col-msg {
        color: var(--text-tertiary);
    }

    .col-action {
        text-align: right;
    }

    .status-success {
        color: var(--success-color); /* SPEC: Emerald #10B981 */
    }
    .status-failed {
        color: var(--error-color);
    }
    .status-active {
        color: var(--accent-color);
    }
    .status-paused {
        color: var(--warning-color);
    }
    .status-pending {
        color: var(--text-tertiary);
    }

    .hash-mismatch {
        color: var(--warning-color);
        font-weight: bold;
    }

    .empty-state {
        padding: 40px;
        text-align: center;
        color: var(--text-tertiary);
        font-style: italic;
    }

    .error-badge {
        background-color: rgba(244, 67, 54, 0.1);
        color: var(--error-color);
        padding: 2px 6px;
        border-radius: 2px;
        font-size: 10px;
        font-weight: bold;
        border: 1px solid rgba(244, 67, 54, 0.2);
    }

    .action-btn {
        background: none;
        border: none;
        cursor: pointer;
        opacity: 0.5;
        transition: opacity 0.2s;
        color: var(--text-secondary);
        font-size: 14px;
    }
    .action-btn:hover {
        opacity: 1;
    }
    .remove-btn:hover {
        color: var(--error-color);
    }

    .retry-btn {
        margin-left: auto;
        background-color: var(--accent-color);
        color: white;
        border: none;
        padding: 4px 12px;
        font-size: 12px;
        cursor: pointer;
        border-radius: 2px;
    }
    .retry-btn:hover {
        background-color: var(--accent-muted);
    }
</style>
