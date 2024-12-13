import {IconButton} from "@mui/material";
import DownloadIcon from '@mui/icons-material/Download';
import ApiConfig from "./ApiConfig.tsx";

interface DownloadTorrentIconButtonProps {
  apiConfig: ApiConfig,
  torrentId: number,
  fileId: number
}

export default function DownloadTorrentIconButton({apiConfig, torrentId, fileId} : DownloadTorrentIconButtonProps) {
  const handleClick = async () => {
    window.open(apiConfig.fileEndpoint(torrentId, fileId), "_blank");
  };

  return (
      <IconButton
        edge="end"
        onClick={(_) => handleClick()}
      >
        <DownloadIcon />
      </IconButton>
  );
}
