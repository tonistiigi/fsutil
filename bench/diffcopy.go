package bench

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil"
	fstypes "github.com/tonistiigi/fsutil/types"
	"golang.org/x/sync/errgroup"
)

type closeableStream interface {
	fsutil.Stream
	io.Closer
}

func diffCopy(proto bool, src, dest string) error {
	return diffCopyPath(proto, src, dest)
}

func diffCopyPath(proto bool, src, dest string) error {
	var s1, s2 closeableStream

	eg, ctx := errgroup.WithContext(context.Background())

	if proto {
		s1, s2 = sockPairProto(ctx)
	} else {
		s1, s2 = sockPair(ctx)
	}

	eg.Go(func() error {
		defer s1.Close()
		fs, err := fsutil.NewFS(src)
		if err != nil {
			return err
		}
		return fsutil.Send(ctx, s1, fs, nil)
	})
	eg.Go(func() error {
		defer s2.Close()
		return fsutil.Receive(ctx, s2, dest, fsutil.ReceiveOpt{})
	})

	return eg.Wait()
}

func diffCopyProto(src, dest string) error {
	return diffCopy(true, src, dest)
}
func diffCopyReg(src, dest string) error {
	return diffCopy(false, src, dest)
}

func diffCopyRoot(proto bool, src, dest string) error {
	var s1, s2 closeableStream

	eg, ctx := errgroup.WithContext(context.Background())

	if proto {
		s1, s2 = sockPairProto(ctx)
	} else {
		s1, s2 = sockPair(ctx)
	}

	eg.Go(func() error {
		defer s1.Close()
		osroot, err := os.OpenRoot(src)
		if err != nil {
			return err
		}
		root := fsutil.NewRoot(osroot)
		defer root.Close()
		return fsutil.Send(ctx, s1, fsutil.NewRootFS(root), nil)
	})
	eg.Go(func() error {
		defer s2.Close()
		osroot, err := os.OpenRoot(dest)
		if err != nil {
			return err
		}
		root := fsutil.NewRoot(osroot)
		defer root.Close()
		return fsutil.ReceiveRoot(ctx, s2, root, fsutil.ReceiveOpt{})
	})

	return eg.Wait()
}

func sockPair(ctx context.Context) (closeableStream, closeableStream) {
	c1 := make(chan *fstypes.Packet, 64)
	c2 := make(chan *fstypes.Packet, 64)
	return &fakeConn{ctx: ctx, recvChan: c1, sendChan: c2}, &fakeConn{ctx: ctx, recvChan: c2, sendChan: c1}
}

func sockPairProto(ctx context.Context) (closeableStream, closeableStream) {
	c1 := make(chan []byte, 64)
	c2 := make(chan []byte, 64)
	return &fakeConnProto{ctx: ctx, recvChan: c1, sendChan: c2}, &fakeConnProto{ctx: ctx, recvChan: c2, sendChan: c1}
}

type fakeConn struct {
	ctx      context.Context
	recvChan chan *fstypes.Packet
	sendChan chan *fstypes.Packet
	close    sync.Once
}

func (fc *fakeConn) Context() context.Context {
	return fc.ctx
}

func (fc *fakeConn) Close() error {
	fc.close.Do(func() {
		close(fc.sendChan)
	})
	return nil
}

func (fc *fakeConn) RecvMsg(m interface{}) error {
	p, ok := m.(*fstypes.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case p2, ok := <-fc.recvChan:
		if !ok {
			return io.EOF
		}
		*p = *p2
		return nil
	}
}

func (fc *fakeConn) SendMsg(m interface{}) error {
	p, ok := m.(*fstypes.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	p2 := *p
	p2.Data = append([]byte{}, p2.Data...)
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case fc.sendChan <- &p2:
		return nil
	}
}

type fakeConnProto struct {
	ctx      context.Context
	recvChan chan []byte
	sendChan chan []byte
	close    sync.Once
}

func (fc *fakeConnProto) Context() context.Context {
	return fc.ctx
}

func (fc *fakeConnProto) Close() error {
	fc.close.Do(func() {
		close(fc.sendChan)
	})
	return nil
}

func (fc *fakeConnProto) RecvMsg(m interface{}) error {
	p, ok := m.(*fstypes.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case dt, ok := <-fc.recvChan:
		if !ok {
			return io.EOF
		}
		return p.Unmarshal(dt)
	}
}

func (fc *fakeConnProto) SendMsg(m interface{}) error {
	p, ok := m.(*fstypes.Packet)
	if !ok {
		return errors.Errorf("invalid msg: %#v", m)
	}
	dt, err := p.Marshal()
	if err != nil {
		return err
	}
	select {
	case <-fc.ctx.Done():
		return fc.ctx.Err()
	case fc.sendChan <- dt:
		return nil
	}
}
