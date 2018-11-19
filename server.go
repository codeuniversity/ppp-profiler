package profiler

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codeuniversity/ppp-mhist"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

var dbPath = "data"

var profileBucketName = []byte("profiles")

//Server that listens for new events and serves profiles
type Server struct {
	Address         string
	incomingChannel chan []byte
	subscriber      *mhist.TCPSubscriber
	conns           []*websocket.Conn
	profiles        []*Profile
	connLock        *sync.RWMutex
	profileLock     *sync.RWMutex
	db              *bolt.DB
}

//NewServer returns a server ready for usage
func NewServer(address string) *Server {
	incomingChannel := make(chan []byte)

	os.MkdirAll(dbPath, os.ModePerm)

	db, err := bolt.Open(filepath.Join(dbPath, "profile.db"), os.ModePerm, nil)
	if err != nil {
		fmt.Println("failed to open db")
		panic(err)
	}
	tx, err := db.Begin(true)
	if err != nil {
		panic(err)
	}

	_, err = tx.CreateBucketIfNotExists(profileBucketName)

	if err != nil {
		panic(err)
	}
	err = tx.Commit()

	if err != nil {
		panic(err)
	}

	s := &Server{
		Address:         address,
		incomingChannel: incomingChannel,
		subscriber:      mhist.NewTCPSubscriber(address, mhist.FilterDefinition{}, incomingChannel),
		connLock:        &sync.RWMutex{},
		profileLock:     &sync.RWMutex{},
		db:              db,
	}

	s.readProfilesIntoMemory()
	go s.keepUpdatingDB()
	return s
}

//Connect to server and retry to establish connections
func (s *Server) Connect() {
	s.subscriber.Connect()
	go s.keepReading()
	go s.Listen()
}

//Listen to incoming http requests to be upgraded to websocket connections
func (s *Server) Listen() {
	r := mux.NewRouter()
	r.HandleFunc("/profiles/{id}", s.profileHandler)
	r.HandleFunc("/profiles", s.profileHandler)
	r.HandleFunc("/", s.websocketHandler)
	http.Handle("/", r)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE"},
	})

	http.ListenAndServe(":4000", c.Handler(r))
}

//Run the server and listen for messages
func (s *Server) Run() {
	s.Connect()

	for byteSlice := range s.incomingChannel {
		message := &mhist.Message{}
		err := json.Unmarshal(byteSlice, message)
		if err != nil {
			fmt.Println(err)
			continue
		}
		s.forEachProfile(func(profile *Profile) {
			profile.Eval(message)
			value := profile.Value()
			message := profileMessage{
				ID: value.ID,
				Data: profileMessageContent{
					Type:  "update",
					State: value.Data,
				},
			}
			s.broadcast(message)
		})
	}
}

func (s *Server) keepUpdatingDB() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.updateProfilesOnDisk()
		}

	}
}

func (s *Server) readProfilesIntoMemory() {
	s.profileLock.Lock()
	defer s.profileLock.Unlock()
	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(profileBucketName)
		bucket.ForEach(func(_, value []byte) error {
			profile := &Profile{}
			err := json.Unmarshal(value, profile)
			if err != nil {
				return nil
			}
			s.profiles = append(s.profiles, profile)
			return nil
		})
		return nil
	})
}

func (s *Server) updateProfilesOnDisk() {
	s.profileLock.RLock()
	defer s.profileLock.RUnlock()

	s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(profileBucketName)
		for _, profile := range s.profiles {
			byteSlice, err := json.Marshal(profile)
			if err != nil {
				fmt.Println(err)
				continue
			}
			bucket.Put([]byte(profile.Definition.ID), byteSlice)

		}
		return nil
	})
}

func (s *Server) keepReading() {
	for {
		err := s.subscriber.Read()
		fmt.Println(err)
		s.subscriber.Connect()
	}
}

func (s *Server) forEachProfile(f func(p *Profile)) {
	s.profileLock.RLock()
	defer s.profileLock.RUnlock()

	for _, profile := range s.profiles {
		f(profile)
	}
}

type profileMessage struct {
	ID   string                `json:"id"`
	Data profileMessageContent `json:"data"`
}

type profileMessageContent struct {
	Type  string                 `json:"type"`
	State map[string]interface{} `json:"state"`
}

func (s *Server) broadcast(m profileMessage) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	indicesToRemove := []int{}

	for index, conn := range s.conns {
		err := conn.WriteJSON(m)
		if err != nil {
			//assume connection is dead
			fmt.Println(err)
			indicesToRemove = append(indicesToRemove, index)
		}
	}

	if len(indicesToRemove) == 0 {
		return
	}

	newSlice := []*websocket.Conn{}
	for index, conn := range s.conns {
		if !isIncluded(index, indicesToRemove) {
			newSlice = append(newSlice, conn)
		}
	}
	s.conns = newSlice
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func (s *Server) profileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleProfilesPost(w, r)
	case http.MethodGet:
		s.handleProfilesGet(w, r)
	case http.MethodDelete:
		s.handleProfileDelete(w, r)
	}
}

func (s *Server) handleProfilesGet(w http.ResponseWriter, r *http.Request) {
	s.profileLock.RLock()
	defer s.profileLock.RUnlock()

	byteSlice, err := json.Marshal(s.profiles)
	if err != nil {
		renderError(err, w, http.StatusInternalServerError)
		return
	}

	w.Write(byteSlice)
}

func (s *Server) handleProfilesPost(w http.ResponseWriter, r *http.Request) {
	byteSlice, err := ioutil.ReadAll(r.Body)
	if err != nil {
		renderError(err, w, http.StatusBadRequest)
		return
	}

	definition := &ProfileDefinition{}
	err = json.Unmarshal(byteSlice, definition)
	if err != nil {
		renderError(err, w, http.StatusBadRequest)
		return
	}

	if definition.ID == "" {
		renderError(errors.New("id has to be set"), w, http.StatusBadRequest)
	}

	var profile *Profile
	s.profileLock.Lock()
	defer s.profileLock.Unlock()

	err = s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(profileBucketName)
		profile = NewProfile(*definition)
		byteSlice, err := json.Marshal(profile)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(definition.ID), byteSlice)
	})

	if err != nil {
		renderError(err, w, http.StatusInternalServerError)
		return
	}
	s.profiles = append(s.profiles, profile)

	answer, err := json.Marshal(definition)
	if err != nil {
		renderError(err, w, http.StatusInternalServerError)
		return
	}
	w.Write(answer)
}

func (s *Server) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	idToBeDeleted := mux.Vars(r)["id"]
	if idToBeDeleted == "" {
		err := errors.New("you have to specify a profile id to delete")
		renderError(err, w, http.StatusBadRequest)
		return
	}

	s.profileLock.Lock()
	defer s.profileLock.Unlock()

	newProfileSlice := []*Profile{}

	for _, profile := range s.profiles {
		if profile.Definition.ID != idToBeDeleted {
			newProfileSlice = append(newProfileSlice, profile)
		}
	}
	s.profiles = newProfileSlice

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(profileBucketName)
		return bucket.Delete([]byte(idToBeDeleted))
	})

	if err != nil {
		renderError(err, w, http.StatusInternalServerError)
		return
	}
	deleteMessage := profileMessage{
		ID: idToBeDeleted,
		Data: profileMessageContent{
			Type: "delete",
		},
	}
	s.broadcast(deleteMessage)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	s.connLock.Lock()
	defer s.connLock.Unlock()
	s.conns = append(s.conns, conn)

	s.forEachProfile(func(profile *Profile) {
		value := profile.Value()
		message := profileMessage{
			ID: value.ID,
			Data: profileMessageContent{
				Type:  "update",
				State: value.Data,
			},
		}
		conn.WriteJSON(message)
	})
}

type errorResponse struct {
	Error string `json:"error"`
}

func renderError(err error, w http.ResponseWriter, status int) {
	fmt.Println(err)
	resp := &errorResponse{Error: err.Error()}
	data, err := json.Marshal(resp)
	if err == nil {
		w.WriteHeader(status)
		w.Write(data)
	}
}

func isIncluded(element int, arr []int) bool {
	for _, arrayElement := range arr {
		if arrayElement == element {
			return true
		}
	}
	return false
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
