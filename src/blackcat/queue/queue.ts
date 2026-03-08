import { randomUUID } from "node:crypto";

import type { QueueConfig, QueueStats, Task } from "./types.js";
import { TaskPriority, TaskStatus } from "./types.js";

function now(): number {
  return Date.now();
}

export class TaskQueue {
  private readonly pending: Task[] = [];
  private readonly processing: Map<string, Task> = new Map();
  private readonly completed: Task[] = [];
  private readonly failed: Task[] = [];
  private readonly config: Required<QueueConfig>;

  constructor(config: QueueConfig = {}) {
    this.config = {
      maxSize: config.maxSize ?? 1000,
      defaultPriority: config.defaultPriority ?? TaskPriority.NORMAL,
      concurrency: config.concurrency ?? 1,
    };
  }

  enqueue(payload: unknown, priority?: TaskPriority): string {
    if (this.pending.length >= this.config.maxSize) {
      throw new Error(`queue is full (max ${this.config.maxSize})`);
    }

    const task: Task = {
      id: randomUUID(),
      priority: priority ?? this.config.defaultPriority,
      status: TaskStatus.PENDING,
      payload,
      createdAt: now(),
    };

    this.pending.push(task);
    this.sortPending();

    return task.id;
  }

  dequeue(): Task | undefined {
    const task = this.pending.shift();
    if (!task) {
      return undefined;
    }

    task.status = TaskStatus.PROCESSING;
    task.startedAt = now();
    this.processing.set(task.id, task);

    return task;
  }

  async process(
    handler: (task: Task) => Promise<void>,
    concurrency?: number,
  ): Promise<void> {
    const limit = concurrency ?? this.config.concurrency;

    const worker = async (): Promise<void> => {
      while (this.pending.length > 0) {
        const task = this.dequeue();
        if (!task) {
          break;
        }

        try {
          await handler(task);
          task.status = TaskStatus.COMPLETED;
          task.completedAt = now();
          this.processing.delete(task.id);
          this.completed.push(task);
        } catch (err) {
          task.status = TaskStatus.FAILED;
          task.completedAt = now();
          task.error = err instanceof Error ? err.message : String(err);
          this.processing.delete(task.id);
          this.failed.push(task);
        }
      }
    };

    const workers: Promise<void>[] = [];
    for (let i = 0; i < limit; i++) {
      workers.push(worker());
    }

    await Promise.all(workers);
  }

  async drain(): Promise<void> {
    // Wait until processing map is empty
    while (this.processing.size > 0) {
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }

  getStats(): QueueStats {
    return {
      pending: this.pending.length,
      processing: this.processing.size,
      completed: this.completed.length,
      failed: this.failed.length,
    };
  }

  getTask(id: string): Task | undefined {
    const inPending = this.pending.find((t) => t.id === id);
    if (inPending) return inPending;

    const inProcessing = this.processing.get(id);
    if (inProcessing) return inProcessing;

    const inCompleted = this.completed.find((t) => t.id === id);
    if (inCompleted) return inCompleted;

    return this.failed.find((t) => t.id === id);
  }

  clear(): void {
    this.pending.length = 0;
    this.processing.clear();
    this.completed.length = 0;
    this.failed.length = 0;
  }

  private sortPending(): void {
    // Higher priority number = processed first (descending), then FIFO by createdAt
    this.pending.sort((a, b) => {
      if (a.priority !== b.priority) {
        return b.priority - a.priority;
      }
      return a.createdAt - b.createdAt;
    });
  }
}
