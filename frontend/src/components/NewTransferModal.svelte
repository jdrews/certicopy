<script lang="ts">
    import { SelectSource, SelectDestination } from "../../wailsjs/go/main/App";
    import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";

    interface Props {
        show: boolean;
        onclose: () => void;
        onstart: (
            source: string,
            destination: string,
            overwrite: boolean,
        ) => void;
    }

    let { show, onclose, onstart } = $props();

    let sourcePath = $state("");
    let destPath = $state("");
    let overwrite = $state(false);

    let sourceZone: HTMLElement | undefined = $state();
    let destZone: HTMLElement | undefined = $state();

    $effect(() => {
        if (show) {
            // Load default overwrite from settings
            import("../../wailsjs/go/main/App").then(async (App) => {
                const s = await App.GetSettings();
                if (s) {
                    overwrite = s.overwrite;
                }
            });

            const cleanup = EventsOn(
                "wails:file-drop",
                (x: number, y: number, paths: string[]) => {
                    if (!paths || paths.length === 0) return;

                    const srcRect = sourceZone?.getBoundingClientRect();
                    const destRect = destZone?.getBoundingClientRect();

                    if (
                        srcRect &&
                        x >= srcRect.left &&
                        x <= srcRect.right &&
                        y >= srcRect.top &&
                        y <= srcRect.bottom
                    ) {
                        sourcePath = paths[0];
                    } else if (
                        destRect &&
                        x >= destRect.left &&
                        x <= destRect.right &&
                        y >= destRect.top &&
                        y <= destRect.bottom
                    ) {
                        destPath = paths[0];
                    }
                },
            );
            return () => {
                EventsOff("wails:file-drop");
            };
        }
    });

    async function pickSource() {
        const result = await SelectSource();
        if (result && result.length > 0) {
            sourcePath = result[0];
        }
    }

    async function pickDest() {
        const result = await SelectDestination();
        if (result) {
            destPath = result;
        }
    }

    function handleStart() {
        if (sourcePath && destPath) {
            onstart(sourcePath, destPath, overwrite);
            reset();
        }
    }

    function handleCancel() {
        reset();
        onclose();
    }

    async function reset() {
        sourcePath = "";
        destPath = "";
        try {
            const App = await import("../../wailsjs/go/main/App");
            const s = await App.GetSettings();
            overwrite = s ? s.overwrite : false;
        } catch (e) {
            overwrite = false;
        }
    }

    function handleDrop(event: DragEvent, type: "source" | "dest") {
        event.preventDefault();
        const files = event.dataTransfer?.files;
        if (files && files.length > 0) {
            console.log(`Dropped into ${type}:`, files[0].name);
        }
    }

    function handleDragOver(event: DragEvent) {
        event.preventDefault();
    }
</script>

{#if show}
    <div class="modal-overlay" onclick={handleCancel} aria-hidden="true">
        <div
            class="modal-content"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.key === "Escape" && handleCancel()}
            role="dialog"
            tabindex="-1"
        >
            <div class="modal-header">
                <h2>New Transfer</h2>
                <button class="close-btn" onclick={handleCancel}>&times;</button
                >
            </div>

            <div class="modal-body">
                <div class="transfer-columns">
                    <!-- Source Column -->
                    <div class="column">
                        <h3>Source</h3>
                        <div class="path-input-group">
                            <input
                                type="text"
                                bind:value={sourcePath}
                                placeholder="Enter source path..."
                            />
                            <button
                                class="icon-btn"
                                onclick={pickSource}
                                title="Open Folder Picker"
                            >
                                <svg
                                    class="folder-icon"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                >
                                    <path
                                        d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                                    ></path>
                                </svg>
                            </button>
                        </div>
                        <div
                            bind:this={sourceZone}
                            class="drop-zone"
                            class:has-path={sourcePath !== ""}
                            onclick={pickSource}
                            ondragover={handleDragOver}
                            ondrop={(e) => handleDrop(e, "source")}
                            aria-hidden="true"
                        >
                            {#if sourcePath}
                                <div class="path-display">{sourcePath}</div>
                            {:else}
                                <div class="plus-icon">+</div>
                                <p>Click or drop source folder here</p>
                            {/if}
                        </div>
                    </div>

                    <!-- Arrow/Separator -->
                    <div class="separator">
                        <span class="arrow">→</span>
                    </div>

                    <!-- Destination Column -->
                    <div class="column">
                        <h3>Destination</h3>
                        <div class="path-input-group">
                            <input
                                type="text"
                                bind:value={destPath}
                                placeholder="Enter destination path..."
                            />
                            <button
                                class="icon-btn"
                                onclick={pickDest}
                                title="Open Folder Picker"
                            >
                                <svg
                                    class="folder-icon"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                >
                                    <path
                                        d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                                    ></path>
                                </svg>
                            </button>
                        </div>
                        <div
                            bind:this={destZone}
                            class="drop-zone"
                            class:has-path={destPath !== ""}
                            onclick={pickDest}
                            ondragover={handleDragOver}
                            ondrop={(e) => handleDrop(e, "dest")}
                            aria-hidden="true"
                        >
                            {#if destPath}
                                <div class="path-display">{destPath}</div>
                            {:else}
                                <div class="plus-icon">+</div>
                                <p>Click or drop destination folder here</p>
                            {/if}
                        </div>
                    </div>
                </div>

                <div class="transfer-options">
                    <div class="option-item">
                        <input
                            id="modal-overwrite"
                            type="checkbox"
                            bind:checked={overwrite}
                        />
                        <label for="modal-overwrite"
                            >Overwrite destination files if they exist</label
                        >
                    </div>
                </div>
            </div>

            <div class="modal-footer">
                <button class="btn btn-secondary" onclick={handleCancel}
                    >Cancel</button
                >
                <button
                    class="btn btn-primary"
                    disabled={!sourcePath || !destPath}
                    onclick={handleStart}
                >
                    Start Transfer
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.7);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
        backdrop-filter: blur(4px);
    }

    .modal-content {
        background-color: var(--bg-secondary);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        width: 800px;
        max-width: 90%;
        box-shadow:
            0 20px 25px -5px rgba(0, 0, 0, 0.5),
            0 10px 10px -5px rgba(0, 0, 0, 0.4);
        animation: modal-appear 0.3s ease-out;
    }

    @keyframes modal-appear {
        from {
            transform: scale(0.95);
            opacity: 0;
        }
        to {
            transform: scale(1);
            opacity: 1;
        }
    }

    .modal-header {
        padding: var(--spacing-md) var(--spacing-lg);
        border-bottom: 1px solid var(--border-color);
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .modal-header h2 {
        font-size: var(--font-size-xl);
        background: linear-gradient(135deg, #fff, #aaa);
        background-clip: text;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .close-btn {
        background: transparent;
        color: var(--text-secondary);
        font-size: 24px;
        padding: 0;
        line-height: 1;
    }

    .close-btn:hover {
        color: white;
    }

    .modal-body {
        padding: var(--spacing-lg);
    }

    .transfer-columns {
        display: flex;
        gap: var(--spacing-lg);
        align-items: stretch;
    }

    .column {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: var(--spacing-md);
    }

    .column h3 {
        font-size: var(--font-size-md);
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .path-input-group {
        display: flex;
        gap: var(--spacing-sm);
    }

    .path-input-group input {
        flex: 1;
        background-color: var(--bg-tertiary);
        border: 1px solid var(--border-color);
        color: var(--text-primary);
        padding: 8px 12px;
        border-radius: 6px;
        font-size: var(--font-size-sm);
        transition: border-color 0.2s;
    }

    .path-input-group input:focus {
        outline: none;
        border-color: var(--accent-color);
    }

    .icon-btn {
        background-color: var(--bg-tertiary);
        border: 1px solid var(--border-color);
        color: var(--text-primary);
        padding: 0 12px;
        display: flex;
        align-items: center;
        border-radius: 6px;
    }

    .folder-icon {
        width: 18px;
        height: 18px;
        color: var(--text-secondary);
        transition: color 0.2s;
    }

    .icon-btn:hover .folder-icon {
        color: var(--text-primary);
    }

    .separator {
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--border-color);
    }

    .arrow {
        font-size: 32px;
        font-weight: bold;
    }

    .drop-zone {
        flex: 1;
        min-height: 200px;
        border: 2px dashed var(--border-color);
        border-radius: 12px;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        cursor: pointer;
        transition: all 0.3s ease;
        background-color: rgba(255, 255, 255, 0.02);
        text-align: center;
        padding: var(--spacing-lg);
    }

    .drop-zone:hover {
        border-color: var(--accent-color);
        background-color: rgba(0, 122, 204, 0.05);
    }

    .drop-zone.has-path {
        border-style: solid;
        border-color: var(--success-color);
        background-color: rgba(78, 201, 176, 0.03);
    }

    .plus-icon {
        font-size: 48px;
        color: var(--border-color);
        margin-bottom: var(--spacing-sm);
        transition: color 0.3s;
    }

    .drop-zone:hover .plus-icon {
        color: var(--accent-color);
    }

    .drop-zone p {
        color: var(--text-secondary);
        font-size: var(--font-size-sm);
        margin: 0;
    }

    .path-display {
        word-break: break-all;
        font-family: "Courier New", Courier, monospace;
        font-size: var(--font-size-sm);
        color: var(--text-primary);
    }

    .transfer-options {
        margin-top: var(--spacing-lg);
        padding-top: var(--spacing-md);
        border-top: 1px solid var(--border-color);
    }

    .option-item {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .option-item label {
        color: var(--text-secondary);
        font-size: var(--font-size-sm);
        cursor: pointer;
    }

    .option-item input[type="checkbox"] {
        cursor: pointer;
    }

    .modal-footer {
        padding: var(--spacing-md) var(--spacing-lg);
        border-top: 1px solid var(--border-color);
        display: flex;
        justify-content: flex-end;
        gap: var(--spacing-md);
    }

    .btn {
        padding: 10px 24px;
        font-weight: 600;
        font-size: var(--font-size-md);
        transition: all 0.2s;
    }

    .btn-primary {
        background-color: var(--accent-color);
        color: white;
    }

    .btn-primary:hover:not(:disabled) {
        background-color: var(--accent-hover);
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(0, 122, 204, 0.3);
    }

    .btn-primary:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn-secondary {
        background-color: transparent;
        color: var(--text-secondary);
        border: 1px solid var(--border-color);
    }

    .btn-secondary:hover {
        background-color: var(--bg-tertiary);
        color: white;
    }
</style>
