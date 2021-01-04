package chatroom

import (
"github.com/gin-gonic/gin"
"github.com/gorilla/websocket"
"net/http"
"sync"
"time"
)

type WebSocketPool struct {
	Set map[string]*WebSocketSession
}

type WebSocketSession struct {
	// Id           string
	Conn         *websocket.Conn
	SendChan     chan []byte
	RecvChan     chan []byte
	CloseChan    chan struct{}
	ListenTicker *time.Ticker
	Mutex        *sync.RWMutex
}

func NewWS(c *gin.Context, interval time.Duration) (*WebSocketSession, error) {
	var upGrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}

	return &WebSocketSession{
		Conn:         conn,
		RecvChan:     make(chan []byte),
		CloseChan:    make(chan struct{}),
		ListenTicker: time.NewTicker(interval),
		Mutex:        &sync.RWMutex{},
	}, nil
}

func (pool *WebSocketPool) Add(id string, conn *WebSocketSession) {
	pool.Set[id] = conn
}

func (pool *WebSocketPool) Del(id string) {
	delete(pool.Set, id)
}

// no share args or returns
func (s *WebSocketSession) MaintainHeartbeat() {
	for {
		select {
		case <-s.RecvChan:
			s.ListenTicker.Reset(20 * time.Second)
		case <-s.ListenTicker.C:
			s.CloseChan <- struct{}{}
		}
	}
}

func (s *WebSocketSession) ReceiveMsg() {
	// var msg string
	for {
		messageType, data, _ := s.Conn.ReadMessage()
		if messageType == -1 {
			// closeChan <- struct{}{}
			return
		}
		// msg = string(data)
		s.RecvChan <- data
	}
}

func (s *WebSocketSession) SendMsg() {
	select {
	case <-s.SendChan:
		err := s.Conn.WriteMessage(1, <-s.SendChan)
		if err != nil {
			return
		}
	case <-s.CloseChan:
		return
	}
}

func (s *WebSocketSession) Close() {
	s.Conn.Close()
	s.ListenTicker.Stop()
	close(s.CloseChan)
	close(s.RecvChan)
}

