import type { RoomBootstrap } from './bootstrap';
export interface EntryProps {
  connection?: RoomBootstrap | null;
  snapshot?: unknown;
  onwronggame?: (code: string) => void;
  onstate?: (code: string) => void;
}
