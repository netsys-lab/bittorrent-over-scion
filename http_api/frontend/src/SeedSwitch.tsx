import {FormControlLabel, Switch} from "@mui/material";
import ApiConfig from "./ApiConfig.tsx";
import {useSnackbar} from "notistack";

interface SeedSwitchProps {
  apiConfig: ApiConfig,
  torrentId: number,
  seedOnCompletion: boolean
}

export default function SeedSwitch({apiConfig, torrentId, seedOnCompletion} : SeedSwitchProps) {
  const {enqueueSnackbar} = useSnackbar();

  const handleClick = async () => {
    const formData = new FormData();
    formData.append("seedOnCompletion", !seedOnCompletion ? "1" : "0");

    try {
      const response = await fetch(
        apiConfig.torrentEndpoint(torrentId),
        {
          method: "POST",
          body: formData
        }
      );
      const body = await response.json();

      if (!response.ok) {
        enqueueSnackbar(
          (seedOnCompletion ? "Disabling" : "Enabling") + " seeding on completion failed: " + body.error.charAt(0).toUpperCase() + body.error.slice(1) + "!",
          {
            variant: "error",
            persist: true
          }
        );
      } else {
        enqueueSnackbar("Successfully " + (seedOnCompletion ? "disabled" : "enabled") + " seeding on completion!", {variant: "success"});
      }
    } catch (error) {
      enqueueSnackbar(
        (seedOnCompletion ? "Disabling" : "Enabling") + " seeding on completion failed: Connection error! API offline? (more info on console)",
        {
          variant: "error",
          persist: true
        }
      );
      console.log((seedOnCompletion ? "Disabling" : "Enabling") + " seeding on completion for torrent with id " + torrentId + " failed:", error);
    }
  }

  return (
    <FormControlLabel
      value="start"
      control={
        <Switch
          checked={seedOnCompletion}
          onChange={handleClick}
          color="primary"
        />
      }
      label="Seed"
      labelPlacement="start"
    />
  );
}
