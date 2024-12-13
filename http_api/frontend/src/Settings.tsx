import ApiConfig from "./ApiConfig.tsx";
import {ChangeEvent, Component} from "react";
import {CircularProgress, Divider, TextField} from "@mui/material";
import {enqueueSnackbar} from "notistack";

interface SettingsProps {
  apiConfig: ApiConfig,
}

interface SettingsState {
  dhtPort: number
  dhtBootstrapNodes: Array<string>
  loaded: boolean
}

export default class Settings extends Component<SettingsProps, SettingsState> {
  public state : SettingsState = { dhtPort: 0, dhtBootstrapNodes: [], loaded: false };

  componentDidMount() {
    this.refreshSettings().then(() => {});
  }

  async refreshSettings() {
    const response = await fetch(this.props.apiConfig.settingsEndpoint());
    const settings = await response.json();
    //TODO error handling

    this.setState({ dhtPort: settings.dhtPort, dhtBootstrapNodes: settings.dhtBootstrapNodes, loaded: true });
  }

  async updateSetting(key: string, value: string | Array<string>) {
    try {
      const formData = new FormData();
      if (value instanceof Array) {
        for (const i in value) {
          formData.append(key, value[i]);
        }
      } else {
        formData.append(key, value);
      }

      const response = await fetch(
        this.props.apiConfig.settingsEndpoint(),
        {
          method: "POST",
          body: formData,
        }
      );
      const body = await response.json();

      if (!response.ok) {
        enqueueSnackbar(
          "Saving setting failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Saved!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        "Saving setting failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log("Saving setting " + key + " failed:", error);
    }
  }

  render() {
    return (
      <>
        {this.state.loaded &&
          <>
              <Divider textAlign="left">DHT-related Settings</Divider>
              <TextField
                label="DHT Port"
                type="number"
                placeholder="0-65535"
                margin="normal"
                InputLabelProps={{
                  shrink: true
                }}
                value={this.state.dhtPort}
                onChange={(event: ChangeEvent<HTMLInputElement>) => {
                  this.setState({ dhtPort: parseInt(event.target.value) });
                }}
                onBlur={event => {
                  this.updateSetting("dhtPort", event.target.value).then(() => {});
                }}
                fullWidth
              />
              <TextField
                label="DHT Bootstrap Nodes"
                type="text"
                placeholder="19-ffaa:1:106d,[127.0.0.1]:43000&#10;17-ffaa:0:cafd,[127.0.0.1]:43000"
                margin="normal"
                InputLabelProps={{
                  shrink: true
                }}
                value={this.state.dhtBootstrapNodes.join("\n")}
                onChange={(event: ChangeEvent<HTMLInputElement>) => {
                  this.setState({ dhtBootstrapNodes: event.target.value.split("\n") });
                }}
                onBlur={event => {
                  this.updateSetting("dhtBootstrapNodes", event.target.value.split("\n")).then(() => {});
                }}
                multiline
                rows={4}
                fullWidth
              />
          </>
        }
        {!this.state.loaded &&
          <CircularProgress />
        }
      </>
    );
  }
}