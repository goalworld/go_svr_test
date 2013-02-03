package server

import (
	"errors"
	"net"
	"sync"
)

const (
	Type_connect int = iota
	Type_message
	Type_close
)

type Message struct {
	Id   int
	Type int
	Data []byte
	Err  error
}
type sendCmd struct {
	Id   int
	Data []byte
}
type Server struct {
	s_lis     net.Listener
	s_out     chan *Message
	s_send    chan *sendCmd
	s_conMap  map[int]ConnCoder
	s_idIndex int
	s_stop    chan bool
	rwmtx     sync.RWMutex
	wg        sync.WaitGroup
}

func NewServer(addr string, out chan *Message) (s *Server, err error) {
	s = new(Server)
	s.s_lis, err = net.Listen("tcp", addr)
	if err != nil {
		s = nil
		return
	}
	s.s_out = out
	s.s_send = make(chan *sendCmd, 1024)
	s.s_conMap = make(map[int]ConnCoder)
	s.s_idIndex = 0
	s.s_stop = make(chan bool)
	return
}
func (p *Server) ConnectionNum() int {
	return len(p.s_conMap)
}
func (p *Server) Run() {
	p.wg.Add(2)
	go p.acceptLoop()
	go p.sendLoop()
}
func (p *Server) Stop() {
	p.s_lis.Close()
	p.s_stop <- false
	p.s_stop <- false
	p.wg.Wait()
}
func (p *Server) Close() {
	defer func() {
		recover()
	}()
	p.s_lis.Close()
	for _, con := range p.s_conMap {
		con.Close()
	}
	close(p.s_out)
	close(p.s_send)
}
func (p *Server) Broadcast(data []byte) {
	p.s_send <- &sendCmd{-1, data}
}
func (p *Server) Send(id int, data []byte) error {
	if con := p.getConn(id); con == nil {
		return errors.New("conn not exist")
	}
	p.s_send <- &sendCmd{id, data}
	return nil
}
func (p *Server) sendLoop() {
	defer func() {
		p.wg.Done()
	}()
	for {
		select {
		case cmd := <-p.s_send:
			if cmd.Id == -1 {
				p.rwmtx.RLock()
				arr := make([]ConnCoder, len(p.s_conMap))
				i := 0
				for _, con := range p.s_conMap {
					arr[i] = con
					i++
				}
				p.rwmtx.RUnlock()
				for _, con := range arr {
					con.WriteMessage(cmd.Data)
				}
			} else if con := p.getConn(cmd.Id); con != nil {
				con.WriteMessage(cmd.Data)
			}
		case <-p.s_stop:
			return
		}
	}
}
func (p *Server) acceptLoop() {
	defer func() {
		p.wg.Done()
	}()
	for {
		select {
		case <-p.s_stop:
			return
		default:
			con, err := p.s_lis.Accept()
			if err != nil {
				break
			}
			id := p.s_idIndex
			p.s_idIndex++
			conc := newHeadConnCoder(con)
			p.addConn(id, conc)
			p.wg.Add(1)
			go p.connLoop(id, conc)
			p.s_out <- &Message{Id: id, Type: Type_connect}
		}
	}
}
func (p *Server) connLoop(id int, con ConnCoder) {
	defer func() {
		p.wg.Done()
	}()
	for {
		buf, err := con.ReadMessage()
		if err != nil {
			p.delConn(id)
			p.s_out <- &Message{Id: id, Type: Type_close, Err: err}
			return
		}
		p.s_out <- &Message{Id: id, Type: Type_message, Data: buf}
	}
}
func (p *Server) addConn(id int, con ConnCoder) {
	defer p.rwmtx.Unlock()
	p.rwmtx.Lock()
	p.s_conMap[id] = con
}
func (p *Server) delConn(id int) {
	defer p.rwmtx.Unlock()
	p.rwmtx.Lock()
	delete(p.s_conMap, id)
}
func (p *Server) getConn(id int) ConnCoder {
	defer p.rwmtx.RUnlock()
	p.rwmtx.RLock()
	con, _ := p.s_conMap[id]
	return con
}
