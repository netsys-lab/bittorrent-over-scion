import {useState, ChangeEvent} from 'react';
import {
  Alert,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  TextField
} from '@mui/material';
import { MuiFileInput } from 'mui-file-input';
import { useSnackbar } from 'notistack';
import ApiConfig from "./ApiConfig.tsx";

interface DownloadTorrentButtonProps {
  apiConfig: ApiConfig
}

export default function DownloadTorrentButton({apiConfig} : DownloadTorrentButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [peer, setPeer] = useState("");
  const [seedOnCompletion, setSeedOnCompletion] = useState(false);
  const [seedPort, setSeedPort] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const clearFields = () => {
    setFile(null);
    setPeer("");
    setSeedOnCompletion(false);
    setSeedPort(null);
    setError(null);
  };

  const handleClickStart = async () => {
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
    formData.append("seedOnCompletion", seedOnCompletion ? "1" : "0");
    formData.append("seedPort", seedPort != null ? seedPort.toString() : "0")
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
        Download Remote Torrent
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>Download Remote Torrent</DialogTitle>
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
            label="Remote Peer"
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
          <FormControlLabel control={
            <Checkbox
              value={seedOnCompletion}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
               setSeedOnCompletion(event.currentTarget.checked);
              }}
            />
          } label="Seed after successful download" />
          {seedOnCompletion && <TextField
            label="Port for Seeding"
            type="number"
            helperText="A random port will be chosen if not specified here."
            placeholder="0-65535"
            margin="normal"
            InputLabelProps={{
              shrink: true
            }}
            value={seedPort}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              if (event.target.value.length == 0) {
                setSeedPort(null);
              } else {
                setSeedPort(parseInt(event.target.value));
              }
            }}
            fullWidth />}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={clearFields}>Clear</Button>
          <Button onClick={handleClickStart}>Start</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}