/**
 * Core state management using Svelte 5 runes
 * 
 * State is automatically reactive with $state()
 * Derived values use $derived() for computed properties
 */

import type { FileInfo, TransferJob, TransferMetrics } from './types';
import { EventsOn } from "../../wailsjs/runtime/runtime";

/**
 * App State Management using Svelte 5 Runes
 * 
 * Using a singleton class with $state and getters
 * is the recommended way to share global state in Svelte 5.
 */
class AppState {
  transfers = $state<TransferJob[]>([]);
  activeTransferId = $state<string | null>(null);
  metrics = $state<Map<string, TransferMetrics>>(new Map());

  // Getters act as $derived values in Svelte 5 classes
  get activeTransfer() {
    return this.transfers.find(t => t.id === this.activeTransferId) || null;
  }

  get activeFiles() {
    return this.activeTransfer?.files || [];
  }

  get viewMetrics() {
    return this.activeTransferId ? this.metrics.get(this.activeTransferId) || null : null;
  }

  get currentFile() {
    return this.viewMetrics?.currentFile || null;
  }

  get lastSpeed() {
    return this.viewMetrics?.lastSpeed || 0;
  }

  // --- State Mutations ---

  setTransfers(queue: TransferJob[]) {
    this.transfers = queue;
  }

  setActiveTransfer(id: string | null) {
    this.activeTransferId = id;
  }

  addTransfer(transfer: TransferJob) {
    this.transfers.push(transfer);
    if (!this.activeTransferId) {
      this.activeTransferId = transfer.id;
    }
  }

  updateFile(file: FileInfo) {
    const transfer = this.transfers.find(t =>
      t.files.some(f => f.sourcePath === file.sourcePath)
    );
    if (transfer) {
      const idx = transfer.files.findIndex(f => f.sourcePath === file.sourcePath);
      if (idx !== -1) {
        transfer.files[idx] = file;
        // Trigger reactivity for anyone watching the files array
        transfer.files = [...transfer.files];
      }
    }
  }

  updateMetrics(transferId: string, updates: Partial<TransferMetrics>) {
    const current = this.metrics.get(transferId) || {
      dataPoints: [],
      maxSpeed: 0,
      lastSpeed: 0,
      currentFile: null
    };
    this.metrics.set(transferId, { ...current, ...updates });
  }

  updateGraphData(transferId: string, bytesCopied: number, speed: number) {
    const current = this.metrics.get(transferId) || {
      dataPoints: [],
      maxSpeed: 0,
      lastSpeed: 0,
      currentFile: null
    };

    const lastPoint = current.dataPoints.at(-1);

    if (lastPoint && bytesCopied < lastPoint.bytesCopied - 1024 * 1024) {
      this.metrics.set(transferId, {
        dataPoints: [{ bytesCopied, speed }],
        maxSpeed: speed,
        lastSpeed: speed,
        currentFile: current.currentFile
      });
      return;
    }

    if (current.dataPoints.length === 0 || bytesCopied > (lastPoint?.bytesCopied || 0)) {
      this.metrics.set(transferId, {
        dataPoints: [...current.dataPoints, { bytesCopied, speed }],
        maxSpeed: Math.max(current.maxSpeed, speed),
        lastSpeed: speed,
        currentFile: current.currentFile
      });
    } else {
      this.metrics.set(transferId, {
        ...current,
        lastSpeed: speed
      });
    }
  }

  /**
   * Initialize global wails listeners.
   * Called in App.svelte onMount to ensure context.
   */
  initEventListeners() {
    EventsOn("queue:updated", (queue: TransferJob[]) => {
      this.setTransfers(queue);
    });

    EventsOn("file:updated", (file: any) => {
      // Critical: update the file in the transfers array so FileList sees it
      this.updateFile(file);

      const job = this.transfers.find((j) =>
        j.files?.some((f) => f.sourcePath === file.sourcePath),
      );
      if (job) {
        if (file.status === "in_progress") {
          this.updateMetrics(job.id, { currentFile: file });
        } else {
          const m = this.metrics.get(job.id);
          if (m?.currentFile?.sourcePath === file.sourcePath) {
            this.updateMetrics(job.id, { currentFile: null });
          }
        }
      }
    });

    EventsOn("transfer:progress", (progress: any) => {
      const jobId = progress.jobId || this.transfers.find((j) => j.status === "in_progress")?.id;
      if (jobId) {
        this.updateGraphData(jobId, progress.bytesCopied || 0, progress.speed || 0);
      }
    });
  }
}

export const appState = new AppState();