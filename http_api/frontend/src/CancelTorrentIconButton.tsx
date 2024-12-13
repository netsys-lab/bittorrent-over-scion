import {IconButton} from "@mui/material";
import StopIcon from '@mui/icons-material/Stop';
import ApiConfig from "./ApiConfig.tsx";
import {useSnackbar} from "notistack";

interface CancelTorrentIconButtonProps {
  apiConfig: ApiConfig,
  torrentId: number
}

export default function CancelTorrentIconButton({apiConfig, torrentId} : CancelTorrentIconButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const handleClick = async () => {
    try {
      const formData = new FormData();
      formData.append("action", "cancel");

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
          "Cancelling torrent failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Successfully cancelled torrent!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        "Cancelling torrent failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log("Cancelling torrent with id " + torrentId + " failed:", error);
    }
  };

  return (
    <IconButton
      edge="end"
      onClick={() => handleClick()}
    >
      <StopIcon />
    </IconButton>
  );
}
