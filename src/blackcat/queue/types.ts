export enum TaskPriority {
  LOW = 1,
  NORMAL = 2,
  HIGH = 3,
}

export enum TaskStatus {
  PENDING = "pending",
  PROCESSING = "processing",
  COMPLETED = "completed",
  FAILED = "failed",
}

export interface Task {
  id: string;
  priority: TaskPriority;
  status: TaskStatus;
  payload: unknown;
  createdAt: number;
  startedAt?: number;
  completedAt?: number;
  error?: string;
}

export interface QueueConfig {
  maxSize?: number;
  defaultPriority?: TaskPriority;
  concurrency?: number;
}

export interface QueueStats {
  pending: number;
  processing: number;
  completed: number;
  failed: number;
}
