package bench

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil"
	"golang.org/x/net/context"
)

func diffCopy(proto bool, src, dest string) error {
	var s1, s2 fsutil.Stream
	if proto {
		s1, s2 = sockPairProto()
	} else {
		s1, s2 = sockPair()
	}

	var err1 error
	var err2 error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err1 = fsutil.Send(context.Background(), s1, src, nil)
		wg.Done()
	}()
	go func() {
		err2 = fsutil.Receive(context.Background(), s2, dest)
		wg.Done()
	}()

	wg.Wait()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func diffCopyProto(src, dest string) error {
	return diffCopy(true, src, dest)
}
func diffCopyReg(src, dest string) error {
	return diffCopy(false, src, dest)
}

func sockPair() (fsutil.Stream, fsutil.Stream) {
	c1 := make(chan *fsutil.Packet, 64)
	c2 := make(chan *fsutil.Packet, 64)
	return &fakeConn{c1, c2}, &fakeConn{c2, c1}
}

type fakeConn struct {
	recvChan chan *fsutil.Packet
	sendChan chan *fsutil.Packet
}

func (fc *fakeConn) Context() context.Context {
	return context.TODO()
}

func (fc *fakeConn) RecvMsg(m interface{}) error {
	p, ok := m.(*fsutil.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := <-fc.recvChan
	*p = *p2
	return nil
}

func (fc *fakeConn) SendMsg(m interface{}) error {
	p, ok := m.(*fsutil.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := *p
	p2.Data = append([]byte{}, p2.Data...)
	fc.sendChan <- &p2
	return nil
}

func sockPairProto() (fsutil.Stream, fsutil.Stream) {
	c1 := make(chan []byte, 64)
	c2 := make(chan []byte, 64)
	return &fakeConnProto{c1, c2}, &fakeConnProto{c2, c1}
}

type fakeConnProto struct {
	recvChan chan []byte
	sendChan chan []byte
}

func (fc *fakeConnProto) Context() context.Context {
	return context.TODO()
}

func (fc *fakeConnProto) RecvMsg(m interface{}) error {
	p, ok := m.(*fsutil.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	dt := <-fc.recvChan
	return p.Unmarshal(dt)
}

func (fc *fakeConnProto) SendMsg(m interface{}) error {
	p, ok := m.(*fsutil.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	dt, err := p.Marshal()
	if err != nil {
		return err
	}
	fc.sendChan <- dt
	return nil
}
