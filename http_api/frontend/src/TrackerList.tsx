import ApiConfig from "./ApiConfig.tsx";
import {Component} from "react";
import {ApiTrackers} from "./types.tsx";
import {
  Checkbox,
  CircularProgress,
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
import DeleteIcon from "@mui/icons-material/Delete";
import AddTrackerButton from "./AddTrackerButton.tsx";
import {enqueueSnackbar} from "notistack";

interface TrackerListProps {
  apiConfig: ApiConfig,
}

interface TrackerListState {
  checked: Array<number>
  trackers: ApiTrackers
  loaded: boolean
}

export default class TrackerList extends Component<TrackerListProps, TrackerListState> {
  public state : TrackerListState = { checked: [], trackers: [], loaded: false };

  componentDidMount() {
    this.refreshTrackers().then(() => {});
  }

  async refreshTrackers() {
    const response = await fetch(this.props.apiConfig.trackerEndpoint());
    const trackers = await response.json();
    //TODO error handling

    console.log(trackers);
    this.setState({ trackers: trackers, loaded: true });
  }

  async deleteTracker(trackerId: number) {
    try {
      const response = await fetch(
        this.props.apiConfig.trackerEndpoint(trackerId),
        {
          method: "DELETE",
        }
      );
      const body = await response.json();

      if (!response.ok) {
        enqueueSnackbar(
          "Deleting tracker failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Successfully deleted tracker!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        "Deleting tracker failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log("Deleting tracker with id " + trackerId + " failed:", error);
    }

    this.refreshTrackers().then(() => {});
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
          <AddTrackerButton apiConfig={this.props.apiConfig} postAddFunc={() => this.refreshTrackers().then(() => {})} />
        </Grid>
        <Grid item xs={12}>
          <Paper elevation={3}>
            <List>
              {Object.keys(this.state.trackers).map((value) => {
                const trackerId = parseInt(value);
                const tracker = this.state.trackers[trackerId];

                return (
                  <ListItem
                    key={value}
                    secondaryAction={
                      <Stack direction="row" spacing={1}>
                        <IconButton
                          edge="end"
                          onClick={async () => { await this.deleteTracker(trackerId) }}
                        >
                          <DeleteIcon />
                        </IconButton>
                      </Stack>
                    }
                    disablePadding
                  >
                    <ListItemButton onClick={() => this.handleToggle(trackerId)} dense>
                      <ListItemIcon>
                        <Checkbox
                          edge="start"
                          checked={this.state.checked.indexOf(trackerId) !== -1}
                          tabIndex={-1}
                          disableRipple
                        />
                      </ListItemIcon>
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <ListItemText
                          primary={tracker.url}
                        />
                      </Stack>
                    </ListItemButton>
                  </ListItem>
                );
              })}
              {(this.state.loaded && Object.keys(this.state.trackers).length == 0) &&
                <ListItem>
                  No trackers yet. Add one with the button above, or add torrent files that contain associated trackers!
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