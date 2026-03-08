export interface MemoryEntry {
  id: string;
  content: string;
  tags?: string[];
  source?: string;
  createdAt: number;
}

export interface CoreMemoryEntry {
  key: string;
  value: string;
  updatedAt: number;
}
