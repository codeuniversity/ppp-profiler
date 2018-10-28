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
)

//Server that listens for new events and serves profiles
type Server struct {
	Address         string
	incomingChannel chan []byte
	subscriber      *mhist.TCPSubscriber
	conns           []*websocket.Conn
	sync.RWMutex
}

//NewServer returns a server ready for usage
func NewServer(address string) *Server {
	incomingChannel := make(chan []byte)
	return &Server{
		Address:         address,
		incomingChannel: incomingChannel,
		subscriber:      mhist.NewTCPSubscriber(address, mhist.FilterDefinition{}, incomingChannel),
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
	http.HandleFunc("/", s.websocketHandler)
	http.ListenAndServe(":4000", nil)
}

//Run the server and listen for messages
func (s *Server) Run() {
	s.Connect()
	runningAverageScript := `
		var average = get("average", 0)
		var count = get("count", 0)
		average = ((average * count) + message.value) / (count+1)
		count++
		set("average", average)
		set("current", message.value)
		set("count", count)

		display("average", average)
		display("current", message.value)
	`
	profile := NewProfile(ProfileDefinition{EvalScript: runningAverageScript})
	for byteSlice := range s.incomingChannel {
		message := &mhist.Message{}
		err := json.Unmarshal(byteSlice, message)
		if err != nil {
			fmt.Println(err)
			continue
		}
		profile.Eval(message)
		s.broadcast(profile.Value())
	}
}

func (s *Server) keepReading() {
	for {
		err := s.subscriber.Read()
		fmt.Println(err)
		s.subscriber.Connect()
	}
}

func (s *Server) broadcast(d map[string]interface{}) {
	s.RLock()
	defer s.RUnlock()
	for index, conn := range s.conns {
		err := conn.WriteJSON(d)
		if err != nil {
			//assume connection is dead
			fmt.Println(err)
			newSlice := make([]*websocket.Conn, 0)
			newSlice = append(newSlice, s.conns[:index]...)
			if index+1 < len(s.conns) {
				newSlice = append(newSlice, s.conns[index+1:]...)
			}

			s.conns = newSlice
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	s.Lock()
	defer s.Unlock()
	s.conns = append(s.conns, conn)
}
