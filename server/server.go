package server

import (
	"net"
	"errors"
	"io"
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
type Server struct{
	s_lis *net.TCPListener
	s_out chan *message
	s_conMap map[int]*net.TCPConn
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
	s.s_conMap = make(map[int]*net.TCPConn)
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
func writeHeadBuf(lens uint32)[]byte{
	buf := make([]byte,4);
	buf[0] = byte(lens>>24 	& 0x000000ff);
	buf[1] = byte(lens>>16 	& 0x000000ff);
	buf[2] = byte(lens>>8  	& 0x000000ff);
	buf[3] = byte(lens 		& 0x000000ff);
	return buf;
}
func readHeadBuf(buf[]byte)uint32{
	return uint32(buf[0])<<24|uint32(buf[1])<<16|uint32(buf[2])<<8|uint32(buf[3]);
}
func (p * Server) SendToAll(data []byte)error{
	buf := writeHeadBuf(uint32(len(data)));
	for _,con := range(p.s_conMap) {
		_ ,err := con.Write(buf);
		if(err != nil){
			return err;
		}
		_ ,err = con.Write(data);
		if(err != nil){
			return err;
		}
	}
	return nil;
}
func (p * Server) SendToId( id int,data []byte) error{
	con := p.s_conMap[id];
	if(con != nil){
		buf := writeHeadBuf(uint32(len(data)));
		_ ,err := con.Write(buf);
		if(err != nil){
			return err;
		}
		_ ,err = con.Write(data);
		return err;
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
		p.s_conMap[id] = con;
		p.wg.Add(1)
		go p.connLoop(id,con)
		p.s_out <- &message{Id:id,Type:type_connect};
	}
}
func (p * Server) connLoop( id int ,con * net.TCPConn){
	defer func(){
			con.Close()
			p.wg.Done();
		}()
	var head uint32;
	var err error;
	var ret []byte = make([]byte,4);
	for !p.s_stop {
		_,err = io.ReadFull(con,ret);
		if err != nil {
			break;
		}
		//BIG E
		head = readHeadBuf(ret);
		buf := make([]byte,head);
		_,err = io.ReadFull(con,buf);
		if err != nil {
			break;
		}
		p.s_out <- &message{Id:id,Type:type_message,Data:buf};
	}
	p.s_out <- &message{Id:id,Type:type_close,Err:err};
	return
}