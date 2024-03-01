package http_api

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	util "github.com/netsys-lab/bittorrent-over-scion/Utils"
	"github.com/netsys-lab/bittorrent-over-scion/http_api/storage"
	"github.com/netsys-lab/bittorrent-over-scion/p2p"
	peers2 "github.com/netsys-lab/bittorrent-over-scion/peers"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
	"github.com/scionproto/scion/go/lib/snet"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

type HttpApi struct {
	LocalAddr          string
	MaxRequestBodySize int
	ScionLocalHost     string
	NumPaths           int
	DialBackStartPort  uint16
	SeedStartPort      uint16
	EnableDht          bool
	DhtPort            uint16
	DhtBootstrapAddr   string

	Storage *storage.Storage

	torrents     map[uint64]*storage.Torrent
	trackers     map[uint64]*storage.Tracker
	usedUdpPorts map[uint16]bool
	localAddr    *snet.UDPAddr
}

type ErrorResponseBody struct {
	Error string `json:"error"`
}

func getInfoHandler(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	defaultHandler(w, &struct {
		Version string `json:"version"`
	}{
		Version: "0.0.1",
	})
}

func listTorrentsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	torrents := make(map[uint64]*storage.Torrent)

	wantedStates := make([]storage.State, 0)
	wantedStatesStr := r.URL.Query().Get("wantedStates")
	if len(wantedStatesStr) > 0 {
		wantedStatesStrs := strings.Split(wantedStatesStr, ",")
		for _, wantedStateStr := range wantedStatesStrs {
			wantedState, ok := storage.StringToState[wantedStateStr]
			if !ok {
				errorHandler(w, http.StatusBadRequest, "invalid wanted state specified")
				return
			}
			wantedStates = append(wantedStates, wantedState)
		}

		// filter wanted states
		for torrentId, torrent := range api.torrents {
			for _, wantedState := range wantedStates {
				if wantedState == torrent.State {
					torrents[torrentId] = torrent
					break
				}
			}
		}
	} else {
		for torrentId, torrent := range api.torrents {
			torrents[torrentId] = torrent
		}
	}

	defaultHandler(w, torrents)
}

func listTrackersHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	defaultHandler(w, api.trackers)
}

func getTorrentByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("torrent"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	torrent, exists := api.torrents[id]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}
	defaultHandler(w, torrent)
}

func getFileByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)

	torrentId, err := strconv.ParseUint(p.ByName("torrent"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid torrent ID specified")
		return
	}

	torrent, exists := api.torrents[torrentId]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}

	// to serve torrent file instead
	if p.ByName("file") == "torrent" {
		// make downloadable
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%d.torrent", torrent.ID))

		// serve file
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(torrent.RawTorrentFile))
		return
	}

	fileId, err := strconv.ParseUint(p.ByName("file"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid file ID specified")
		return
	}

	for _, file := range torrent.Files {
		if file.ID == fileId {
			// make downloadable
			w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(file.Path))

			// serve file
			http.ServeFile(w, r, torrent.GetFileDir(api.Storage.FS)+"/"+file.Path)
			return
		}
	}

	errorHandler(w, http.StatusNotFound, "file with given ID not found")
}

func getTrackerByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("tracker"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	tracker, exists := api.trackers[id]
	if !exists {
		errorHandler(w, http.StatusNotFound, "tracker with given ID not found")
		return
	}
	defaultHandler(w, tracker)
}

func addTorrentHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)

	// limit request body (torrent file)
	r.Body = http.MaxBytesReader(w, r.Body, int64(api.MaxRequestBodySize))
	err := r.ParseMultipartForm(int64(api.MaxRequestBodySize))
	if err != nil {
		errorHandler(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body too large (maximum %d bytes)", api.MaxRequestBodySize))
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		errorHandler(w, http.StatusUnsupportedMediaType, "invalid content type (\"multipart/form-data\" wanted)")
		return
	}

	seedOnCompletionStr := r.FormValue("seedOnCompletion")
	seedOnCompletionBool := false
	if len(seedOnCompletionStr) > 0 {
		seedOnCompletionBool, err = strconv.ParseBool(seedOnCompletionStr)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"seedOnCompletion\" specified (boolean wanted)")
			return
		}
	}

	seedPortStr := r.FormValue("seedPort")
	var seedPortNum uint64
	if len(seedPortStr) > 0 {
		seedPortNum, err = strconv.ParseUint(seedPortStr, 10, 16)
		if err != nil || seedPortNum > 65535 {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"seedPort\" specified (0-65535 wanted)")
			return
		}
	}

	enableDhtStr := r.FormValue("enableDht")
	enableDhtBool := false
	if len(enableDhtStr) > 0 {
		enableDhtBool, err = strconv.ParseBool(enableDhtStr)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"enableDht\" specified (boolean wanted)")
			return
		}
	}

	enableTrackersStr := r.FormValue("enableTrackers")
	enableTrackersBool := false
	if len(enableTrackersStr) > 0 {
		enableTrackersBool, err = strconv.ParseBool(enableTrackersStr)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"enableTrackers\" specified (boolean wanted)")
			return
		}
	}

	/* handle uploaded torrent file */

	file, remoteFileHdr, err := r.FormFile("torrentFile")
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "file field \"torrentFile\" as part of POST form data is missing")
		return
	}

	// read torrent remoteFile into byte slice
	fileBuf := make([]byte, remoteFileHdr.Size)
	_, err = file.Read(fileBuf)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, "torrent file could not be read")
		return
	}

	// parse torrent file
	torrentFile, err := torrentfile.Parse(bytes.NewReader(fileBuf))
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "torrent file could not be parsed")
		return
	}
	log.Debugf("TorrentFile{Announce: \"%s\", Length: %d, Name: \"%s\", PieceLength: %d}", torrentFile.Announce, torrentFile.Length, torrentFile.Name, torrentFile.PieceLength)
	//TODO add tracker to trackers once it is actually parsed from torrent file

	err = file.Close()
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, "torrent file could not be closed")
		return
	}

	// construct internal torrent representation
	torrent := &storage.Torrent{
		// persisted in database
		FriendlyName:     remoteFileHdr.Filename,
		Peers:            make([]storage.Peer, 0),
		State:            storage.StateNotStartedYet,
		SeedOnCompletion: seedOnCompletionBool,
		SeedPort:         uint16(seedPortNum),
		EnableDht:        enableDhtBool,
		EnableTrackers:   enableTrackersBool,
		RawTorrentFile:   fileBuf,

		// only in-memory
		Metrics:     &storage.TorrentMetrics{},
		TorrentFile: &torrentFile,
		PeerSet:     peers2.NewPeerSet(0),
	}

	// add files to the torrent representation (potentially saving files to disk will be done later because we need torrent ID autoincrement from database)
	remoteFileHdrs := r.MultipartForm.File["files"]
	// if there are no files provided, add a dummy file so that it is displayed in UI and can be updated by leecher
	// (information comes from torrent file only)
	//TODO multiple files per torrent
	if len(remoteFileHdrs) == 0 {
		torrent.Files = append(torrent.Files, storage.File{
			Path:   torrentFile.Name,
			Length: uint64(torrentFile.Length),
		})
	}
	// otherwise, the metadata is taken from the uploaded files in the POST data
	//TODO this should probably be deferred from torrentfile metadata once multiple files are supported
	for _, remoteFileHdr = range remoteFileHdrs {
		torrent.Files = append(torrent.Files, storage.File{
			Path:     remoteFileHdr.Filename,
			Length:   uint64(remoteFileHdr.Size),
			Progress: uint64(remoteFileHdr.Size),
		})
	}

	// peer only needed when there is anything to download basically
	peers := r.Form["peers"]
	//TODO consideration of partial downloads in multiple files are supported in future
	//TODO once DHT & trackers are supported, empty peers should be made possible
	if len(peers) == 0 || len(peers[0]) == 0 {
		if len(remoteFileHdrs) == 0 {
			errorHandler(w, http.StatusBadRequest, "field \"peers\" as part of POST form data is missing (or upload all files instead)")
			return
		}
	}
	for i := 0; i < len(peers); i++ {
		peer := peers[i]

		_, err = snet.ParseUDPAddr(peer)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, fmt.Sprintf("field \"peers\" no. %d is not a valid SCION address", i))
			return
		}

		torrent.Peers = append(torrent.Peers, storage.Peer{Address: peer})

		p := peers2.Peer{
			Addr:  peer,
			Index: i,
		}
		torrent.PeerSet.Add(p)
	}

	// put it in database
	result := api.Storage.DB.Save(torrent)
	if result.Error != nil {
		errorHandler(w, http.StatusInternalServerError, "database error")
		return
	}

	// create file directory if not existing
	fileDir := torrent.GetFileDir(api.Storage.FS)
	err = os.MkdirAll(fileDir, os.ModePerm)
	if err != nil {
		errorHandler(w, http.StatusInternalServerError, "could not create file directory")
		return
	}

	// save files to disk
	errStr := ""
	for _, remoteFileHdr = range remoteFileHdrs {
		remoteFile, err := remoteFileHdr.Open()
		if err != nil {
			errStr = "one of the remote files could not be parsed"
			break
		}

		localFile, err := os.Create(filepath.Join(fileDir, remoteFileHdr.Filename))
		if err != nil {
			errStr = "one of the local files could not be created"
			break
		}

		_, err = io.Copy(localFile, remoteFile)
		if err != nil {
			errStr = "one of the files could not be copied"
			break
		}

		err = localFile.Close()
		if err != nil {
			errStr = "one of the local files could not be closed"
			break
		}

		err = remoteFile.Close()
		if err != nil {
			errorHandler(w, http.StatusInternalServerError, "one of the remote files could not be closed")
			return
		}
	}
	// delete from persistent storage again, so it does not phantom on next startup
	if len(errStr) > 0 {
		result := api.Storage.DB.Delete(torrent)
		if result.Error != nil {
			errorHandler(w, http.StatusInternalServerError, "database error")
			return
		}

		errorHandler(w, http.StatusInternalServerError, errStr)
		return
	}

	// also put it in memory
	api.torrents[torrent.ID] = torrent

	// start leecher if something needs to be downloaded
	//TODO later on, when multiple files are supported, you could still start the leecher if one or more files are still missing (partial downloads)
	if len(remoteFileHdrs) == 0 {
		ctx, cancel := context.WithCancel(context.Background())
		go api.RunLeecher(ctx, torrent)
		torrent.CancelFunc = &cancel
	} else {
		// mark torrent as finished
		//TODO when multiple files are supported, this is not necessarily true when only partial files where provided
		torrent.State = storage.StateFinishedSuccessfully

		// start seeder if requested
		if torrent.SeedOnCompletion {
			// fix number of downloaded pieces (normally a leecher process would take care of updating that field)
			torrent.P2pTorrent = &p2p.Torrent{NumDownloadedPieces: len(torrent.TorrentFile.PieceHashes)}

			ctx, cancel := context.WithCancel(context.Background())
			go api.RunSeeder(ctx, torrent)
			torrent.CancelFunc = &cancel
		}
	}

	w.WriteHeader(http.StatusCreated)
	defaultHandler(w, torrent)
}

func addTrackerHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)

	url := r.FormValue("url")
	if len(url) == 0 {
		errorHandler(w, http.StatusBadRequest, "field \"url\" as part of POST form data is missing")
		return
	}

	// construct Tracker object
	tracker := &storage.Tracker{
		// persisted in database
		URL: url,

		// only in-memory
		// ...
	}

	// put it in database
	result := api.Storage.DB.FirstOrCreate(tracker)
	if result.Error != nil {
		errorHandler(w, http.StatusInternalServerError, "database error")
		return
	}
	if result.RowsAffected == 0 {
		errorHandler(w, http.StatusConflict, "tracker with given URL already exists")
		return
	}

	// also put it in memory
	api.trackers[tracker.ID] = tracker

	w.WriteHeader(http.StatusCreated)
	defaultHandler(w, tracker)
}

func updateTorrentByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)

	torrentId, err := strconv.ParseUint(p.ByName("torrent"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	torrent, exists := api.torrents[torrentId]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		errorHandler(w, http.StatusUnsupportedMediaType, "invalid content type (\"multipart/form-data\" wanted)")
		return
	}

	action := r.FormValue("action")
	if len(action) > 0 {
		if action == "cancel" {
			if torrent.State != storage.StateRunning {
				errorHandler(w, http.StatusBadRequest, "torrent must be running to cancel it")
				return
			}

			(*torrent.CancelFunc)()
			torrent.CancelFunc = nil

			w.WriteHeader(http.StatusOK)
			defaultHandler(w, torrent)
			return
		} else if action == "retry" {
			if !torrent.State.IsFinished() {
				errorHandler(w, http.StatusBadRequest, "torrent must be finished to retry it")
				return
			}

			if torrent.CancelFunc != nil {
				(*torrent.CancelFunc)()
			}

			ctx, cancel := context.WithCancel(context.Background())
			go api.RunLeecher(ctx, torrent)
			torrent.CancelFunc = &cancel

			w.WriteHeader(http.StatusOK)
			defaultHandler(w, torrent)
			return
		}

		errorHandler(w, http.StatusBadRequest, "invalid value for field \"action\" (must be one of the following: 'cancel')")
		return
	}

	seedOnCompletionStr := r.FormValue("seedOnCompletion")
	if len(seedOnCompletionStr) > 0 {
		seedOnCompletionBool, err := strconv.ParseBool(seedOnCompletionStr)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"seedOnCompletion\" (must be 0 or 1)")
			return
		}

		// start or stop seeder if not yet done
		if seedOnCompletionBool && !torrent.SeedOnCompletion && torrent.State == storage.StateFinishedSuccessfully {
			ctx, cancel := context.WithCancel(context.Background())
			go api.RunSeeder(ctx, torrent)
			torrent.CancelFunc = &cancel
		} else if !seedOnCompletionBool && torrent.State == storage.StateSeeding && torrent.CancelFunc != nil {
			(*torrent.CancelFunc)()
			torrent.CancelFunc = nil
		}

		torrent.SeedOnCompletion = seedOnCompletionBool
	}

	// save changes to database
	result := api.Storage.DB.Save(torrent)
	if result.Error != nil {
		errorHandler(w, http.StatusInternalServerError, "database error")
		return
	}

	w.WriteHeader(http.StatusOK)
	defaultHandler(w, torrent)
}

func deleteTorrentByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("torrent"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	deleteFromFs := false
	param := r.URL.Query().Get("deleteFiles")
	if len(param) > 0 {
		deleteFromFs, err = strconv.ParseBool(param)
		if err != nil {
			errorHandler(w, http.StatusBadRequest, "invalid value for field \"deleteFiles\" (must be 0 or 1)")
			return
		}
	}

	torrent, exists := api.torrents[id]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}

	if !torrent.State.IsFinished() {
		errorHandler(w, http.StatusConflict, "torrent is running, stop it before deletion")
		return
	}

	// stop seeder if not yet done
	if torrent.CancelFunc != nil {
		(*torrent.CancelFunc)()
		torrent.CancelFunc = nil
	}

	// delete files associated to the torrent
	if deleteFromFs {
		err := os.RemoveAll(torrent.GetFileDir(api.Storage.FS))
		if err != nil {
			log.Error(err)
			errorHandler(w, http.StatusInternalServerError, "could not delete files associated with torrent")
			return
		}
	}

	// delete torrent from memory
	delete(api.torrents, id)

	// delete torrent from database
	api.Storage.DB.Delete(torrent)

	defaultHandler(w, nil)
}

func deleteTrackerByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("tracker"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	tracker, exists := api.trackers[id]
	if !exists {
		errorHandler(w, http.StatusNotFound, "tracker with given ID not found")
		return
	}

	// delete torrent from memory
	delete(api.trackers, id)

	// delete torrent from database
	api.Storage.DB.Delete(tracker)

	defaultHandler(w, nil)
}

func defaultHandler(w http.ResponseWriter, payload interface{}) {
	str, err := json.Marshal(&payload)
	if err != nil {
		log.Error(err)
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = w.Write(str)
	if err != nil {
		log.Error(err)
		errorHandler(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func errorHandler(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	body := &ErrorResponseBody{Error: message}

	str, err := json.Marshal(&body)
	if err != nil {
		log.Error(err)
		return
	}

	_, err = w.Write(str)
	if err != nil {
		log.Error(err)
		return
	}
}

// https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
// open opens the specified URL in the default browser of the user
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func (api *HttpApi) LoadFromStorage() error {
	api.torrents = make(map[uint64]*storage.Torrent)
	api.trackers = make(map[uint64]*storage.Tracker)
	return nil
}

//go:embed frontend/dist/*
var static embed.FS

func (api *HttpApi) ListenAndServe() error {
	api.usedUdpPorts = make(map[uint16]bool)

	// resolve local SCION address
	var err error
	if api.ScionLocalHost != "" {
		api.localAddr, err = snet.ParseUDPAddr(api.ScionLocalHost)
	} else {
		api.localAddr, err = util.GetDefaultLocalAddr() //TODO currently wastes a port
	}
	if err != nil {
		log.Error("[HTTP API] Could not parse local SCION address (specified with -local argument)!")
		return err
	}

	frontend, err := fs.Sub(static, "frontend/dist")
	if err != nil {
		log.Error("[HTTP API] Could not load static assets!")
		return err
	}

	router := httprouter.New()
	router.GET("/api/info", getInfoHandler)
	router.GET("/api/torrent", listTorrentsHandler)
	router.GET("/api/torrent/:torrent", getTorrentByIdHandler)
	router.GET("/api/torrent/:torrent/file/:file", getFileByIdHandler)
	router.GET("/api/tracker", listTrackersHandler)
	router.GET("/api/tracker/:tracker", getTrackerByIdHandler)
	router.POST("/api/torrent", addTorrentHandler)
	router.POST("/api/torrent/:torrent", updateTorrentByIdHandler)
	router.POST("/api/tracker", addTrackerHandler)
	router.DELETE("/api/torrent/:torrent", deleteTorrentByIdHandler)
	router.DELETE("/api/tracker/:tracker", deleteTrackerByIdHandler)
	router.ServeFiles("/frontend/*filepath", http.FS(frontend))

	server := &http.Server{
		Addr: api.LocalAddr,
		Handler: cors.New(cors.Options{
			AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		}).Handler(router),
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(context.Background(), "api", api)
		},
	}

	frontendAddr := fmt.Sprintf("http://%s/frontend", api.LocalAddr)
	if strings.Contains(frontendAddr, "0.0.0.0") {
		frontendAddr = strings.Replace(frontendAddr, "0.0.0.0", "127.0.0.1", -1)
	} else if strings.Contains(frontendAddr, "[::]") {
		frontendAddr = strings.Replace(frontendAddr, "[::]", "[::1]", -1)
	}
	log.Infof("[HTTP API] Listening on %s, frontend available at: %s", api.LocalAddr, frontendAddr)
	err = open(frontendAddr)
	if err == nil {
		log.Infof("[HTTP API] Opened frontend in your browser!")
	}

	return server.ListenAndServe()
}
