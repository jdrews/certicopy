/**
 * State actions for mutating the state in a controlled way
 */

import type { FileInfo, TransferJob, TransferMetrics } from './types';
import { appState } from './state.svelte';

/**
 * Add a new transfer to the queue
 */
export function addTransfer(transfer: TransferJob) {
  appState.transfers.push(transfer);
  if (!appState.activeTransferId) {
    appState.activeTransferId = transfer.id;
  }
}

/**
 * Update a file in the transfer queue
 */
export function updateFile(file: FileInfo) {
  const transfer = appState.transfers.find(t =>
    t.files.some(f => f.sourcePath === file.sourcePath)
  );
  if (transfer) {
    const idx = transfer.files.findIndex(f => f.sourcePath === file.sourcePath);
    if (idx !== -1) {
      transfer.files[idx] = file;
    }
  }
}

/**
 * Update appState.metrics for a specific transfer
 */
export function updateMetrics(transferId: string, updates: Partial<TransferMetrics>) {
  const current = appState.metrics.get(transferId);
  appState.metrics.set(transferId, {
    dataPoints: current?.dataPoints || [],
    maxSpeed: current?.maxSpeed || 0,
    lastSpeed: current?.lastSpeed || 0,
    currentFile: current?.currentFile || null,
    ...updates
  });
}

/**
 * Set appState.metrics for a specific transfer (fully replaces)
 */
export function setTransferMetrics(transferId: string, updates: Partial<TransferMetrics>) {
  const current = appState.metrics.get(transferId) || {
    dataPoints: [],
    maxSpeed: 0,
    lastSpeed: 0,
    currentFile: null
  };
  appState.metrics.set(transferId, {
    dataPoints: current.dataPoints,
    maxSpeed: current.maxSpeed,
    lastSpeed: current.lastSpeed,
    currentFile: current.currentFile,
    ...updates
  });
}

/**
 * Set the active transfer by ID
 */
export function setActiveTransfer(id: string | null) {
  appState.activeTransferId = id;
}

/**
 * Update graph data for a transfer
 * Handles resets, new data points, and speed updates
 */
export function updateGraphData(transferId: string, bytesCopied: number, speed: number) {
  const current = appState.metrics.get(transferId) || {
    dataPoints: [],
    maxSpeed: 0,
    lastSpeed: 0,
    currentFile: null
  };

  const lastPoint = current.dataPoints.at(-1);

  // Handle potential reset (e.g. retry or new job with same ID)
  if (lastPoint && bytesCopied < lastPoint.bytesCopied - 1024 * 1024) {
    // 1MB threshold for "significant reset"
    appState.metrics.set(transferId, {
      dataPoints: [{ bytesCopied, speed }],
      maxSpeed: speed,
      lastSpeed: speed,
      currentFile: current.currentFile
    });
    return;
  }

  // Only add if we have new forward progress
  if (current.dataPoints.length === 0 || bytesCopied > (lastPoint?.bytesCopied || 0)) {
    appState.metrics.set(transferId, {
      dataPoints: [...current.dataPoints, { bytesCopied, speed }],
      maxSpeed: Math.max(current.maxSpeed, speed),
      lastSpeed: speed,
      currentFile: current.currentFile
    });
  } else {
    // Just update current speed if bytes haven't changed
    appState.metrics.set(transferId, {
      ...current,
      lastSpeed: speed
    });
  }
}