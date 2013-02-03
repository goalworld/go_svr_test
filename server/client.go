package server
import "net"
type Client struct{
	con ConnCoder
}

func NewClient(ip string,port int) (p*Client,err error) {
	
	addr := net.TCPAddr{net.ParseIP(ip),port};
	var con *net.TCPConn
	con ,err = net.DialTCP("tcp",nil,&addr);
	if err != nil {
		p = nil
		return 
	}
	p = new(Client)
	p.con = newHeadConnCoder(con)
	return
}
func (p * Client)Send( data []byte) error{
	return p.con.WriteMessage(data)
}
func (p * Client)Recv( ) ([]byte,error){
	return p.con.ReadMessage()
}