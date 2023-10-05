import {OverridableStringUnion} from "@mui/types";

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
  state: OverridableStringUnion<'not started yet' | 'running' | 'completed' | 'failed' | 'cancelled'>;
  status: string;
  peer: string;
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