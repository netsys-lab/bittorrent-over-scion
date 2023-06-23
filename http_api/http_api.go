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
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type HttpApi struct {
	Port      int
	EnableDht bool
	Storage   *storage.Storage
	LocalHost string
	torrents  map[uint]*storage.Torrent
}

type ErrorResponseBody struct {
	Error string `json:"error"`
}

func getInfoHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func getTorrentByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("id"), 10, 0)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, "invalid ID specified")
		return
	}

	torrent, exists := api.torrents[uint(id)]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}
	defaultHandler(w, torrent)
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

	// construct Torrent object
	torrent := &storage.Torrent{
		// part of database
		FriendlyName: fileHdr.Filename,
		State:        storage.StateNotStartedYet,
		Peer:         peer,
		Files: []storage.File{
			{
				Path: torrentFile.Name,
			},
		},

		// only in-memory
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
	go api.RunTorrent(ctx, torrent)
	torrent.CancelFunc = &cancel
	//TODO make cancellation actually possible

	w.WriteHeader(http.StatusCreated)
	defaultHandler(w, torrent)
}

func deleteTorrentByIdHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	api := r.Context().Value("api").(*HttpApi)
	id, err := strconv.ParseUint(p.ByName("id"), 10, 0)
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
		}
	}

	torrent, exists := api.torrents[uint(id)]
	if !exists {
		errorHandler(w, http.StatusNotFound, "torrent with given ID not found")
		return
	}

	if !torrent.State.IsFinished() {
		errorHandler(w, http.StatusConflict, "torrent is running, stop it before deletion")
		return
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
	delete(api.torrents, uint(id))

	// delete torrent from database
	api.Storage.DB.Delete(torrent)

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
	api.torrents = make(map[uint]*storage.Torrent)
	return nil
}

func (api *HttpApi) ListenAndServe() error {
	router := httprouter.New()
	router.GET("/api/info", getInfoHandler)
	router.GET("/api/torrent", listTorrentsHandler)
	router.GET("/api/torrent/:id", getTorrentByIdHandler)
	router.POST("/api/torrent", addTorrentHandler)
	router.DELETE("/api/torrent/:id", deleteTorrentByIdHandler)
	router.ServeFiles("/frontend/*filepath", AssetFile())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", api.Port),
		Handler: router,
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(context.Background(), "api", api)
		},
	}

	log.Infof("[HTTP API] Listening on 0.0.0.0:%d, frontend available at: http://localhost:%d/frontend", api.Port, api.Port)
	return server.ListenAndServe()
}
