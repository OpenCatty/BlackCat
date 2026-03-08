import { describe, it, expect, beforeEach } from "vitest";

import { TaskQueue } from "./queue.js";
import { TaskPriority, TaskStatus } from "./types.js";

describe("TaskQueue", () => {
  let queue: TaskQueue;

  beforeEach(() => {
    queue = new TaskQueue();
  });

  it("enqueue returns a task ID", () => {
    const id = queue.enqueue({ message: "hello" });
    expect(typeof id).toBe("string");
    expect(id.length).toBeGreaterThan(0);
  });

  it("enqueue increments pending count in stats", () => {
    queue.enqueue("task-1");
    queue.enqueue("task-2");
    queue.enqueue("task-3");

    const stats = queue.getStats();
    expect(stats.pending).toBe(3);
    expect(stats.processing).toBe(0);
    expect(stats.completed).toBe(0);
    expect(stats.failed).toBe(0);
  });

  it("dequeue returns highest priority task first", () => {
    queue.enqueue("low-task", TaskPriority.LOW);
    queue.enqueue("high-task", TaskPriority.HIGH);
    queue.enqueue("normal-task", TaskPriority.NORMAL);

    const first = queue.dequeue();
    expect(first?.payload).toBe("high-task");
    expect(first?.status).toBe(TaskStatus.PROCESSING);

    const second = queue.dequeue();
    expect(second?.payload).toBe("normal-task");

    const third = queue.dequeue();
    expect(third?.payload).toBe("low-task");
  });

  it("dequeue returns undefined when queue is empty", () => {
    const task = queue.dequeue();
    expect(task).toBeUndefined();
  });

  it("FIFO ordering within same priority", () => {
    queue.enqueue("first", TaskPriority.NORMAL);
    queue.enqueue("second", TaskPriority.NORMAL);
    queue.enqueue("third", TaskPriority.NORMAL);

    expect(queue.dequeue()?.payload).toBe("first");
    expect(queue.dequeue()?.payload).toBe("second");
    expect(queue.dequeue()?.payload).toBe("third");
  });

  it("process executes all pending tasks", async () => {
    const processed: unknown[] = [];

    queue.enqueue("a");
    queue.enqueue("b");
    queue.enqueue("c");

    await queue.process(async (task) => {
      processed.push(task.payload);
    });

    expect(processed).toEqual(["a", "b", "c"]);

    const stats = queue.getStats();
    expect(stats.pending).toBe(0);
    expect(stats.processing).toBe(0);
    expect(stats.completed).toBe(3);
  });

  it("process records failed tasks", async () => {
    queue.enqueue("will-fail");
    queue.enqueue("will-succeed");

    await queue.process(async (task) => {
      if (task.payload === "will-fail") {
        throw new Error("boom");
      }
    });

    const stats = queue.getStats();
    expect(stats.failed).toBe(1);
    expect(stats.completed).toBe(1);
  });

  it("process respects concurrency limit", async () => {
    let maxConcurrent = 0;
    let currentConcurrent = 0;

    for (let i = 0; i < 6; i++) {
      queue.enqueue(`task-${i}`);
    }

    await queue.process(async () => {
      currentConcurrent++;
      if (currentConcurrent > maxConcurrent) {
        maxConcurrent = currentConcurrent;
      }
      await new Promise((resolve) => setTimeout(resolve, 20));
      currentConcurrent--;
    }, 2);

    // With concurrency=2, we should see up to 2 concurrent
    expect(maxConcurrent).toBeLessThanOrEqual(2);
    expect(queue.getStats().completed).toBe(6);
  });

  it("getTask retrieves task by ID across all states", async () => {
    const id = queue.enqueue("findme");

    // In pending state
    const pending = queue.getTask(id);
    expect(pending?.payload).toBe("findme");
    expect(pending?.status).toBe(TaskStatus.PENDING);

    // Process to completion
    await queue.process(async () => {});

    const completed = queue.getTask(id);
    expect(completed?.status).toBe(TaskStatus.COMPLETED);
  });

  it("enforces maxSize limit", () => {
    const smallQueue = new TaskQueue({ maxSize: 2 });
    smallQueue.enqueue("a");
    smallQueue.enqueue("b");

    expect(() => smallQueue.enqueue("c")).toThrow("queue is full");
  });

  it("clear resets all state", () => {
    queue.enqueue("a");
    queue.enqueue("b");
    queue.dequeue(); // move one to processing

    queue.clear();

    const stats = queue.getStats();
    expect(stats.pending).toBe(0);
    expect(stats.processing).toBe(0);
    expect(stats.completed).toBe(0);
    expect(stats.failed).toBe(0);
  });

  it("uses default priority from config", () => {
    const highQueue = new TaskQueue({ defaultPriority: TaskPriority.HIGH });
    highQueue.enqueue("task-a");

    const task = highQueue.dequeue();
    expect(task?.priority).toBe(TaskPriority.HIGH);
  });

  it("failed tasks store error message", async () => {
    const id = queue.enqueue("err-task");

    await queue.process(async () => {
      throw new Error("something went wrong");
    });

    const task = queue.getTask(id);
    expect(task?.status).toBe(TaskStatus.FAILED);
    expect(task?.error).toBe("something went wrong");
  });
});
