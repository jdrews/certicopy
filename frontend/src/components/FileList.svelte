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

    const counts = $derived({
        all: files.length,
        success: files.filter((f: any) => f.status === "success").length,
        failed: files.filter((f: any) => f.status === "failed" || f.status === "paused").length,
    });

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

    // --- Column Resizing Logic ---
    let colWidths = $state({
        name: 200,
        sourceHash: 110,
        destHash: 110,
        size: 90,
        message: 400,
    });

    let resizingCol: string | null = null;
    let startX = 0;
    let startWidth = 0;

    function startResize(e: MouseEvent, col: string) {
        e.preventDefault();
        resizingCol = col;
        startX = e.pageX;
        startWidth = (colWidths as any)[col];

        window.addEventListener("mousemove", handleResize);
        window.addEventListener("mouseup", stopResize);
        document.body.style.cursor = "col-resize";
        document.body.style.userSelect = "none";
    }

    function handleResize(e: MouseEvent) {
        if (!resizingCol) return;
        const diff = e.pageX - startX;
        (colWidths as any)[resizingCol] = Math.max(50, startWidth + diff);
    }

    function stopResize() {
        resizingCol = null;
        window.removeEventListener("mousemove", handleResize);
        window.removeEventListener("mouseup", stopResize);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
    }

    const gridTemplate = $derived(
        `35px ${colWidths.name}px ${colWidths.sourceHash}px ${colWidths.destHash}px ${colWidths.size}px ${colWidths.message}px 40px`,
    );
</script>

<div class="file-list-container">
    <div class="list-tabs">
        <button
            class="tab-btn {filter === 'all' ? 'active' : ''}"
            onclick={() => (filter = "all")}>All ({counts.all})</button
        >
        <button
            class="tab-btn {filter === 'success' ? 'active' : ''}"
            onclick={() => (filter = "success")}>Succeeded ({counts.success})</button
        >
        <button
            class="tab-btn {filter === 'failed' ? 'active' : ''}"
            onclick={() => (filter = "failed")}>Failed ({counts.failed})</button
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

    <div class="file-rows" style="--grid-template: {gridTemplate}">
        <div class="header-row">
            <div class="col-icon-container"></div>
            <div class="col-header">
                Name
                <div
                    class="resizer"
                    role="button"
                    tabindex="-1"
                    aria-label="Resize column"
                    onmousedown={(e) => startResize(e, "name")}
                ></div>
            </div>
            <div class="col-header">
                Source Hash
                <div
                    class="resizer"
                    role="button"
                    tabindex="-1"
                    aria-label="Resize column"
                    onmousedown={(e) => startResize(e, "sourceHash")}
                ></div>
            </div>
            <div class="col-header">
                Dest Hash
                <div
                    class="resizer"
                    role="button"
                    tabindex="-1"
                    aria-label="Resize column"
                    onmousedown={(e) => startResize(e, "destHash")}
                ></div>
            </div>
            <div class="col-header">
                Size
                <div
                    class="resizer"
                    role="button"
                    tabindex="-1"
                    aria-label="Resize column"
                    onmousedown={(e) => startResize(e, "size")}
                ></div>
            </div>
            <div class="col-header">
                Message
                <div
                    class="resizer"
                    role="button"
                    tabindex="-1"
                    aria-label="Resize column"
                    onmousedown={(e) => startResize(e, "message")}
                ></div>
            </div>
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
                <div class="col-icon-container">
                    {#if file.transferCompleted}
                        <div class="col-icon status-success" title="Transfer Complete">✓</div>
                    {:else if file.status === "failed"}
                         <div class="col-icon status-failed">{getStatusIcon(file.status)}</div>
                    {:else if file.status === "paused"}
                         <div class="col-icon status-paused">{getStatusIcon(file.status)}</div>
                    {:else}
                         <div class="col-icon {getStatusClass(file.status)}">{getStatusIcon(file.status)}</div>
                    {/if}

                    {#if file.status === "hashing"}
                        <div class="col-icon status-hashing animate-spin" title="Verifying Integrity...">⟳</div>
                    {:else if file.endHashVerified}
                        <div class="col-icon status-hashing" title="Integrity Verified">✓</div>
                    {/if}
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
        grid-template-columns: var(--grid-template);
        grid-column-gap: 0px; /* Separators handled by headers */
        background-color: var(--bg-secondary);
        padding: 8px 15px; /* Slightly reduced vertical padding */
        font-weight: 600;
        font-size: 13px;
        color: var(--text-secondary);
        border-bottom: 1px solid var(--border-color);
        position: sticky;
        top: 0;
        z-index: 10;
        text-transform: none;
    }

    .col-header {
        position: relative;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding: 0 10px;
        display: flex;
        align-items: center;
    }

    .header-row > div:not(:last-child) {
        border-right: 1px solid var(--border-color);
    }

    .resizer {
        position: absolute;
        right: -5px;
        top: 0;
        bottom: 0;
        width: 10px;
        cursor: col-resize;
        background-color: transparent;
        transition: background-color 0.2s;
        z-index: 20;
    }

    .resizer:hover,
    .resizer:active {
        background-color: var(--accent-color);
    }

    .file-rows {
        flex: 1;
        overflow-y: auto;
    }

    .file-row {
        display: grid;
        grid-template-columns: var(--grid-template);
        grid-column-gap: 0px; /* Match header */
        padding: 10px 15px;
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

    .col-icon-container {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 4px;
        width: 35px;
    }
    
    .file-row > div {
        padding: 0 10px; /* Align with header padding */
    }
    
    .file-row .col-icon-container,
    .file-row .col-action {
        padding: 0; /* Keep fixed columns non-padded relative to flex ones */
    }

    .col-icon {
        text-align: center;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 14px;
        min-width: 14px;
    }

    .col-name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        text-align: left;
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
    .status-hashing {
        color: #9d50da; /* SPEC: Purple for End Hash */
    }
    .status-pending {
        color: var(--text-tertiary);
    }

    .animate-spin {
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
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
