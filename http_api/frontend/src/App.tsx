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
import GroupsIcon from '@mui/icons-material/Groups';
import TorrentList from "./TorrentList";

export default function App() {
  const [currentTabIndex, setCurrentTabIndex] = useState(0);
  const [settingsListOpen, setSettingsListOpen] = useState(true);
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
              BitTorrent-over-SCION User Interface
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
              <ListItemButton selected={currentTabIndex === 0} onClick={() => setCurrentTabIndex(0)}>
                <ListItemIcon>
                  <DownloadIcon />
                </ListItemIcon>
                <ListItemText primary="Torrents" />
              </ListItemButton>
              <ListItemButton selected={currentTabIndex === 1} onClick={() => setCurrentTabIndex(1)}>
                <ListItemIcon>
                  <ManageSearchIcon />
                </ListItemIcon>
                <ListItemText primary="Trackers" />
              </ListItemButton>
              <ListItemButton onClick={() => setSettingsListOpen(!settingsListOpen)}>
                <ListItemIcon>
                  <SettingsIcon />
                </ListItemIcon>
                <ListItemText primary="Settings" />
                {settingsListOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </ListItemButton>
              <Collapse in={settingsListOpen} timeout="auto" unmountOnExit>
                <List component="div" disablePadding>
                  <ListItemButton sx={{ pl: 4 }} selected={currentTabIndex === 2} onClick={() => setCurrentTabIndex(2)}>
                    <ListItemIcon>
                      <GroupsIcon />
                    </ListItemIcon>
                    <ListItemText primary="DHT" />
                  </ListItemButton>
                </List>
              </Collapse>
            </List>
          </Box>
        </Drawer>
        <Box component="main" sx={{ flexGrow: 1, p: '1vw' }}>
          <Toolbar />
          {currentTabIndex === 0 &&
            <TorrentList apiConfig={apiConfig}/>
          }
          {currentTabIndex === 1 &&
            <>TODO</>
          }
          {currentTabIndex === 2 &&
            <>TODO</>
          }
        </Box>
      </Box>
    </SnackbarProvider>
  );
}