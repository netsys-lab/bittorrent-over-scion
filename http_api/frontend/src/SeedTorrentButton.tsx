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

interface SeedTorrentButtonProps {
  apiConfig: ApiConfig
}

export default function SeedTorrentButton({apiConfig} : SeedTorrentButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const [open, setOpen] = useState(false);
  const [torrentFile, setTorrentFile] = useState<File | null>(null);
  const [localFile, setLocalFile] = useState<File | null>(null);
  const [seedImmediately, setSeedImmediately] = useState(true);
  const [seedPort, setSeedPort] = useState<number | string>("");
  const [error, setError] = useState<string | null>(null);

  const clearFields = () => {
    setTorrentFile(null);
    setLocalFile(null);
    setSeedImmediately(true);
    setSeedPort("");
    setError(null);
  };

  const handleClickStart = async () => {
    if (torrentFile == null) {
      setError("Torrent file needs to be selected!");
      return;
    }

    if (localFile == null) {
      setError("A file you want to seed needs to be selected!");
      return;
    }

    const formData = new FormData();
    formData.append("seedOnCompletion", seedImmediately ? "1" : "0");
    formData.append("seedPort", seedPort ? seedPort.toString() : "0")
    formData.append("torrentFile", torrentFile!!);
    formData.append("files", localFile!!);

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
        Seed Local Torrent
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>Seed Local Torrent</DialogTitle>
        <DialogContent>
          {error != null && <Alert hidden severity="error">{error}</Alert>}
          <MuiFileInput
            label="Torrent File"
            value={torrentFile}
            margin="normal"
            onChange={(newFile: File | null) => setTorrentFile(newFile)}
            fullWidth
            required
          />
          {/*TODO support multiple files */}
          <MuiFileInput
            label="Local File (file to seed)"
            value={localFile}
            margin="normal"
            onChange={(newFile: File | null) => setLocalFile(newFile)}
            fullWidth
            required
          />
          <FormControlLabel control={
            <Checkbox
              checked={seedImmediately}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
               setSeedImmediately(event.currentTarget.checked);
              }}
            />
          } label="Start seeding immediately" />
         <TextField
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
              setSeedPort("");
            } else {
              setSeedPort(parseInt(event.target.value));
            }
          }}
          fullWidth />
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