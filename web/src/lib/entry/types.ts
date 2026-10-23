import type { RoomBootstrap } from './bootstrap';
export interface EntryProps {
  connection?: RoomBootstrap | null;
  snapshot?: unknown;
  /** The room and the device link's claim the connection joined with. */
  code?: string;
  claim?: string | null;
  onwronggame?: (code: string) => void;
  onstate?: (code: string) => void;
}
