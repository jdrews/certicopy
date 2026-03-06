<script lang="ts">
    import { onMount } from "svelte";
    import * as App from "../../wailsjs/go/main/App";

    let { show = $bindable(false) } = $props<{ show?: boolean }>();

    let settings = $state({
        hashAlgorithm: "xxhash",
        bufferSize: 1048576, // 1MB
        overwrite: false,
    });

    const formattedBufferSize = $derived(() => {
        const bytes = settings.bufferSize;
        const kb = bytes / 1024;
        let text = "";
        if (kb >= 1024) {
            text = (kb / 1024).toFixed(1) + " MB";
        } else {
            text = kb.toFixed(0) + " KB";
        }

        // Indicate default for 1MB
        if (bytes === 1048576) {
            text += " (Default)";
        }
        return text;
    });

    onMount(async () => {
        const win = window as any;
        if (win.go?.main?.App) {
            try {
                const s = await App.GetSettings();
                if (s) {
                    settings.hashAlgorithm = s.hashAlgorithm;
                    settings.bufferSize = s.bufferSize;
                    settings.overwrite = s.overwrite;
                }
            } catch (e) {
                console.error("Failed to load settings:", e);
            }
        }
    });

    async function save() {
        try {
            settings.bufferSize = Number(settings.bufferSize);
            await App.SaveSettings(settings as any);
            show = false;
        } catch (e) {
            console.error("Failed to save settings:", e);
        }
    }

    function close() {
        show = false;
    }
</script>

{#if show}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="modal-backdrop" onclick={close}>
        <div class="modal" onclick={(e) => e.stopPropagation()}>
            <h2>Settings</h2>

            <div class="form-group">
                <div class="label-row">
                    <label for="hash">Hash Algorithm</label>
                </div>
                <select id="hash" bind:value={settings.hashAlgorithm}>
                    <option value="xxhash">xxHash (Fastest, Default)</option>
                    <option value="blake2b">BLAKE2b (Secure & Fast)</option>
                    <option value="sha256">SHA-256 (Standard)</option>
                    <option value="md5">MD5 (Legacy)</option>
                </select>
            </div>

            <div class="form-group checkbox-group">
                <input
                    id="overwrite"
                    type="checkbox"
                    bind:checked={settings.overwrite}
                />
                <label for="overwrite"
                    >Overwrite destination files if they exist</label
                >
            </div>

            <div class="form-group">
                <div class="label-row">
                    <label for="buffer">Buffer Size</label>
                    <span class="value">{formattedBufferSize()}</span>
                </div>
                <input
                    id="buffer"
                    type="range"
                    min="65536"
                    max="16777216"
                    step="65536"
                    bind:value={settings.bufferSize}
                />
            </div>

            <div class="actions">
                <button class="btn-cancel" onclick={close}>Cancel</button>
                <button class="btn-save" onclick={save}>Save Changes</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
        backdrop-filter: blur(4px);
    }
    .modal {
        background: var(--bg-secondary);
        color: var(--text-primary);
        padding: 24px;
        border-radius: 8px;
        width: 400px;
        border: 1px solid var(--border-color);
        box-shadow: 0 20px 50px rgba(0, 0, 0, 0.6);
    }
    h2 {
        margin-top: 0;
        margin-bottom: 24px;
        border-bottom: 1px solid var(--border-color);
        padding-bottom: 12px;
        font-size: 1.1rem;
        font-weight: 600;
        color: var(--text-primary);
        letter-spacing: 0.5px;
        text-transform: uppercase;
    }
    .form-group {
        margin-bottom: 28px;
    }
    .checkbox-group {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    .checkbox-group input[type="checkbox"] {
        width: 18px;
        height: 18px;
        cursor: pointer;
    }
    .checkbox-group label {
        margin-bottom: 0;
    }
    .label-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
        height: 20px;
    }
    label {
        display: block;
        font-weight: 600;
        font-size: 0.8rem;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    .value {
        font-size: 0.85rem;
        font-weight: 500;
        color: var(--text-primary);
        background: var(--bg-tertiary);
        padding: 2px 8px;
        border-radius: 4px;
        border: 1px solid var(--border-color);
    }

    select {
        width: 100%;
        padding: 10px 12px;
        background: var(--bg-tertiary);
        color: var(--text-primary) !important;
        border: 1px solid var(--border-color);
        border-radius: 4px;
        font-size: 0.9rem;
        outline: none;
        appearance: none;
        cursor: pointer;
        transition: border-color 0.2s;
    }

    select option {
        background: var(--bg-secondary);
        color: var(--text-primary);
        padding: 10px;
    }

    select:focus {
        border-color: var(--text-secondary);
    }

    input[type="range"] {
        width: 100%;
        height: 4px;
        background: var(--bg-tertiary);
        border-radius: 2px;
        outline: none;
        cursor: pointer;
        appearance: none;
        margin: 10px 0;
    }

    input[type="range"]::-webkit-slider-thumb {
        appearance: none;
        width: 16px;
        height: 16px;
        background: #888;
        border: 2px solid var(--bg-secondary);
        border-radius: 50%;
        cursor: pointer;
        transition:
            transform 0.1s ease,
            background 0.2s;
        box-shadow: 0 0 5px rgba(0, 0, 0, 0.3);
    }

    input[type="range"]::-webkit-slider-thumb:hover {
        transform: scale(1.2);
        background: #bbb;
    }

    .actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 36px;
    }
    button {
        padding: 8px 24px;
        border-radius: 4px;
        cursor: pointer;
        border: 1px solid var(--border-color);
        font-size: 0.85rem;
        font-weight: 600;
        transition: all 0.2s;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    .btn-cancel {
        background: transparent;
        color: var(--text-secondary);
    }
    .btn-cancel:hover {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-primary);
    }
    .btn-save {
        background: var(--bg-tertiary);
        color: var(--text-primary);
    }
    .btn-save:hover {
        background: #444;
        border-color: #666;
    }
</style>
