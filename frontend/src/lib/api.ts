export const API_BASE = "http://localhost:8090/api";

export interface Notebook {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  fileCount: number;
}

export interface Source {
  fileName: string;
  fileSize: number;
  chunkCount: number;
  ingestedAt: string;
  status: "processing" | "ingested" | "failed";
  error?: string;
}

export interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  citations?: { fileName: string; pageNumber: number; snippet: string; index: number }[];
}

export interface NotebookDetail extends Notebook {
  sources: Source[];
  messages?: Message[];
}

export interface IngestProgress {
  totalFiles: number;
  processedFiles: number;
  currentFile: string;
  status: string;
  error?: string;
}

export async function fetchNotebooks(signal?: AbortSignal): Promise<Notebook[]> {
  const res = await fetch(`${API_BASE}/notebooks`, { signal });
  if (!res.ok) throw new Error("Failed to fetch notebooks");
  return res.json();
}

export async function createNotebook(name: string, description: string, signal?: AbortSignal): Promise<Notebook> {
  const res = await fetch(`${API_BASE}/notebooks`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
    signal,
  });
  if (!res.ok) throw new Error("Failed to create notebook");
  return res.json();
}

export async function getNotebook(id: string): Promise<NotebookDetail> {
  const res = await fetch(`${API_BASE}/notebooks/${id}`);
  if (!res.ok) throw new Error("Failed to fetch notebook details");
  return res.json();
}

export async function deleteNotebook(id: string, signal?: AbortSignal): Promise<void> {
  const res = await fetch(`${API_BASE}/notebooks/${id}`, { method: "DELETE", signal });
  if (!res.ok) throw new Error("Failed to delete notebook");
}

export async function uploadFiles(notebookId: string, files: File[]): Promise<{ message: string; totalFiles: number }> {
  const formData = new FormData();
  for (const file of files) {
    formData.append("files", file);
  }

  const res = await fetch(`${API_BASE}/notebooks/${notebookId}/upload`, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Upload failed" }));
    throw new Error(error.error || "Upload failed");
  }

  return res.json();
}
