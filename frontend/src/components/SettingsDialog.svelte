<script lang="ts">
    import { onMount } from "svelte";
    // Note: These imports will fail until 'wails dev' or 'wails build' is run to generate bindings
    // We'll trust they will be generated correctly matching App.go
    import { GetSettings, SaveSettings } from "../../wailsjs/go/main/App";

    let { show = $bindable(false) } = $props<{ show?: boolean }>();

    let settings = $state({
        theme: "dark",
        defaultDestPath: "",
        hashAlgorithm: "xxhash",
        bufferSize: 1048576, // 1MB (1024 * 1024)
        showNotifications: true,
        playSoundOnFinish: true,
        autoVerify: true,
    });

    // While bindings are not generated, we might want to mock GetSettings/SaveSettings locally for UI dev
    // or just rely on the fact that we will build soon.

    onMount(async () => {
        if (
            window["go"] &&
            window["go"]["main"] &&
            window["go"]["main"]["App"]
        ) {
            try {
                const s = await GetSettings();
                if (s) settings = s;
            } catch (e) {
                console.error("Failed to load settings:", e);
            }
        }
    });

    async function save() {
        try {
            settings.bufferSize = Number(settings.bufferSize);
            await SaveSettings(settings);
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
    <div class="modal-backdrop" onclick={close}>
        <div class="modal" stopPropagation.onclick>
            <h2>Settings</h2>

            <div class="form-group">
                <label for="theme">Theme</label>
                <select id="theme" bind:value={settings.theme}>
                    <option value="dark">Dark</option>
                    <option value="light">Light</option>
                </select>
            </div>

            <div class="form-group">
                <label for="hash">Hash Algorithm</label>
                <select id="hash" bind:value={settings.hashAlgorithm}>
                    <option value="xxhash">xxHash (Fastest)</option>
                    <option value="blake2b">BLAKE2b (Secure & Fast)</option>
                    <option value="sha256">SHA-256 (Standard)</option>
                    <option value="md5">MD5 (Legacy)</option>
                </select>
            </div>

            <div class="form-group">
                <label for="buffer">Buffer Size (bytes)</label>
                <input
                    id="buffer"
                    type="number"
                    bind:value={settings.bufferSize}
                />
            </div>

            <div class="form-group checkbox">
                <input
                    type="checkbox"
                    id="notify"
                    bind:checked={settings.showNotifications}
                />
                <label for="notify">Show Notifications</label>
            </div>

            <div class="form-group checkbox">
                <input
                    type="checkbox"
                    id="sound"
                    bind:checked={settings.playSoundOnFinish}
                />
                <label for="sound">Play Sound on Finish</label>
            </div>

            <div class="form-group checkbox">
                <input
                    type="checkbox"
                    id="verify"
                    bind:checked={settings.autoVerify}
                />
                <label for="verify">Auto-Verify After Copy</label>
            </div>

            <div class="actions">
                <button class="btn-cancel" onclick={close}>Cancel</button>
                <button class="btn-save" onclick={save}>Save</button>
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
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }
    .modal {
        background: var(--bg-secondary);
        color: var(--text-primary);
        padding: 20px;
        border-radius: 5px;
        width: 400px;
        border: 1px solid var(--border-color);
        box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
    }
    h2 {
        margin-top: 0;
        border-bottom: 1px solid var(--border-color);
        padding-bottom: 10px;
        font-size: 18px;
    }
    .form-group {
        margin-bottom: 15px;
    }
    label {
        display: block;
        margin-bottom: 5px;
        font-weight: 500;
        font-size: 13px;
    }
    input[type="text"],
    input[type="number"],
    select {
        width: 100%;
        padding: 8px;
        background: var(--bg-tertiary);
        color: var(--text-primary);
        border: 1px solid var(--border-color);
        border-radius: 3px;
        font-size: 13px;
    }
    .checkbox {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    .checkbox input {
        width: auto;
    }
    .checkbox label {
        margin-bottom: 0;
        cursor: pointer;
    }

    .actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 20px;
    }
    button {
        padding: 6px 16px;
        border-radius: 3px;
        cursor: pointer;
        border: none;
        font-size: 13px;
    }
    .btn-cancel {
        background: transparent;
        color: var(--text-secondary);
        border: 1px solid var(--border-color);
    }
    .btn-cancel:hover {
        background: var(--bg-tertiary);
        color: var(--text-primary);
    }
    .btn-save {
        background: var(--accent-color);
        color: white;
    }
    .btn-save:hover {
        filter: brightness(1.1);
    }
</style>
