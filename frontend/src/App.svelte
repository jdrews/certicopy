<script lang="ts">
  import { onMount } from "svelte";
  import type { FileInfo } from "./lib/types";
  import TransferQueue from "./components/TransferQueue.svelte";
  import CurrentFileProgress from "./components/CurrentFileProgress.svelte";
  import OverallProgress from "./components/OverallProgress.svelte";
  import TransferGraph from "./components/TransferGraph.svelte";
  import FileList from "./components/FileList.svelte";
  import SettingsDialog from "./components/SettingsDialog.svelte";
  import "./styles/main.css";

  // Import Wails runtime
  import {
    AddTransferToQueue,
    CancelTransfer,
    GetQueue,
    PauseTransfer,
    ResumeTransfer,
    SelectDestination,
    SelectSource,
    StartQueue,
  } from "../wailsjs/go/main/App";

  import { appState } from "./lib/state.svelte";

  let showSettings = $state(false);

  // Actions
  onMount(() => {
    appState.initEventListeners();
    loadQueue();
  });

  async function loadQueue() {
    try {
      const queue = (await GetQueue()) || [];
      appState.setTransfers(queue as any);
      if (queue.length > 0 && !appState.activeTransferId) {
        appState.setActiveTransfer(queue[0].id);
      }
    } catch (e) {
      console.error("Failed to load queue:", e);
    }
  }

  function handleTransferSelect(transfer: any) {
    appState.setActiveTransfer(transfer ? transfer.id : null);
  }

  // --- Actions ---

  async function addNewTransfer() {
    try {
      const sources = await SelectSource();
      if (!sources || sources.length === 0) return;

      const dest = await SelectDestination();
      if (!dest) return;

      await AddTransferToQueue(sources, dest);
      console.log("Transfer added to queue");
      StartQueue();
      console.log("StartQueue called");
      loadQueue(); // Refresh queue
    } catch (e) {
      console.error("Failed to add transfer:", e);
    }
  }
</script>

<main class="app-layout">
  <!-- Left Sidebar -->
  <aside class="app-sidebar">
    <TransferQueue
      queue={appState.transfers}
      activeTransfer={appState.activeTransfer}
      onselect={handleTransferSelect}
    />
    <!-- Debug Add Button & Settings -->
    <div style="padding: 10px; display: flex; gap: 5px;">
      <button
        onclick={addNewTransfer}
        style="flex: 1; padding: 5px; opacity: 0.8; font-weight: bold;"
        >+ New Transfer</button
      >

      <button
        onclick={() => (showSettings = true)}
        style="padding: 5px 10px; opacity: 0.5;"
        title="Settings">⚙</button
      >
    </div>
  </aside>

  <!-- Main Content Area -->
  <section class="main-content">
    <!-- Top Progress Bars Area -->
    <div class="progress-area">
      <CurrentFileProgress
        currentFileIndex={appState.currentFile && appState.activeFiles
          ? appState.activeFiles.findIndex(
              (f: any) => f.sourcePath === appState.currentFile?.sourcePath,
            ) + 1
          : 0}
        totalFiles={appState.activeFiles.length}
        transferStatus={appState.activeTransfer?.status || ""}
      />
      <OverallProgress
        transfer={appState.activeTransfer}
        currentSpeed={appState.lastSpeed}
      />
      <TransferGraph
        transfer={appState.activeTransfer}
        currentSpeed={appState.lastSpeed}
      />

      <!-- Action Buttons Bar -->
      <div class="action-bar">
        <div class="actions-left">
          {#if appState.activeTransfer?.status === "in_progress"}
            <button class="btn btn-action" title="Pause" onclick={PauseTransfer}
              >Pause</button
            >
          {/if}
          {#if appState.activeTransfer?.status === "paused" || appState.activeTransfer?.status === "failed"}
            <button
              class="btn btn-action btn-success"
              title="Resume Transfer"
              onclick={ResumeTransfer}>Resume</button
            >
          {/if}
          {#if appState.activeTransfer?.status === "in_progress" || appState.activeTransfer?.status === "pending" || appState.activeTransfer?.status === "paused"}
            <button
              class="btn btn-action btn-danger"
              title="Cancel Transfer"
              onclick={CancelTransfer}>Cancel</button
            >
          {/if}
        </div>
      </div>
    </div>

    <!-- Bottom File List Area -->
    <div class="list-area">
      <FileList files={appState.activeFiles} />
    </div>
  </section>

  <!-- Settings Dialog -->
  <SettingsDialog bind:show={showSettings} />
</main>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
    width: 100vw;
    background-color: var(--bg-primary);
    color: var(--text-primary);
    overflow: hidden;
  }

  .app-sidebar {
    width: 250px;
    flex-shrink: 0;
    z-index: 10;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0; /* Prevent flex child from overflowing */
  }

  .progress-area {
    display: flex;
    flex-direction: column;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
    z-index: 5;
  }

  .action-bar {
    background-color: var(--bg-secondary);
    padding: 8px 15px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .actions-left {
    display: flex;
    gap: 10px;
  }

  .btn-action {
    background-color: var(--bg-tertiary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    padding: 5px 15px;
    font-size: 13px;
    border-radius: 3px;
  }

  .btn-action:hover {
    background-color: #444;
  }

  .btn-danger:hover {
    background-color: #a00;
    color: white;
    border-color: #a00;
  }

  .btn-success:hover {
    background-color: #080;
    color: white;
    border-color: #080;
  }

  .list-area {
    flex: 1;
    overflow: hidden;
    position: relative;
    background-color: var(--bg-primary);
  }
</style>
