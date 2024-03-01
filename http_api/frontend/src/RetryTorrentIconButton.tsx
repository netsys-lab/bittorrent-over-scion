import {IconButton} from "@mui/material";
import ReplayIcon from '@mui/icons-material/Replay';
import ApiConfig from "./ApiConfig.tsx";
import {useSnackbar} from "notistack";

interface RetryTorrentIconButtonProps {
  apiConfig: ApiConfig,
  torrentId: number
}

export default function RetryTorrentIconButton({apiConfig, torrentId} : RetryTorrentIconButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const handleClick = async () => {
    try {
      const formData = new FormData();
      formData.append("action", "retry");

      const response = await fetch(
        apiConfig.torrentEndpoint(torrentId),
        {
          method: "POST",
          body: formData,
        }
      );
      const body = await response.json();

      if (!response.ok) {
        enqueueSnackbar(
          "Retrying torrent failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Successfully enqueued torrent again!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        "Retrying torrent failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log("Retrying torrent with id " + torrentId + " failed:", error);
    }
  };

  return (
    <IconButton
      edge="end"
      onClick={() => handleClick()}
    >
      <ReplayIcon />
    </IconButton>
  );
}
