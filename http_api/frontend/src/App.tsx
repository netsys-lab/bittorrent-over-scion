import {useState} from "react";
import ApiConfig from "./ApiConfig";
import {
  AppBar,
  Box,
  Collapse,
  CssBaseline,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography
} from "@mui/material";
import {closeSnackbar, SnackbarKey, SnackbarProvider} from "notistack";
import CloseIcon from "@mui/icons-material/Close";
import DownloadIcon from "@mui/icons-material/Download";
import ManageSearchIcon from '@mui/icons-material/ManageSearch';
import SettingsIcon from '@mui/icons-material/Settings';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ViewListIcon from '@mui/icons-material/ViewList';
import UploadIcon from '@mui/icons-material/Upload';
import SwapVertIcon from '@mui/icons-material/SwapVert';
import TorrentList from "./TorrentList";
import TrackerList from "./TrackerList.tsx";
import {ApiTorrentState, NonSeedingTorrentStates} from "./types.tsx";
import Settings from "./Settings.tsx";

export default function App() {
  const [currentTabIndex, setCurrentTabIndex] = useState(0);
  const [torrentsListOpen, setTorrentsListOpen] = useState(true);
  const apiConfig = new ApiConfig();
  const snackbarCloseAction = (snackbarId: SnackbarKey) => (
    <IconButton aria-label="delete" onClick={() => { closeSnackbar(snackbarId) }}>
      <CloseIcon />
    </IconButton>
  );

  return (
    <SnackbarProvider autoHideDuration={3000} action={snackbarCloseAction}>
      <Box sx={{ display: 'flex', width: '100vw' }}>
        <CssBaseline />
        <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
          <Toolbar>
            <Typography variant="h6" noWrap component="div">
              BitTorrent over SCION
            </Typography>
          </Toolbar>
        </AppBar>
        <Drawer
          variant="permanent"
          sx={{
            width: '20vw',
            flexShrink: 0,
            [`& .MuiDrawer-paper`]: { width: '20vw', boxSizing: 'border-box' },
          }}
        >
          <Toolbar />
          <Box sx={{ overflow: 'auto' }}>
            <List>
              <ListItemButton onClick={() => setTorrentsListOpen(!torrentsListOpen)}>
                <ListItemIcon>
                  <ViewListIcon />
                </ListItemIcon>
                <ListItemText primary="Torrents" />
                {torrentsListOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </ListItemButton>
              <Collapse in={torrentsListOpen} timeout="auto" unmountOnExit>
                <List component="div" disablePadding>
                  <ListItemButton sx={{ pl: 4 }} selected={currentTabIndex === 0} onClick={() => setCurrentTabIndex(0)}>
                    <ListItemIcon>
                      <SwapVertIcon />
                    </ListItemIcon>
                    <ListItemText primary="All Torrents" />
                  </ListItemButton>
                  <ListItemButton sx={{ pl: 4 }} selected={currentTabIndex === 1} onClick={() => setCurrentTabIndex(1)}>
                    <ListItemIcon>
                      <DownloadIcon />
                    </ListItemIcon>
                    <ListItemText primary="Downloads only" />
                  </ListItemButton>
                  <ListItemButton sx={{ pl: 4 }} selected={currentTabIndex === 2} onClick={() => setCurrentTabIndex(2)}>
                    <ListItemIcon>
                      <UploadIcon />
                    </ListItemIcon>
                    <ListItemText primary="Seeders only" />
                  </ListItemButton>
                </List>
              </Collapse>
              <ListItemButton selected={currentTabIndex === 3} onClick={() => setCurrentTabIndex(3)}>
                <ListItemIcon>
                  <ManageSearchIcon />
                </ListItemIcon>
                <ListItemText primary="Trackers" />
              </ListItemButton>
              <ListItemButton selected={currentTabIndex === 4} onClick={() => setCurrentTabIndex(4)}>
                <ListItemIcon>
                  <SettingsIcon />
                </ListItemIcon>
                <ListItemText primary="Settings" />
              </ListItemButton>
            </List>
          </Box>
        </Drawer>
        <Box component="main" sx={{ flexGrow: 1, p: '1vw' }}>
          <Toolbar />
          {currentTabIndex === 0 &&
            <TorrentList apiConfig={apiConfig} wantedTorrentStates={[]} />
          }
          {currentTabIndex === 1 &&
            <TorrentList apiConfig={apiConfig} wantedTorrentStates={NonSeedingTorrentStates} />
          }
          {currentTabIndex === 2 &&
            <TorrentList apiConfig={apiConfig} wantedTorrentStates={[ApiTorrentState.Seeding]} />
          }
          {currentTabIndex === 3 &&
            <TrackerList apiConfig={apiConfig} />
          }
          {currentTabIndex === 4 &&
            <Settings apiConfig={apiConfig} />
          }
        </Box>
      </Box>
    </SnackbarProvider>
  );
}