import {useState, ChangeEvent} from 'react';
import {Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField} from '@mui/material';
import { useSnackbar } from 'notistack';
import ApiConfig from "./ApiConfig.tsx";

interface AddTrackerButtonProps {
  apiConfig: ApiConfig,
  postAddFunc: () => void
}

export default function AddTrackerButton({apiConfig, postAddFunc} : AddTrackerButtonProps) {
  const { enqueueSnackbar} = useSnackbar();

  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState("");
  const [error, setError] = useState<string | null>(null);

  const clearFields = () => {
    setUrl("");
    setError(null);
  };

  const handleClickAdd = async () => {
    if (url.length == 0) {
      setError("Peer field needs to be filled out!");
      return;
    }

    const formData = new FormData();
    formData.append("url", url);

    try {
      const response = await fetch(apiConfig.trackerEndpoint(), {
        method: "POST",
        body: formData,
      });
      const body = await response.json();

      if (!response.ok) {
        setError(body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!");
        return;
      }

      enqueueSnackbar("Successfully added tracker!", {variant: "success"});
      setOpen(false);
      clearFields();
    } catch (error) {
      setError("Connection error! API offline? (" + error + ")");
    }

    postAddFunc();
  };

  return (
    <div>
      <Button variant="contained" onClick={() => setOpen(true)}>
        Add Tracker
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>Add Tracker</DialogTitle>
        <DialogContent>
          {error != null && <Alert hidden severity="error">{error}</Alert>}
          <TextField
              id="url"
              label="URL"
              type="text"
              placeholder="http://tracker.openbittorrent.com:80/announce"
              margin="normal"
              InputLabelProps={{
                shrink: true
              }}
              value={url}
              onChange={(event: ChangeEvent<HTMLInputElement>) => {
                setUrl(event.target.value);
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