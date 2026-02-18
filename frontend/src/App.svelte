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
      console.log("App: queue:updated event received", queue);
      transferQueue = queue;

      // If we have an active transfer, update it from the new queue
      if (activeTransfer) {
        console.log("App: Updating active transfer", activeTransfer.id);
        const updated = queue.find((j) => j.id === activeTransfer.id);
        if (updated) {
          activeTransfer = updated;
          activeFiles = updated.files || []; // JSON tag is 'files'
          console.log(
            "App: Updated active transfer files count:",
            activeFiles.length,
          );

          // Also update current file if it belongs to this transfer
          if (
            currentFile &&
            !activeFiles.find((f) => f.SourcePath === currentFile.SourcePath)
          ) {
            currentFile = null;
          }
        }
      }

      // Auto-select first if none selected
      if (!activeTransfer && queue.length > 0) {
        console.log("App: Auto-selecting first transfer", queue[0]);
        handleTransferSelect({ detail: queue[0] });
      }
    });

    EventsOn("file:updated", (file: any) => {
      console.log(
        "App: file:updated event received",
        file.sourcePath,
        file.status,
      );
      // Update file in our local list if it matches
      if (activeFiles) {
        const idx = activeFiles.findIndex(
          (f) => f.sourcePath === file.sourcePath,
        );
        if (idx !== -1) {
          console.log("App: Updating file in activeFiles at index", idx);
          activeFiles[idx] = file;
          // Trigger reactivity
          activeFiles = [...activeFiles];
        } else {
          console.warn(
            "App: Received update for file not in activeFiles",
            file.sourcePath,
          );
        }
      }

      if (file.status === "in_progress") {
        currentFile = file;
      } else if (
        currentFile &&
        currentFile.sourcePath === file.sourcePath &&
        file.status !== "in_progress"
      ) {
        currentFile = null; // Clear if it finished
      }
    });

    EventsOn("transfer:progress", (progress: any) => {
      // Update speed/eta if we have an active transfer
      if (activeTransfer && activeTransfer.status === "in_progress") {
        currentSpeed = progress.speed || 0;
        if (currentSpeed > 0) {
          estimatedTime =
            (activeTransfer.totalBytes - activeTransfer.bytesCopied) /
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
    activeFiles = activeTransfer ? activeTransfer.files || [] : [];
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
              (f) => f.sourcePath === currentFile.sourcePath,
            ) + 1
          : 0}
        totalFiles={activeFiles.length}
      />
      <OverallProgress transfer={activeTransfer} {currentSpeed} />
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
