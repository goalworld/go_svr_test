package server

import "testing"

func TestClient(t *testing.T) {
	svr, err := NewClient("192.168.0.100", 7686)
	if err != nil {
		t.Log(err)
		t.FailNow()
		return
	}
	err = svr.Send([]byte("hello world"))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	buf, errs := svr.Recv()
	if errs != nil {
		t.Log(err)
		t.FailNow()
	} else {
		t.Log(string(buf))
	}
}
