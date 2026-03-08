export interface SessionMessage {
  role: "user" | "assistant" | "system";
  content: string;
  timestamp: number;
}

export interface Session {
  id: string;
  channel: string;
  accountId: string;
  peer: string;
  messages: SessionMessage[];
  createdAt: number;
  updatedAt: number;
}
