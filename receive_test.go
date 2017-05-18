// +build linux

package fsutil

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestCopySimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
		"ADD foo2 file dat2",
		"ADD zzz dir",
		"ADD zzz/aa file data3",
		"ADD zzz/bb dir",
		"ADD zzz/bb/cc dir",
		"ADD zzz/bb/cc/dd symlink ../../",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest, err := ioutil.TempDir("", "dest")
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	s1, s2 := sockPairProto()

	ts := NewTarsum("")
	chs := &changes{fn: ts.HandleChange}

	var err1 error
	var err2 error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err1 = Send(context.Background(), s1, d, nil, nil)
		wg.Done()
	}()
	go func() {
		err2 = Receive(context.Background(), s2, dest, chs.HandleChange)
		wg.Done()
	}()

	wg.Wait()
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
file foo2
dir zzz
file zzz/aa
dir zzz/bb
dir zzz/bb/cc
symlink:../../ zzz/bb/cc/dd
`)

	dt, err := ioutil.ReadFile(filepath.Join(dest, "zzz/aa"))
	assert.NoError(t, err)
	assert.Equal(t, "data3", string(dt))

	dt, err = ioutil.ReadFile(filepath.Join(dest, "foo2"))
	assert.NoError(t, err)
	assert.Equal(t, "dat2", string(dt))

	h, err := ts.Hash("zzz/aa")
	assert.NoError(t, err)
	assert.Equal(t, h, "b1c874520ef2887ebace8ff70591ff248138b19197e9e232df9c9866cb581705")

	h, err = ts.Hash("foo2")
	assert.NoError(t, err)
	assert.Equal(t, h, "4524c0852a5745ea830e63da563f58e6b507ca1bfdf0075db3baa627317651cb")

	h, err = ts.Hash("zzz/bb/cc/dd")
	assert.NoError(t, err)
	assert.Equal(t, h, "320a4733517590b42d628d51ac2ec8f305fc985ec36ac9bcb7d4e7376441c851")

	k, ok := chs.c["zzz/aa"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindAdd)

	err = ioutil.WriteFile(filepath.Join(d, "zzz/bb/cc/foo"), []byte("data5"), 0600)
	assert.NoError(t, err)

	err = os.RemoveAll(filepath.Join(d, "foo2"))
	assert.NoError(t, err)

	chs = &changes{fn: ts.HandleChange}

	wg.Add(2)
	go func() {
		err1 = Send(context.Background(), s1, d, nil, nil)
		wg.Done()
	}()
	go func() {
		err2 = Receive(context.Background(), s2, dest, chs.HandleChange)
		wg.Done()
	}()

	wg.Wait()
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	b = &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
dir zzz
file zzz/aa
dir zzz/bb
dir zzz/bb/cc
symlink:../../ zzz/bb/cc/dd
file zzz/bb/cc/foo
`)

	dt, err = ioutil.ReadFile(filepath.Join(dest, "zzz/bb/cc/foo"))
	assert.NoError(t, err)
	assert.Equal(t, "data5", string(dt))

	h, err = ts.Hash("zzz/bb/cc/dd")
	assert.NoError(t, err)
	assert.Equal(t, h, "320a4733517590b42d628d51ac2ec8f305fc985ec36ac9bcb7d4e7376441c851")

	h, err = ts.Hash("zzz/bb/cc/foo")
	assert.NoError(t, err)
	assert.Equal(t, h, "d953e7f96eda58e257c2bfc033e5de66a541999d884b46d235709e6414898638")

	h, err = ts.Hash("foo2")
	assert.NoError(t, err)
	assert.Equal(t, h, "foo2")

	k, ok = chs.c["foo2"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindDelete)

	k, ok = chs.c["zzz/bb/cc/foo"]
	assert.Equal(t, ok, true)
	assert.Equal(t, k, ChangeKindAdd)

	_, ok = chs.c["zzz/aa"]
	assert.Equal(t, ok, false)
}

func sockPair() (Stream, Stream) {
	c1 := make(chan *Packet, 32)
	c2 := make(chan *Packet, 32)
	return &fakeConn{c1, c2}, &fakeConn{c2, c1}
}

func sockPairProto() (Stream, Stream) {
	c1 := make(chan []byte, 32)
	c2 := make(chan []byte, 32)
	return &fakeConnProto{c1, c2}, &fakeConnProto{c2, c1}
}

type fakeConn struct {
	recvChan chan *Packet
	sendChan chan *Packet
}

func (fc *fakeConn) RecvMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := <-fc.recvChan
	*p = *p2
	return nil
}

func (fc *fakeConn) SendMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := *p
	p2.Data = append([]byte{}, p2.Data...)
	fc.sendChan <- &p2
	return nil
}

type fakeConnProto struct {
	recvChan chan []byte
	sendChan chan []byte
}

func (fc *fakeConnProto) RecvMsg(m interface{}) error {
	p, ok := m.(*Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	dt := <-fc.recvChan
	return p.Unmarshal(dt)
}

func (fc *fakeConnProto) SendMsg(m interface{}) error {
	p, ok := m.(*Packet)
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

type changes struct {
	c  map[string]ChangeKind
	fn ChangeFunc
}

func (c *changes) HandleChange(kind ChangeKind, p string, fi os.FileInfo, err error) error {
	if c.c == nil {
		c.c = make(map[string]ChangeKind)
	}
	c.c[p] = kind
	return c.fn(kind, p, fi, err)
}
