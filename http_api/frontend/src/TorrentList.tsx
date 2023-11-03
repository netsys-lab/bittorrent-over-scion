import ApiConfig from "./ApiConfig.tsx";
import {Component} from "react";
import {ApiTorrents} from "./types.tsx";
import {
  Checkbox,
  CircularProgress,
  Divider,
  Grid,
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
import SeedSwitch from "./SeedSwitch.tsx";

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
                let progress = <></>;

                let finished = false;
                let progressValue = 0;
                let progressColor : OverridableStringUnion<'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' | 'inherit'> = 'primary';
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
                    if (torrent.status) {
                      status = torrent.status;
                    } else {
                      status = `${torrent.numDownloadedPieces}/${torrent.numPieces} pieces | ${torrent.files.length}/${torrent.files.length} files`
                    }

                    downloadButton = (
                      <IconButton
                        edge="end"
                        onClick={
                          (_) => window.open(
                            this.props.apiConfig.fileEndpoint(torrentId, torrent.files[0].id)
                          )
                        }
                      >
                        <DownloadIcon />
                      </IconButton>
                    );

                    finished = true;
                    break;
                  case 'failed':
                    progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                    progressColor = 'error';
                    status = 'dowloading torrent failed'; //TODO add error info
                    finished = true;
                    break;
                  case 'cancelled':
                    progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                    progressColor = 'error';
                    status = 'cancelled by user';
                    finished = true;
                    break;
                  case 'seeding':
                    progress = <CircularProgressWithLabel label="SEED" />;
                    status = 'seeding at ' + torrent.seedAddr.toString();

                    downloadButton = (
                      <IconButton
                        edge="end"
                        onClick={
                          (_) => window.open(
                            this.props.apiConfig.fileEndpoint(torrentId, torrent.files[0].id)
                          )
                        }
                      >
                        <DownloadIcon />
                      </IconButton>
                    );

                    finished = true;
                    break;
                }

                if (torrent.state != 'seeding') {
                  progress = <CircularProgressWithLabel variant="determinate" label={`${Math.round(progressValue)}%`} value={progressValue} color={progressColor} />;
                }

                if (finished || torrent.state == 'not started yet') {
                  deleteButton = <DeleteTorrentIconButton apiConfig={this.props.apiConfig} torrentId={torrentId} />;
                }

                return (
                  <ListItem
                    key={value}
                    secondaryAction={
                      <Stack direction="row" spacing={1}>
                        <SeedSwitch apiConfig={this.props.apiConfig} torrentId={torrentId} seedOnCompletion={torrent.seedOnCompletion} />
                        <Divider orientation="vertical" variant="middle" flexItem />
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
                        {progress}
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