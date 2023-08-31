import {Component} from 'react';
import {
  CssBaseline,
  Box,
  Grid,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemButton,
  ListItemIcon,
  Stack,
  IconButton,
  Checkbox, Typography
} from '@mui/material';
import {OverridableStringUnion} from "@mui/types";
import DownloadIcon from '@mui/icons-material/Download';
import CloseIcon from '@mui/icons-material/Close';
import CircularProgressWithLabel from './CircularProgressWithLabel.tsx';
import './App.css';
import ApiConfig from "./ApiConfig.tsx";
import AddTorrentButton from "./AddTorrentButton.tsx";
import {SnackbarProvider, closeSnackbar, SnackbarKey} from 'notistack';
import DeleteTorrentIconButton from "./DeleteTorrentIconButton.tsx";
import ViewTorrentIconButton from './ViewTorrentIconButton.tsx';
import { ApiTorrents } from './types.tsx';
import { filesize } from "filesize";

interface AppState {
  checked: Array<number>
  torrents: ApiTorrents
}

class App extends Component<{}, AppState> {
  //const [count, setCount] = useState(0)
  public state : AppState = { checked: [], torrents: [] };
  private timerID = -1;
  private apiConfig = new ApiConfig();
  private snackbarCloseAction = (snackbarId: SnackbarKey) => (
    <IconButton aria-label="delete" onClick={() => { closeSnackbar(snackbarId) }}>
      <CloseIcon />
    </IconButton>
  );

  componentDidMount() {
    this.timerID = setInterval(async () => { await this._refreshTorrents() }, 1000);
  }

  componentWillUnmount() {
    clearInterval(this.timerID);
  }

  async _refreshTorrents() {
    console.log(this.apiConfig.torrentEndpoint());
    const response = await fetch(this.apiConfig.torrentEndpoint());
    const torrents = await response.json();
    //TODO error handling

    console.log(torrents);
    this.setState({ torrents: torrents });
  }

  handleToggle(value: number) {
    const currentIndex = this.state.checked.indexOf(value);
    const newChecked = [...this.state.checked];

    if (currentIndex === -1) {
      newChecked.push(value);
    } else {
      newChecked.splice(currentIndex, 1);
    }

    this.setState({checked: newChecked});
  }

  render() {
    return (
      <SnackbarProvider autoHideDuration={3000} action={this.snackbarCloseAction}>
        <CssBaseline />
        <Box sx={{ width: '60vw' }}>
          <Grid container spacing={1}>
            <Grid item xs={12}>
              <Typography variant="h5">
                BitTorrent-over-SCION UI
              </Typography>
            </Grid>
            <Grid item xs={12}>
              <AddTorrentButton apiConfig={this.apiConfig}/>
            </Grid>
            <Grid item xs={12}>
              <Paper elevation={2}>
                <List>
                  {Object.keys(this.state.torrents).map((value) => {
                    const torrentId = parseInt(value);
                    const torrent = this.state.torrents[torrentId];
                    let downloadButton = <></>;
                    let deleteButton = <></>;

                    let finished = false;
                    let progressValue : number;
                    let progressColor : OverridableStringUnion<'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' | 'inherit'>;
                    let status = '';
                    switch (torrent.state) {
                      case 'running':
                        progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                        progressColor = 'info';
                        status = `${torrent.numDownloadedPieces}/${torrent.numPieces} pieces | rx: ${filesize(torrent.metrics.rx, {bits: true})}/s | tx: ${filesize(torrent.metrics.tx, {bits: true})}/s | #conns: ${torrent.metrics.numConns} | #paths: ${torrent.metrics.numPaths}`

                        break;
                      case 'completed':
                        progressValue = 100;
                        progressColor = 'success';
                        status = `${torrent.numDownloadedPieces}/${torrent.numPieces} pieces | ${torrent.files.length}/${torrent.files.length} files`

                        const fileId = torrent.files[0].id;
                        downloadButton = (
                          <IconButton
                            edge="end"
                            onClick={
                              (_) => window.open(
                                this.apiConfig.fileEndpoint(torrentId, fileId)
                              )
                            }
                          >
                            <DownloadIcon />
                          </IconButton>
                        );

                        finished = true;
                        break;
                      case 'failed':
                      case 'cancelled':
                        progressColor = 'error';
                        progressValue = 100;
                        finished = true;
                        break;
                      default:
                        progressValue = 0;
                        progressColor = 'primary';
                        break;
                    }

                    if (finished || torrent.state == 'not started yet') {
                      deleteButton = <DeleteTorrentIconButton apiConfig={this.apiConfig} torrentId={torrentId} />;
                    }

                    return (
                      <ListItem
                        key={value}
                        secondaryAction={
                          <Stack direction="row" spacing={1}>
                            {/*<FormControlLabel
                              value="start"
                              control={<Switch color="primary" />}
                              label="Seed"
                              labelPlacement="start"
                            />
                            <Divider orientation="vertical" variant="middle" flexItem />*/}
                            <ViewTorrentIconButton apiConfig={this.apiConfig} torrent={torrent} />
                            {downloadButton}
                            {deleteButton}
                          </Stack>
                        }
                        disablePadding
                      >
                        <ListItemButton onClick={() => this.handleToggle(torrentId)} dense>
                          <ListItemIcon>
                            <Checkbox
                              edge="start"
                              checked={this.state.checked.indexOf(torrentId) !== -1}
                              tabIndex={-1}
                              disableRipple
                            />
                          </ListItemIcon>
                          {/*<ListItemAvatar>
                            <Avatar>{123}</Avatar>
                          </ListItemAvatar>*/}
                          <Stack direction="row" alignItems="center" spacing={1}>
                            <CircularProgressWithLabel value={progressValue} color={progressColor} />
                            <ListItemText
                              primary={torrent.name}
                              secondary={status != '' ? status : false}
                            />
                          </Stack>

                        </ListItemButton>
                      </ListItem>
                    );
                  })}
                  {Object.keys(this.state.torrents).length == 0 &&
                    <ListItem>
                      No torrents yet. Add one with the buttons above!
                    </ListItem>
                  }
                </List>
              </Paper>
            </Grid>
          </Grid>
        </Box>
      </SnackbarProvider>
    );
  }
}

export default App;
