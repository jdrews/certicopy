/**
 * Type definitions for certicopy frontend
 */

export type TransferStatus = 'pending' | 'in_progress' | 'success' | 'failed' | 'paused';
export type FileStatus = TransferStatus | 'skipped';

export interface FileInfo {
  sourcePath: string;
  destPath: string;
  name: string;
  size: number;
  bytesCopied: number;
  status: FileStatus;
  sourceHash?: string;
  destHash?: string;
  errorMessage?: string;
}

export interface TransferJob {
  id: string;
  sources: string[];
  destination: string;
  status: TransferStatus;
  totalFiles: number;
  totalBytes: number;
  bytesCopied: number;
  files: FileInfo[];
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}

export interface DataPoint {
  bytesCopied: number;
  speed: number;
}

export interface TransferMetrics {
  dataPoints: DataPoint[];
  maxSpeed: number;
  lastSpeed: number;
  currentFile: FileInfo | null;
}