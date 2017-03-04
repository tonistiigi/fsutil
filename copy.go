package fsutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1<<10)
	},
}

type Stream interface {
	RecvMsg(interface{}) error
	SendMsg(m interface{}) error
}

func Send(ctx context.Context, conn Stream, root string, opt *WalkOpt) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := &sender{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
		root:   root,
		opt:    opt,
		files:  make(map[uint32]string),
	}
	return s.run()
}

type sender struct {
	ctx    context.Context
	conn   Stream
	cancel func()
	opt    *WalkOpt
	root   string
	files  map[uint32]string
	mu     sync.RWMutex
}

func (s *sender) run() error {
	go s.send()
	for {
		var p Packet
		if err := s.conn.RecvMsg(&p); err == nil {
			switch p.Type {
			case PACKET_REQ:
				if err := s.queue(p.ID); err != nil {
					return err
				}
			case PACKET_FIN:
				return s.conn.SendMsg(&Packet{Type: PACKET_FIN})
			}
		}
	}
}

func (s *sender) queue(id uint32) error {
	// TODO: add worker threads
	// TODO: use something faster than map
	s.mu.Lock()
	p, ok := s.files[id]
	if !ok {
		s.mu.Unlock()
		return errors.Errorf("invalid file id %d", id)
	}
	delete(s.files, id)
	s.mu.Unlock()
	go s.sendFile(id, p)
	return nil
}

func (s *sender) sendFile(id uint32, p string) error {
	f, err := os.Open(filepath.Join(s.root, p))
	if err == nil {
		buf := bufPool.Get().([]byte)
		defer bufPool.Put(buf)
		if _, err := io.CopyBuffer(&fileSender{conn: s.conn, id: id}, f, buf); err != nil {
			return err // TODO: handle error
		}
	}
	return s.conn.SendMsg(&Packet{ID: id, Type: PACKET_DATA})
}

func (s *sender) send() error {
	var i uint32 = 0
	err := Walk(s.ctx, s.root, s.opt, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		stat, ok := fi.Sys().(*Stat)
		if !ok {
			return errors.Wrapf(err, "invalid fileinfo without stat info: %s", path)
		}
		p := &Packet{
			Type: PACKET_STAT,
			Stat: stat,
		}
		s.mu.Lock()
		s.files[i] = stat.Path
		i++
		s.mu.Unlock()
		return errors.Wrapf(s.conn.SendMsg(p), "failed to send stat %s", path)
	})
	if err != nil {
		return err
	}
	return errors.Wrapf(s.conn.SendMsg(&Packet{Type: PACKET_STAT}), "failed to send last stat")
}

type fileSender struct {
	conn Stream
	id   uint32
}

func (fs *fileSender) Write(dt []byte) (int, error) {
	if len(dt) == 0 {
		return 0, nil
	}
	if err := fs.conn.SendMsg(&Packet{Type: PACKET_DATA, ID: fs.id, Data: dt}); err != nil {
		return 0, err
	}
	return len(dt), nil
}

func Receive(ctx context.Context, conn Stream, dest string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &receiver{
		ctx: ctx,
		// cancel: cancel,
		conn:     conn,
		dest:     dest,
		files:    make(map[string]uint32),
		pipes:    make(map[uint32]*io.PipeWriter),
		walkChan: make(chan *currentPath, 128),
		walkDone: make(chan struct{}),
	}
	return r.run()
}

type receiver struct {
	dest     string
	ctx      context.Context
	conn     Stream
	files    map[string]uint32
	pipes    map[uint32]*io.PipeWriter
	mu       sync.RWMutex
	muPipes  sync.RWMutex
	walkChan chan *currentPath
	walkDone chan struct{}
}

func (r *receiver) readStat(ctx context.Context, pathC chan<- *currentPath) error {
	for p := range r.walkChan {
		pathC <- p
	}
	return nil
}

func (r *receiver) run() error {
	dw := DiskWriter{
		asyncDataFunc: r.getAsyncDataFunc(),
		dest:          r.dest,
	}
	//todo: add errgroup
	go func() {
		doubleWalkDiff(r.ctx, dw.HandleChange, GetWalkerFn(r.dest), r.readStat)
		close(r.walkDone)
	}()

	var i uint32 = 0

	var p Packet
	for {
		p = Packet{Data: p.Data[:0]}
		if err := r.conn.RecvMsg(&p); err == nil {
			switch p.Type {
			case PACKET_STAT:
				if p.Stat == nil {
					close(r.walkChan)
					<-r.walkDone
					go func() {
						dw.Wait()
						r.conn.SendMsg(&Packet{Type: PACKET_FIN})
					}()
					break
				}
				if os.FileMode(p.Stat.Mode)&(os.ModeDir|os.ModeSymlink|os.ModeNamedPipe|os.ModeDevice) == 0 {
					r.mu.Lock()
					r.files[p.Stat.Path] = i
					r.mu.Unlock()
				}
				i++
				r.walkChan <- &currentPath{path: p.Stat.Path, f: &StatInfo{p.Stat}}
			case PACKET_DATA:
				r.muPipes.Lock()
				pw, ok := r.pipes[p.ID]
				if !ok {
					r.muPipes.Unlock()
					return errors.Errorf("invalid file request %s", p.ID)
				}
				r.muPipes.Unlock()
				if len(p.Data) == 0 {
					if err := pw.Close(); err != nil {
						return err
					}
				} else {
					if _, err := pw.Write(p.Data); err != nil {
						return err
					}
				}
			case PACKET_FIN:
				return nil
			}
		}
	}
	return nil
}

func (r *receiver) getAsyncDataFunc() writeToFunc {
	return func(ctx context.Context, p string, wc io.WriteCloser) error {
		r.mu.Lock()
		id, ok := r.files[p]
		if !ok {
			r.mu.Unlock()
			return errors.Errorf("invalid file request %s", p)
		}
		delete(r.files, p)
		r.mu.Unlock()

		pr, pw := io.Pipe()
		r.muPipes.Lock()
		r.pipes[id] = pw
		r.muPipes.Unlock()

		if err := r.conn.SendMsg(&Packet{Type: PACKET_REQ, ID: id}); err != nil {
			return err
		}

		buf := bufPool.Get().([]byte)
		defer bufPool.Put(buf)
		if _, err := io.CopyBuffer(wc, pr, buf); err != nil {
			return err
		}
		return wc.Close()
	}
}
