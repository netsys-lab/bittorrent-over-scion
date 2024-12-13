import {useState, ChangeEvent} from 'react';
import {
  Alert,
  Button, Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle, Divider,
  FormControlLabel, Switch,
  TextField, Tooltip
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
  const [peers, setPeers] = useState(new Array<string>());
  const [seedOnCompletion, setSeedOnCompletion] = useState(false);
  const [seedPort, setSeedPort] = useState<number | null>(null);
  const [enableDht, setEnableDht] = useState(false);
  const [enableTrackers, setEnableTrackers] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearFields = () => {
    setFile(null);
    setPeers(new Array<string>());
    setSeedOnCompletion(false);
    setSeedPort(null);
    setEnableDht(false);
    setEnableTrackers(false);
    setError(null);
  };

  const handleClickStart = async () => {
    if (file == null) {
      setError("Torrent file needs to be selected!");
      return;
    }
    if (peers.length == 0) {
      setError("Peer field needs to be filled out!");
      return;
    }

    const formData = new FormData();
    for (let i = 0; i < peers.length; ++i) {
      formData.append(`peers`, peers[i]);
    }
    formData.append("seedOnCompletion", seedOnCompletion ? "1" : "0");
    formData.append("seedPort", seedPort != null ? seedPort.toString() : "0");
    formData.append("enableDht", enableDht ? "1" : "0");
    formData.append("enableTrackers", enableTrackers ? "1" : "0");
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
          <FormControlLabel control={
            <Checkbox
              checked={seedOnCompletion}
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
          <Divider textAlign="left">Peer Discovery</Divider>
          <FormControlLabel control={
            <Switch
              checked={enableDht}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setEnableDht(event.currentTarget.checked);
              }}
            />
          } label="Use DHT for peer discovery" />
          <FormControlLabel control={
            <Switch
              checked={enableTrackers}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setEnableTrackers(event.currentTarget.checked);
              }}
            />
          } label="Use trackers for peer discovery" />
          <Tooltip title="Addresses of peers that will be used in addition to DHT and trackers">
            <TextField
              label="Additional peer addresses"
              type="text"
              placeholder="19-ffaa:1:106d,[127.0.0.1]:43000&#10;17-ffaa:0:cafd,[127.0.0.1]:43000"
              margin="normal"
              InputLabelProps={{
                shrink: true
              }}
              value={peers.join("\n")}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setPeers(event.target.value.split("\n"));
              }}
              multiline
              rows={4}
              fullWidth
            />
          </Tooltip>
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