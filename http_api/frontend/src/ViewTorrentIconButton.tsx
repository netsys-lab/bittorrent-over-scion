import {useState} from 'react';
import {
  Button, Dialog,
  DialogActions,
  DialogContent,
  DialogTitle, Divider, FormControlLabel, IconButton, List,
  ListItem, ListItemText,
  Paper, Stack, Switch, Table, TableBody, TableCell, TableContainer, TableRow, TextField, Tooltip
} from '@mui/material';
import { MuiFileInput } from 'mui-file-input';
import ApiConfig from "./ApiConfig.tsx";
import VisibilityOutlinedIcon from "@mui/icons-material/VisibilityOutlined";
import DownloadIcon from "@mui/icons-material/Download";
import CircularProgressWithLabel from "./CircularProgressWithLabel.tsx";
import { ApiTorrent, ApiFile } from './types.tsx';
import {filesize} from "filesize";

interface ViewTorrentIconButtonProps {
  apiConfig: ApiConfig,
  torrent: ApiTorrent
}

export default function ViewTorrentIconButton({apiConfig, torrent} : ViewTorrentIconButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <div>
      <IconButton edge="end" onClick={() => setOpen(true)}>
        <VisibilityOutlinedIcon />
      </IconButton>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>View Torrent</DialogTitle>
        <DialogContent>
          <Stack spacing={2}>
            <Stack direction="row">
              <MuiFileInput
                label="Torrent File"
                value={new File([], `${torrent.id}.torrent`)}
                margin="normal"
                fullWidth
                hideSizeText
                onClick={(_) => window.open(
                  apiConfig.fileEndpoint(torrent.id, "torrent"),
                  "_blank"
                )}
                disabled
              />
              <IconButton
                edge="end"
                onClick={
                  (_) => window.open(
                    apiConfig.fileEndpoint(torrent.id, "torrent")
                  )
                }
              >
                <DownloadIcon />
              </IconButton>
            </Stack>
            <Divider textAlign="left">Peer Discovery</Divider>
            <FormControlLabel control={
              <Switch checked={torrent.enableDht} disabled />
            } label="Use DHT for peer discovery" />
            <FormControlLabel control={
              <Switch checked={torrent.enableTrackers} disabled />
            } label="Use trackers for peer discovery" />
            <Tooltip title="Comma-separated addresses of peers that will be used in addition to DHT and trackers">
              <TextField
                label="Additional peer addresses"
                type="text"
                placeholder="19-ffaa:1:106d,[127.0.0.1]:43000&#10;17-ffaa:0:cafd,[127.0.0.1]:43000"
                margin="normal"
                InputLabelProps={{
                  shrink: true
                }}
                value={torrent.peers.join("\n")}
                fullWidth
                multiline
                disabled
              />
            </Tooltip>
            <Divider textAlign="left">Metadata</Divider>
            <TableContainer component={Paper} elevation={2}>
              <Table>
                <TableBody>
                  <TableRow>
                    <TableCell component="th" variant="head" scope="row" sx={{ fontWeight: "bold" }}>Piece Length</TableCell>
                    <TableCell align="right">{filesize(torrent.pieceLength, {bits: false})}</TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell component="th" variant="head" scope="row" sx={{ fontWeight: "bold" }}>Number of Pieces</TableCell>
                    <TableCell align="right">{torrent.numPieces}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableContainer>
            <Divider textAlign="left">Files</Divider>
            <Paper elevation={2}>
              <List>
                {torrent.files.map((file: ApiFile) => {
                  const progress = file.progress / file.length * 100;
                  let downloadButton = <></>;

                  if (progress >= 100.0) {
                    downloadButton = (
                      <IconButton
                        edge="end"
                        onClick={
                          (_) => window.open(
                            apiConfig.fileEndpoint(torrent.id, file.id)
                          )
                        }
                      >
                        <DownloadIcon />
                      </IconButton>
                    );
                  }

                  return (
                    <ListItem
                      key={file.id}
                      secondaryAction={
                        <Stack direction="row" spacing={1}>
                          {downloadButton}
                        </Stack>
                      }
                      disablePadding
                    >
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <CircularProgressWithLabel variant="determinate" label={`${Math.round(progress)}%`} value={progress} color="primary" />
                        <ListItemText primary={file.path} secondary={`${filesize(file.progress, {bits: false})} / ${filesize(file.length, {bits: false})}`} />
                      </Stack>
                    </ListItem>
                  );
                })}
              </List>
            </Paper>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}