package profiler

import (
	"encoding/json"
	"fmt"
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
	average := &runningAverage{}
	for byteSlice := range s.incomingChannel {
		message := &mhist.Message{}
		err := json.Unmarshal(byteSlice, message)
		if err != nil {
			fmt.Println(err)
			continue
		}
		latestValue, ok := message.Value.(float64)
		if ok {
			average.Add(latestValue)
			s.broadcast(data{Average: average.Value, Current: latestValue})
		}
	}
}

func (s *Server) keepReading() {
	for {
		err := s.subscriber.Read()
		fmt.Println(err)
		s.subscriber.Connect()
	}
}

type data struct {
	Average float64 `json:"average"`
	Current float64 `json:"current"`
}

func (s *Server) broadcast(d data) {
	s.RLock()
	defer s.RUnlock()
	for _, conn := range s.conns {
		err := conn.WriteJSON(d)
		if err != nil {
			fmt.Println(err)
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
