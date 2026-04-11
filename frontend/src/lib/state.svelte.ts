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
 */
class AppState {
  transfers = $state<TransferJob[]>([]);
  activeTransferId = $state<string | null>(null);
  metrics = $state<Record<string, TransferMetrics>>({});

  // Getters act as $derived values in Svelte 5 classes
  get activeTransfer() {
    if (!this.activeTransferId || this.transfers.length === 0) return null;
    return this.transfers.find(t => t.id === this.activeTransferId) || null;
  }

  get activeFiles() {
    const transfer = this.activeTransfer;
    if (!transfer) return [];

    // Defensive check for 'files' property existence and naming (Wails serialization)
    return transfer.files ?? (transfer as any).Files ?? [];
  }

  get viewMetrics() {
    return this.activeTransferId ? this.metrics[this.activeTransferId] ?? null : null;
  }

  get currentFile() {
    return this.viewMetrics?.currentFile ?? null;
  }

  get lastSpeed() {
    return this.viewMetrics?.lastSpeed ?? 0;
  }

  // --- State Mutations ---

  setTransfers(queue: TransferJob[]) {
    this.transfers = queue ?? [];

    // Auto-select first job if none selected
    if (!this.activeTransferId && this.transfers.length > 0) {
      this.activeTransferId = this.transfers[0].id;
    }
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
    if (!file?.sourcePath || !file?.jobId) return;

    // Use jobId for unambiguous transfer lookup
    const transferId = file.jobId;
    const transfer = this.transfers.find(t => t.id === transferId);

    if (transfer) {
      const files = transfer.files ?? (transfer as any).Files;
      if (!files) return;

      const idx = files.findIndex((f: any) => f.sourcePath === file.sourcePath);
      if (idx !== -1) {
        files[idx] = file;

        // Trigger Svelte's proxy reactivity for array modifications
        if (transfer.files) transfer.files = [...transfer.files];
        if ((transfer as any).Files) (transfer as any).Files = [...(transfer as any).Files];

        // Ensure the currentFile in metrics is updated if it matches
        const m = this.metrics[transfer.id];
        if (m?.currentFile?.sourcePath === file.sourcePath) {
          m.currentFile = file;
        }
      }
    }
  }

  updateMetrics(transferId: string, updates: Partial<TransferMetrics>) {
    const current = this.metrics[transferId] ?? {
      dataPoints: [],
      maxSpeed: 0,
      lastSpeed: 0,
      currentFile: null
    };
    this.metrics[transferId] = { ...current, ...updates };
  }

  updateGraphData(transferId: string, bytesCopied: number, speed: number) {
    const current = this.metrics[transferId] ?? {
      dataPoints: [],
      maxSpeed: 0,
      lastSpeed: 0,
      currentFile: null
    };

    const lastPoint = current.dataPoints.at(-1);

    // If speed is significantly lower than before or bytesCopied resets, start a new point
    if (lastPoint && bytesCopied < lastPoint.bytesCopied - 1024 * 1024) {
      this.metrics[transferId] = {
        dataPoints: [{ bytesCopied, speed }],
        maxSpeed: speed,
        lastSpeed: speed,
        currentFile: current.currentFile
      };
      return;
    }

    if (current.dataPoints.length === 0 || bytesCopied > (lastPoint?.bytesCopied ?? 0)) {
      this.metrics[transferId] = {
        dataPoints: [...current.dataPoints, { bytesCopied, speed }],
        maxSpeed: Math.max(current.maxSpeed, speed),
        lastSpeed: speed,
        currentFile: current.currentFile
      };
    } else {
      this.metrics[transferId] = {
        ...current,
        lastSpeed: speed
      };
    }
  }

  initEventListeners() {
    EventsOn("queue:updated", (queue: any[]) => {
      console.log("[AppState] Queue updated:", queue?.length || 0, "jobs");
      this.setTransfers(queue);
    });

    EventsOn("file:updated", (file: any) => {
      this.updateFile(file);

      const jobId = file.jobId;
      if (jobId) {
        if (file.status === "in_progress") {
          this.updateMetrics(jobId, { currentFile: file });
        } else {
          const m = this.metrics[jobId];
          if (m?.currentFile?.sourcePath === file.sourcePath) {
            this.updateMetrics(jobId, { currentFile: null });
          }
        }
      }
    });

    EventsOn("transfer:progress", (progress: any) => {
      const jobId = progress.jobId || this.transfers.find((j) => j.status === "in_progress")?.id;
      if (jobId) {
        this.updateGraphData(jobId, progress.bytesCopied ?? 0, progress.speed ?? 0);
      }
    });
  }
}

export const appState = new AppState();