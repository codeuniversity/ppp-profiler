package profiler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/codeuniversity/ppp-mhist"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

//Server that listens for new events and serves profiles
type Server struct {
	Address         string
	incomingChannel chan []byte
	subscriber      *mhist.TCPSubscriber
	conns           []*websocket.Conn
	profiles        []*Profile
	connLock        *sync.RWMutex
	profileLock     *sync.RWMutex
}

//NewServer returns a server ready for usage
func NewServer(address string) *Server {
	incomingChannel := make(chan []byte)
	return &Server{
		Address:         address,
		incomingChannel: incomingChannel,
		subscriber:      mhist.NewTCPSubscriber(address, mhist.FilterDefinition{}, incomingChannel),
		connLock:        &sync.RWMutex{},
		profileLock:     &sync.RWMutex{},
	}
}

//Connect to server and retry to establish connections
func (s *Server) Connect() {
	s.subscriber.Connect()
	go s.keepReading()
	go s.Listen()
}

//Listen to incoming http requests to be upgraded to websocket connections
func (s *Server) Listen() {
	http.HandleFunc("/profiles", s.profileHandler)
	http.HandleFunc("/", s.websocketHandler)
	http.ListenAndServe(":4000", nil)
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
			s.broadcast(profile.Value())
		})
	}
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

func (s *Server) broadcast(d ProfileDisplayValue) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	indicesToRemove := []int{}

	for index, conn := range s.conns {
		err := conn.WriteJSON(d)
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
	id := uuid.NewV4()
	definition.ID = id.String()
	profile := NewProfile(*definition)
	s.profileLock.Lock()
	defer s.profileLock.Unlock()

	s.profiles = append(s.profiles, profile)

	answer, err := json.Marshal(definition)
	if err != nil {
		renderError(err, w, http.StatusInternalServerError)
		return
	}
	w.Write(answer)
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
		d := profile.Value()
		conn.WriteJSON(d)
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
