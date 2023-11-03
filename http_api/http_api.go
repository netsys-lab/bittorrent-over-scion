package http_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/netsys-lab/bittorrent-over-scion/http_api/storage"
	"github.com/netsys-lab/bittorrent-over-scion/torrentfile"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

type HttpApi struct {
	Port              int
	LocalHost         string
	NumPaths          int
	DialBackStartPort uint16
	SeedStartPort     uint16
	EnableDht         bool
	DhtPort           uint16
	DhtBootstrapAddr  string

	Storage *storage.Storage

	torrents     map[uint64]*storage.Torrent
	trackers     map[uint64]*storage.Tracker
	usedUdpPorts map[uint16]bool
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
	defaultHandler(w, api.torrents)
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

	// limit request body (torrent file) to 10 MByte
	r.Body = http.MaxBytesReader(w, r.Body, 10000000)

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		errorHandler(w, http.StatusUnsupportedMediaType, "invalid content type (\"multipart/form-data\" wanted)")
		return
	}

	peer := r.FormValue("peer")
	if len(peer) == 0 {
		errorHandler(w, http.StatusBadRequest, "field \"peer\" as part of POST form data is missing")
		return
	}

	var err error
	seedOnCompletionStr := r.FormValue("seedOnCompletion")
	seedOnCompletionBool := false
	if len(peer) > 0 {
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

	file, fileHdr, err := r.FormFile("torrentFile")
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "file field \"torrentFile\" as part of POST form data is missing")
		return
	}

	// read torrent file into byte slice
	fileBuf := make([]byte, fileHdr.Size)
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

	// construct Torrent object
	torrent := &storage.Torrent{
		// persisted in database
		FriendlyName:     fileHdr.Filename,
		State:            storage.StateNotStartedYet,
		Peer:             peer,
		SeedOnCompletion: seedOnCompletionBool,
		SeedPort:         uint16(seedPortNum),
		//TODO multiple files per torrent
		Files: []storage.File{
			{
				Path:   torrentFile.Name,
				Length: uint64(torrentFile.Length),
			},
		},
		RawTorrentFile: fileBuf,

		// only in-memory
		Metrics:     &storage.TorrentMetrics{},
		TorrentFile: &torrentFile,
	}

	// put it in database
	result := api.Storage.DB.Save(torrent)
	if result.Error != nil {
		errorHandler(w, http.StatusInternalServerError, "database error")
		return
	}

	// also put it in memory
	api.torrents[torrent.ID] = torrent

	// start torrent
	ctx, cancel := context.WithCancel(context.Background())
	go api.RunLeecher(ctx, torrent)
	torrent.CancelFunc = &cancel
	//TODO make cancellation actually possible

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
	if torrent.State == storage.StateSeeding && torrent.CancelFunc != nil {
		(*torrent.CancelFunc)()
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

func (api *HttpApi) LoadFromStorage() error {
	api.torrents = make(map[uint64]*storage.Torrent)
	api.trackers = make(map[uint64]*storage.Tracker)
	return nil
}

func (api *HttpApi) ListenAndServe() error {
	api.usedUdpPorts = make(map[uint16]bool)

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
	router.ServeFiles("/frontend/*filepath", AssetFile())

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", api.Port),
		Handler: cors.New(cors.Options{
			AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		}).Handler(router),
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(context.Background(), "api", api)
		},
	}

	log.Infof("[HTTP API] Listening on 0.0.0.0:%d, frontend available at: http://localhost:%d/frontend", api.Port, api.Port)
	return server.ListenAndServe()
}
