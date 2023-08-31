import {useState} from 'react';
import {
  Button, Dialog,
  DialogActions,
  DialogContent,
  DialogTitle, Divider, IconButton, List,
  ListItem, ListItemText,
  Paper, Stack, Table, TableBody, TableCell, TableContainer, TableRow, TextField, Typography
} from '@mui/material';
import { MuiFileInput } from 'mui-file-input';
import ApiConfig from "./ApiConfig.tsx";
import VisibilityOutlinedIcon from "@mui/icons-material/VisibilityOutlined";
import DownloadIcon from "@mui/icons-material/Download";
import CircularProgressWithLabel from "./CircularProgressWithLabel.tsx";
import { ApiTorrent, ApiFile } from './types.tsx';

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
            <TextField
              id="name"
              label="Peer"
              type="text"
              placeholder="19-ffaa:1:106d,[127.0.0.1]:43000"
              margin="normal"
              InputLabelProps={{
                shrink: true
              }}
              value={torrent.peer}
              fullWidth
              disabled
            />
            <Divider />
            <div>
              <Typography variant="subtitle1">
                Metadata
              </Typography>
              <TableContainer component={Paper} elevation={2}>
                <Table>
                  <TableBody>
                    <TableRow>
                      <TableCell component="th" variant="head" scope="row" sx={{ fontWeight: "bold" }}>Piece Length</TableCell>
                      <TableCell align="right">{torrent.pieceLength} B</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell component="th" variant="head" scope="row" sx={{ fontWeight: "bold" }}>Number of Pieces</TableCell>
                      <TableCell align="right">{torrent.numPieces}</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </div>
            <Divider />
            <div>
              <Typography variant="subtitle1">
                Files
              </Typography>
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
                          <CircularProgressWithLabel value={progress} color="primary" />
                          <ListItemText primary={file.path} secondary={`${file.progress}/${file.length} bytes`} />
                        </Stack>
                      </ListItem>
                    );
                  })}
                </List>
              </Paper>
            </div>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </div>
  );
}