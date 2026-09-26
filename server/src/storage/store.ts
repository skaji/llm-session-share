export interface Store {
  get(key: string): Promise<string | null>;
  put(key: string, body: string): Promise<void>;
  delete(key: string): Promise<void>;
}

export const validID = (id: string) =>
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id);

export function sessionKey(id: string): string {
  if (!validID(id)) throw new Error('Invalid session ID');
  return `llm-session-share/${id}.json`;
}
