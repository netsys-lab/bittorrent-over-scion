import ApiConfig from "./ApiConfig.tsx";
import {Component} from "react";
import {ApiTorrents, ApiTorrentState} from "./types.tsx";
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
import DownloadTorrentButton from "./DownloadTorrentButton.tsx";
import {OverridableStringUnion} from "@mui/types";
import {filesize} from "filesize";
import DownloadIcon from "@mui/icons-material/Download";
import DeleteTorrentIconButton from "./DeleteTorrentIconButton.tsx";
import ViewTorrentIconButton from "./ViewTorrentIconButton.tsx";
import CircularProgressWithLabel from "./CircularProgressWithLabel.tsx";
import SeedSwitch from "./SeedSwitch.tsx";
import SeedTorrentButton from "./SeedTorrentButton.tsx";
import CancelTorrentIconButton from "./CancelTorrentIconButton.tsx";
import RetryTorrentIconButton from "./RetryTorrentIconButton.tsx";

interface TorrentListProps {
  apiConfig: ApiConfig
  wantedTorrentStates: Array<ApiTorrentState>
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
    let endpoint = this.props.apiConfig.torrentEndpoint()
    if (this.props.wantedTorrentStates.length > 0) {
      endpoint += '?wantedStates=' + this.props.wantedTorrentStates.join(',');
    }
    console.log(endpoint);
    const response = await fetch(endpoint);
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
          <Stack direction="row" spacing={1}>
            <DownloadTorrentButton apiConfig={this.props.apiConfig} />
            <SeedTorrentButton apiConfig={this.props.apiConfig} />
          </Stack>
        </Grid>
        <Grid item xs={12}>
          <Paper elevation={3}>
            <List>
              {Object.keys(this.state.torrents).map((value) => {
                const torrentId = parseInt(value);
                const torrent = this.state.torrents[torrentId];

                let seedSwitch = <></>;
                let cancelButton = <></>;
                let downloadButton = <></>;
                let retryButton = <></>;
                let deleteButton = <></>;
                let progress = <></>;

                let finished = false;
                let progressValue = 0;
                let progressColor : OverridableStringUnion<'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' | 'inherit'> = 'primary';
                let status = '';
                switch (torrent.state) {
                  case ApiTorrentState.Running:
                    progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                    progressColor = 'info';
                    status = `${torrent.numDownloadedPieces}/${torrent.numPieces} pieces | rx: ${filesize(torrent.metrics.rx, {bits: true})}/s | tx: ${filesize(torrent.metrics.tx, {bits: true})}/s | #conns: ${torrent.metrics.numConns} | #paths: ${torrent.metrics.numPaths}`
                    cancelButton = <CancelTorrentIconButton apiConfig={this.props.apiConfig} torrentId={torrentId} />;
                    break;
                  case ApiTorrentState.Completed:
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
                  case ApiTorrentState.Failed:
                    progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                    progressColor = 'error';
                    status = 'Downloading torrent failed: ' + torrent.status;
                    finished = true;
                    break;
                  case ApiTorrentState.Cancelled:
                    progressValue = torrent.numDownloadedPieces / torrent.numPieces * 100;
                    progressColor = 'error';
                    status = 'Cancelled by user';
                    finished = true;
                    break;
                  case ApiTorrentState.Seeding:
                    progress = <CircularProgressWithLabel label="SEED" />;
                    status = 'Seeding at ' + torrent.seedAddr.toString();

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

                if (torrent.state != ApiTorrentState.Seeding) {
                  progress = <CircularProgressWithLabel variant="determinate" label={`${Math.round(progressValue)}%`} value={progressValue} color={progressColor} />;

                  if (finished && torrent.state != ApiTorrentState.Completed) {
                    retryButton = <RetryTorrentIconButton apiConfig={this.props.apiConfig} torrentId={torrentId} />;
                  }
                }

                if (torrent.state == ApiTorrentState.Seeding || torrent.state == ApiTorrentState.Completed) {
                  seedSwitch = <>
                    <SeedSwitch apiConfig={this.props.apiConfig} torrentId={torrentId} seedOnCompletion={torrent.seedOnCompletion} />
                    <Divider orientation="vertical" variant="middle" flexItem />
                  </>;
                }

                if (finished || torrent.state == ApiTorrentState.NotStartedYet) {
                  deleteButton = <DeleteTorrentIconButton apiConfig={this.props.apiConfig} torrentId={torrentId} />;
                }

                return (
                  <ListItem
                    key={value}
                    secondaryAction={
                      <Stack direction="row" spacing={1}>
                        {seedSwitch}
                        <ViewTorrentIconButton apiConfig={this.props.apiConfig} torrent={torrent} />
                        {downloadButton}
                        {cancelButton}
                        {retryButton}
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