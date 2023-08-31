import {useState, ChangeEvent} from 'react';
import {Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField} from '@mui/material';
import { MuiFileInput } from 'mui-file-input';
import { useSnackbar } from 'notistack';
import ApiConfig from "./ApiConfig.tsx";

interface AddTorrentButtonProps {
  apiConfig: ApiConfig
}

export default function AddTorrentButton({apiConfig} : AddTorrentButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [peer, setPeer] = useState("");
  const [error, setError] = useState<string | null>(null);

  const clearFields = () => {
    setFile(null);
    setPeer("");
    setError(null);
  };

  const handleClickAdd = async () => {
    if (file == null) {
      setError("Torrent file needs to be selected!");
      return;
    }
    if (peer.length == 0) {
      setError("Peer field needs to be filled out!");
      return;
    }

    const formData = new FormData();
    formData.append("peer", peer);
    formData.append("torrentFile", file!!);

    try {
      const response = await fetch(apiConfig.torrentEndpoint(), {
        method: "POST",
        body: formData,
      });
      const body = await response.json();

      if (!response.ok) {
        setError(body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!");
        return;
      }

      enqueueSnackbar("Successfully added torrent!", {variant: "success"});
      setOpen(false);
      clearFields();
    } catch (error) {
      setError("Connection error! API offline? (" + error + ")");
    }
  };

  return (
    <div>
      <Button variant="contained" onClick={() => setOpen(true)}>
        Add Torrent
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>Add Torrent</DialogTitle>
        <DialogContent>
          {/*<DialogContentText>
            To subscribe to this website, please enter your email address here. We
            will send updates occasionally.
          </DialogContentText>*/}
          {error != null && <Alert hidden severity="error">{error}</Alert>}
          <MuiFileInput
              label="Torrent File"
              value={file}
              margin="normal"
              onChange={(newFile: File | null) => setFile(newFile)}
              fullWidth
              required
          />
          <TextField
              id="name"
              label="Peer"
              type="text"
              placeholder="19-ffaa:1:106d,[127.0.0.1]:43000"
              margin="normal"
              InputLabelProps={{
                shrink: true
              }}
              value={peer}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setPeer(event.target.value);
              }}
              fullWidth
              required
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={clearFields}>Clear</Button>
          <Button onClick={handleClickAdd}>Add</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}