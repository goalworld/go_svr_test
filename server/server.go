package server

import (
	"net"
	"errors"
	"sync"
)
const(
	Type_connect int = iota
	Type_message 
	Type_close 
)
type Message struct{
	Id int
	Type int
	Data []byte
	Err error
}

type Server struct{
	s_lis net.Listener
	s_out chan *Message
	s_conMap map[int]ConnCoder
	s_idIndex int
	s_stop bool
	mtx sync.Mutex
	wg sync.WaitGroup
}
func NewServer(addr string , out chan *Message) (s *Server ,err error){
	s = new(Server)
	s.s_lis ,err = net.Listen("tcp",addr)
	if(err != nil){
		s = nil
		return
	}
	s.s_out = out
	s.s_conMap = make(map[int]ConnCoder)
	s.s_idIndex = 0;
	s.s_stop = false;
	return
}
func (p *Server) ConnectionNum() int{
	return len(p.s_conMap)
}
func (p *Server) Run(){
	p.wg.Add(1);
	go p.acceptLoop()
}
func (p *Server) Wait(){
	p.wg.Wait();
}
func (p *Server) Close()  {
	p.s_lis.Close();
	for _,con := range(p.s_conMap) {
		con.Close();
	}
	close(p.s_out);
}
func (p * Server) Broadcast(data []byte)error{
	for _,con := range(p.s_conMap) {
		con.WriteMessage(data);
	}
	return nil;
} 
func (p * Server) Send( id int,data []byte) error{
	con := p.s_conMap[id];
	if(con != nil){
		return con.WriteMessage(data);
	}
	return errors.New("conn not exist")
}
func (p *Server) acceptLoop(){
	defer func(){
			p.wg.Done();
		}()
	for !p.s_stop {
		con , err := p.s_lis.Accept()
		if(err != nil){
			break
		}
		id := p.s_idIndex;
		p.s_idIndex++;
		conc := newHeadConnCoder(con);
		p.s_conMap[id] = conc;
		p.wg.Add(1)
		go p.connLoop(id,conc)
		p.s_out <- &Message{Id:id,Type:Type_connect};
	}
}
func (p * Server) connLoop( id int ,con  ConnCoder){
	defer func(){
			p.wg.Done();
		}()
	for !p.s_stop {
		buf,err := con.ReadMessage();
		if err != nil {
			p.delConn(id)
			p.s_out <- &Message{Id:id,Type:Type_close,Err:err};
			return
		}
		p.s_out <- &Message{Id:id,Type:Type_message,Data:buf};
	}
}
func (p *Server) delConn(id int){
	defer p.mtx.Unlock()
	p.mtx.Lock()
	delete(p.s_conMap,id)
}