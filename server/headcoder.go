package server

import (
	"bufio"
	"errors"
	"io"
)

type ConnCoder interface {
	WriteMessage(buf []byte) error
	ReadMessage() ([]byte, error)
	Close() error
}

type headConnCoder struct {
	w   *bufio.Writer
	wrc io.ReadWriteCloser
	buf [4]byte
}

func newHeadConnCoder(wrc io.ReadWriteCloser) *headConnCoder {
	return &headConnCoder{w: bufio.NewWriter(wrc), wrc: wrc}
}
func (p *headConnCoder) WriteMessage(buf []byte) error {
	lens := len(buf)
	if lens == 0 {
		return errors.New("write buf empty")
	}
	p.writeHead(uint32(len(buf)))
	p.w.Write(buf)
	return p.w.Flush()
}
func (p *headConnCoder) ReadMessage() (buf []byte, err error) {
	var head uint32
	head, err = p.readHead()
	if err != nil {
		return
	}
	buf = make([]byte, head)
	_, err = io.ReadFull(p.wrc, buf)
	if err != nil {
		buf = nil
	}
	return
}
func (p *headConnCoder) Close() error {
	return p.wrc.Close()
}
func (p *headConnCoder) writeHead(lens uint32) {
	p.w.WriteByte(byte(lens >> 24 & 0x000000ff))
	p.w.WriteByte(byte(lens >> 16 & 0x000000ff))
	p.w.WriteByte(byte(lens >> 8 & 0x000000ff))
	p.w.WriteByte(byte(lens & 0x000000ff))
}
func (p *headConnCoder) readHead() (uint32, error) {
	_, err := io.ReadFull(p.wrc, p.buf[0:])
	if err != nil {
		return 0, err
	}
	return uint32(p.buf[0])<<24 | uint32(p.buf[1])<<16 | uint32(p.buf[2])<<8 | uint32(p.buf[3]), nil
}
