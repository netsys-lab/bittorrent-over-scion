import ApiConfig from "./ApiConfig.tsx";
import {Component} from "react";
import {ApiTorrents} from "./types.tsx";
import {
  Checkbox, CircularProgress, Grid,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Paper,
  Stack
} from "@mui/material";
import AddTorrentButton from "./AddTorrentButton.tsx";
import {OverridableStringUnion} from "@mui/types";
import {filesize} from "filesize";
import DownloadIcon from "@mui/icons-material/Download";
import DeleteTorrentIconButton from "./DeleteTorrentIconButton.tsx";
import ViewTorrentIconButton from "./ViewTorrentIconButton.tsx";
import CircularProgressWithLabel from "./CircularProgressWithLabel.tsx";

interface TorrentListProps {
  apiConfig: ApiConfig,
}

interface TorrentListState {
  checked: Array<number>
  torrents: ApiTorrents
  loaded: boolean
}

export default class TorrentList extends Component<TorrentListProps, TorrentListState> {
  public state : TorrentListState = { checked: [], torrents: [], loaded: false };
  private timerID = -1;

  componentDidMount() {
    this.timerID = setInterval(async () => { await this.refreshTorrents() }, 1000);
  }

  componentWillUnmount() {
    clearInterval(this.timerID);
  }

  async refreshTorrents() {
    console.log(this.props.apiConfig.torrentEndpoint());
    const response = await fetch(this.props.apiConfig.torrentEndpoint());
    const torrents = await response.json();
    //TODO error handling

    console.log(torrents);
    this.setState({ torrents: torrents, loaded: true });
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
      <Grid container rowSpacing={1} sx={{ width: '78vw' }}>
        <Grid item xs={12}>
          <AddTorrentButton apiConfig={this.props.apiConfig}/>
        </Grid>
        <Grid item xs={12}>
          <Paper elevation={3}>
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
                            this.props.apiConfig.fileEndpoint(torrentId, fileId)
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
                  deleteButton = <DeleteTorrentIconButton apiConfig={this.props.apiConfig} torrentId={torrentId} />;
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
                        <ViewTorrentIconButton apiConfig={this.props.apiConfig} torrent={torrent} />
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
              {(this.state.loaded && Object.keys(this.state.torrents).length == 0) &&
                <ListItem>
                  No torrents yet. Add one with the buttons above!
                </ListItem>
              }
              {!this.state.loaded &&
                <ListItem>
                  <CircularProgress />
                </ListItem>
              }
            </List>
          </Paper>
        </Grid>
      </Grid>
    );
  }
}