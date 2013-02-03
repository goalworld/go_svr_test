package server
import "fmt"
import "testing"
type Lis struct{
	svr *Server
}
func ( p *Lis)OnConnect(id int){
	println("OnConnect",id);
}
func (p * Lis)OnMessage(id int,data []byte){
	println("OnMessage",id,string(data))
	err := p.svr.SendToId(id,data);
	if(err!=nil){
		//fmt.Println("ssend",err);
	}
}
func (p * Lis)OnClose(id int,err error){
	fmt.Println("OnClose",id,err,p.svr.ConnectionNum());
	if(p.svr.ConnectionNum() == 0 ){
		p.svr.Stop();
	}
}
func TestServer(t * testing.T) {
	svr,err := NewServer(":7686");
	if(err != nil){
		t.Log(err);
		t.FailNow();
		return
	}
	svr.SetListener(&Lis{svr})
	svr.Run()
}