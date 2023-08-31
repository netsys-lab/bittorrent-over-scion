import {OverridableStringUnion} from "@mui/types";

export interface ApiFile {
  id: number;
  path: string;
  length: number;
  progress: number;
}

export interface ApiMetrics {
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
  metrics: ApiMetrics;
  numPieces: number;
  numDownloadedPieces: number;
  pieceLength: number;
}

export interface ApiTorrents {
  [key: number]: ApiTorrent;
}