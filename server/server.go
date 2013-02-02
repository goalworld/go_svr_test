package server

import (
	"net"
	"errors"
	"sync"
)
const(
	type_connect int = iota
	type_message 
	type_close 
)
type message struct{
	Id int
	Type int
	Data []byte
	Err error
}
type ServerListener interface{
	OnConnect(id int);
	OnMessage(id int,data []byte);
	OnClose(id int,err error);
}
type ConnCoder interface{
	WriteMessage(buf []byte) error
	ReadMessage()([]byte, error)
	Close()error
}
type Server struct{
	s_lis *net.TCPListener
	s_out chan *message
	s_conMap map[int]ConnCoder
	s_idIndex int
	s_stop bool
	s_ober ServerListener
	wg sync.WaitGroup
}
func NewServer(ip string,port int) (s *Server ,err error){
	s = new(Server)
	addr := net.TCPAddr{net.ParseIP(ip),port};
	s.s_lis ,err = net.ListenTCP("tcp",&addr)
	if(err != nil){
		s = nil
		return
	}
	s.s_out = make(chan *message,1024)
	s.s_conMap = make(map[int]ConnCoder)
	s.s_idIndex = 0;
	s.s_stop = false;
	return
}
func (p *Server) ConnectionNum() int{
	return len(p.s_conMap)
}
func (p *Server) SetListener(lis ServerListener){
	p.s_ober  = lis;
}
func (p *Server) Run(){
	defer func(){
			p.s_lis.Close();
			for _,con := range(p.s_conMap) {
				con.Close();
			}
			p.wg.Wait();
			close(p.s_out);
		}()
	p.wg.Add(1);
	go p.acceptLoop()
	for !p.s_stop {
		select{
			case msg := <- p.s_out:
				switch msg.Type{
				case type_connect:
					p.s_ober.OnConnect(msg.Id)
				case type_message:
					p.s_ober.OnMessage(msg.Id,msg.Data)
				case type_close:
					delete(p.s_conMap,msg.Id)
					p.s_ober.OnClose(msg.Id,msg.Err)
				default:
					println("UNKONW MESSAGE",msg)
				}
		}
	}
	
}
func (p *Server) Stop() {
	if(p.s_stop){
		return
	}
	p.s_stop = true;
	
}
func (p * Server) SendToAll(data []byte)error{
	for _,con := range(p.s_conMap) {
		con.WriteMessage(data);
	}
	return nil;
} 
func (p * Server) SendToId( id int,data []byte) error{
	con := p.s_conMap[id];
	if(con != nil){
		return con.WriteMessage(data);
	}
	return errors.New("conn not exist")
}
func (p *Server) acceptLoop(){
	defer func(){
			p.s_lis.Close()
			p.wg.Done();
		}()
	for !p.s_stop {
		con , err := p.s_lis.AcceptTCP()
		if(err != nil){
			break
		}
		id := p.s_idIndex;
		p.s_idIndex++;
		conc := newHeadConnCoder(con);
		p.s_conMap[id] = conc;
		p.wg.Add(1)
		go p.connLoop(id,conc)
		p.s_out <- &message{Id:id,Type:type_connect};
	}
}
func (p * Server) connLoop( id int ,con  ConnCoder){
	defer func(){
			con.Close()
			p.wg.Done();
		}()
	for !p.s_stop {
		buf,err := con.ReadMessage();
		if err != nil {
			p.s_out <- &message{Id:id,Type:type_close,Err:err};
			return
		}
		p.s_out <- &message{Id:id,Type:type_message,Data:buf};
	}
}