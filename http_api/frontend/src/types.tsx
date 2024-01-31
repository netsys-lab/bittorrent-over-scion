export enum ApiTorrentState {
  NotStartedYet = 'not_started_yet',
  Running = 'running',
  Completed = 'completed',
  Failed = 'failed',
  Cancelled = 'cancelled',
  Seeding = 'seeding',
}

export const NonSeedingTorrentStates : Array<ApiTorrentState> = [ApiTorrentState.NotStartedYet, ApiTorrentState.Running, ApiTorrentState.Completed, ApiTorrentState.Failed, ApiTorrentState.Cancelled]

export interface ApiFile {
  id: number;
  path: string;
  length: number;
  progress: number;
}

export interface ApiTorrentMetrics {
  rx: number;
  tx: number;
  numConns: number;
  numPaths: number;
}

export interface ApiTorrent {
  id: number;
  name: string;
  state: ApiTorrentState;
  status: string;
  peer: string;
  seedOnCompletion: boolean;
  seedPort: number;
  seedAddr: string;
  files: Array<ApiFile>;
  metrics: ApiTorrentMetrics;
  numPieces: number;
  numDownloadedPieces: number;
  pieceLength: number;
}

export interface ApiTorrents {
  [key: number]: ApiTorrent;
}

export interface ApiTracker {
  id: number;
  url: string;
}

export interface ApiTrackers {
  [key: number]: ApiTracker;
}