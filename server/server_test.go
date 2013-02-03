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
	err := p.svr.Send(id,data);
	if(err!=nil){
		//fmt.Println("ssend",err);
	}
}
func (p * Lis)OnClose(id int,err error){
	fmt.Println("OnClose",id,err,p.svr.ConnectionNum());
}
func TestServer(t * testing.T) {
	out := make(chan *Message)
	svr,err := NewServer(":7686",out);
	if(err != nil){
		t.Log(err);
		t.FailNow();
		return
	}
	lis := Lis{svr}
	svr.Run()
	for {
		select{
			case msg := <- out:
				switch msg.Type{
				case Type_connect:
					lis.OnConnect(msg.Id)
				case Type_message:
					lis.OnMessage(msg.Id,msg.Data)
				case Type_close:
					lis.OnClose(msg.Id,msg.Err)
				default:
					println("UNKONW MESSAGE",msg)
				}
		}
	}
	svr.Close()
	svr.Wait()
	
}