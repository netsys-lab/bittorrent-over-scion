import {useState} from 'react';
import {IconButton, Menu, MenuItem} from "@mui/material";
import DeleteIcon from '@mui/icons-material/Delete';
import ApiConfig from "./ApiConfig.tsx";
import {useSnackbar} from "notistack";

interface DeleteTorrentIconButtonProps {
  apiConfig: ApiConfig,
  torrentId: number
}

export default function DeleteTorrentIconButton({apiConfig, torrentId} : DeleteTorrentIconButtonProps) {
  const { enqueueSnackbar} = useSnackbar();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleClick = async (keepFilesOnDisk: boolean) => {
    try {
      const response = await fetch(
        apiConfig.torrentEndpoint(torrentId) + (!keepFilesOnDisk ? "?deleteFiles=1" : ""),
        {
          method: "DELETE",
        }
      );
      const body = await response.json();

      if (!response.ok) {
        enqueueSnackbar(
          "Deleting torrent failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Successfully deleted torrent!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        "Deleting torrent failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log("Deleting torrent with id " + torrentId + " failed:", error);
    }

    setAnchorEl(null);
  };

  return (
    <>
      <IconButton
        edge="end"
        onClick={(e) => { setAnchorEl(e.currentTarget) }}
      >
        <DeleteIcon />
      </IconButton>
      <Menu
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        MenuListProps={{
          dense: true
        }}
      >
        <MenuItem onClick={() => handleClick(true)}>
          Keep Downloaded/Uploaded Files on Disk
        </MenuItem>
        <MenuItem onClick={() => handleClick(false)}>
          Delete Everything
        </MenuItem>
      </Menu>
    </>
  );
}
