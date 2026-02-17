<script lang="ts">
  import { onMount, tick } from "svelte";
  import TransferQueue from "./components/TransferQueue.svelte";
  import CurrentFileProgress from "./components/CurrentFileProgress.svelte";
  import OverallProgress from "./components/OverallProgress.svelte";
  import TransferGraph from "./components/TransferGraph.svelte";
  import FileList from "./components/FileList.svelte";
  import SettingsDialog from "./components/SettingsDialog.svelte";
  import "./styles/main.css";

  // Import Wails runtime
  import { EventsOn } from "../wailsjs/runtime/runtime";
  import {
    AddTransferToQueue,
    StartQueue,
    GetQueue,
    PauseTransfer,
    ResumeTransfer,
    CancelTransfer,
    SelectSource,
    SelectDestination,
  } from "../wailsjs/go/main/App";

  // State
  let transferQueue: any[] = [];
  let activeTransfer: any = null;
  let currentFile: any = null;
  let activeFiles: any[] = []; // Files for the currently selected transfer
  let showSettings = false;

  // Overall Stats
  let currentSpeed = 0;
  let estimatedTime = 0;

  onMount(async () => {
    // Initial Load
    await loadQueue();

    // Event Listeners
    EventsOn("queue:updated", (queue: any[]) => {
      transferQueue = queue;
      // If we have an active transfer, update it from the new queue
      if (activeTransfer) {
        const updated = queue.find((j) => j.id === activeTransfer.id);
        if (updated) {
          activeTransfer = updated;
          activeFiles = updated.Files || [];
          // Also update current file if it belongs to this transfer
          if (
            currentFile &&
            !activeFiles.find((f) => f.SourcePath === currentFile.SourcePath)
          ) {
            currentFile = null;
          }
        }
      } else if (queue.length > 0) {
        // Auto-select first if none selected
        handleTransferSelect({ detail: queue[0] });
      }
    });

    EventsOn("file:updated", (file: any) => {
      // Update file in our local list if it matches
      if (activeFiles) {
        const idx = activeFiles.findIndex(
          (f) => f.SourcePath === file.SourcePath,
        );
        if (idx !== -1) {
          activeFiles[idx] = file;
          // Trigger reactivity
          activeFiles = [...activeFiles];
        }
      }

      if (file.Status === "in_progress") {
        currentFile = file;
      } else if (
        currentFile &&
        currentFile.SourcePath === file.SourcePath &&
        file.Status !== "in_progress"
      ) {
        currentFile = null; // Clear if it finished
      }
    });

    EventsOn("transfer:progress", (progress: any) => {
      // Update speed/eta if we have an active transfer
      if (activeTransfer && activeTransfer.status === "in_progress") {
        currentSpeed = progress.CurrentSpeed;
        if (currentSpeed > 0) {
          estimatedTime =
            (activeTransfer.TotalBytes - activeTransfer.BytesCopied) /
            currentSpeed;
        }
      }
    });
  });

  async function loadQueue() {
    try {
      transferQueue = (await GetQueue()) || [];
      if (transferQueue.length > 0 && !activeTransfer) {
        handleTransferSelect({ detail: transferQueue[0] });
      }
    } catch (e) {
      console.error("Failed to load queue:", e);
    }
  }

  function handleTransferSelect(event: CustomEvent | { detail: any }) {
    activeTransfer = event.detail;
    activeFiles = activeTransfer ? activeTransfer.Files || [] : [];
  }

  // --- Actions ---

  async function addNewTransfer() {
    try {
      // Select source directory (backend now returns [path])
      const sources = await SelectSource();
      if (!sources || sources.length === 0) return;

      const dest = await SelectDestination();
      if (!dest) return;

      await AddTransferToQueue(sources, dest);
      console.log("Transfer added to queue");
      StartQueue();
      console.log("StartQueue called");
    } catch (e) {
      console.error("Failed to add transfer:", e);
    }
  }
</script>

<main class="app-layout">
  <!-- Left Sidebar -->
  <aside class="app-sidebar">
    <TransferQueue
      queue={transferQueue}
      {activeTransfer}
      on:select={handleTransferSelect}
    />
    <!-- Debug Add Button & Settings -->
    <div style="padding: 10px; display: flex; gap: 5px;">
      <button
        on:click={addNewTransfer}
        style="flex: 1; padding: 5px; opacity: 0.8; font-weight: bold;"
        >+ New Transfer</button
      >

      <button
        on:click={() => (showSettings = true)}
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
        {currentFile}
        currentFileIndex={currentFile && activeFiles
          ? activeFiles.findIndex(
              (f) => f.SourcePath === currentFile.SourcePath,
            ) + 1
          : 0}
        totalFiles={activeFiles.length}
      />
      <OverallProgress transfer={activeTransfer} />
      <TransferGraph transfer={activeTransfer} {currentSpeed} />

      <!-- Action Buttons Bar -->
      <div class="action-bar">
        <div class="actions-left">
          <button class="btn btn-action" title="Pause" on:click={PauseTransfer}
            >Pause</button
          >
          <button class="btn btn-action" title="Skip Current File">Skip</button>
          <button
            class="btn btn-action btn-danger"
            title="Stop Transfer"
            on:click={CancelTransfer}>Stop</button
          >
        </div>
      </div>
    </div>

    <!-- Bottom File List Area -->
    <div class="list-area">
      <FileList files={activeFiles} />
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

  .list-area {
    flex: 1;
    overflow: hidden;
    position: relative;
    background-color: var(--bg-primary);
  }
</style>
