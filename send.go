package fsutil

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil/types"
	"golang.org/x/sync/errgroup"
)

const bufferSize = 32 * 1024

var bufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, bufferSize)
		return &buf
	},
}

type Stream interface {
	RecvMsg(any) error
	SendMsg(m any) error
	Context() context.Context
}

func Send(ctx context.Context, conn Stream, fs FS, progressCb func(int, bool)) error {
	s := &sender{
		conn:         &syncStream{Stream: conn},
		fs:           WithHardlinkReset(fs),
		files:        make(map[uint32]string),
		progressCb:   progressCb,
		sendpipeline: make(chan *sendHandle, 128),
	}
	return s.run(ctx)
}

type sendHandle struct {
	id   uint32
	path string
}

type sender struct {
	conn              Stream
	fs                FS
	files             map[uint32]string
	mu                sync.RWMutex
	progressCb        func(int, bool)
	progressCurrent   int
	progressCurrentMu sync.Mutex
	sendpipeline      chan *sendHandle
}

func (s *sender) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	hashLimit := make(chan struct{}, hashConcurrency())

	defer s.updateProgress(0, true)

	g.Go(func() error {
		err := s.walk(ctx)
		if err != nil {
			s.conn.SendMsg(&types.Packet{Type: types.PACKET_ERR, Data: []byte(err.Error())})
		}
		return err
	})

	for range 4 {
		g.Go(func() error {
			for h := range s.sendpipeline {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if err := s.sendFile(h); err != nil {
					return err
				}
			}
			return nil
		})
	}

	g.Go(func() error {
		defer close(s.sendpipeline)

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			var p types.Packet
			if err := s.conn.RecvMsg(&p); err != nil {
				return err
			}
			switch p.Type {
			case types.PACKET_ERR:
				return errors.Errorf("error from receiver: %s", p.Data)
			case types.PACKET_REQ:
				if err := s.queue(p.ID); err != nil {
					return err
				}
			case types.PACKET_HASH_REQ:
				path, err := s.lookup(p.ID)
				if err != nil {
					return err
				}
				h := &sendHandle{p.ID, path}
				g.Go(func() error {
					select {
					case hashLimit <- struct{}{}:
						defer func() { <-hashLimit }()
					case <-ctx.Done():
						return ctx.Err()
					}
					return s.sendHash(h)
				})
			case types.PACKET_FIN:
				return s.conn.SendMsg(&types.Packet{Type: types.PACKET_FIN})
			}
		}
	})

	return g.Wait()
}

func hashConcurrency() int {
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		return 1
	}
	return n
}

func (s *sender) updateProgress(size int, last bool) {
	if s.progressCb != nil {
		s.progressCurrentMu.Lock()
		defer s.progressCurrentMu.Unlock()
		s.progressCurrent += size
		s.progressCb(s.progressCurrent, last)
	}
}

func (s *sender) queue(id uint32) error {
	s.mu.Lock()
	p, ok := s.files[id]
	if !ok {
		s.mu.Unlock()
		return errors.Errorf("invalid file id %d", id)
	}
	delete(s.files, id)
	s.mu.Unlock()
	s.sendpipeline <- &sendHandle{id, p}
	return nil
}

func (s *sender) lookup(id uint32) (string, error) {
	s.mu.RLock()
	p, ok := s.files[id]
	s.mu.RUnlock()
	if !ok {
		return "", errors.Errorf("invalid file id %d", id)
	}
	return p, nil
}

func (s *sender) sendHash(hdl *sendHandle) error {
	f, err := s.fs.Open(hdl.path)
	if err != nil {
		return err
	}

	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)

	h := sha256.New()
	if _, err := io.CopyBuffer(h, f, *buf); err != nil {
		f.Close()
		return errors.WithStack(err)
	}
	if err := f.Close(); err != nil {
		return errors.WithStack(err)
	}

	packet := &types.Packet{ID: hdl.id, Type: types.PACKET_HASH, Data: h.Sum(nil)}
	s.updateProgress(packet.Size(), false)
	return s.conn.SendMsg(packet)
}

func (s *sender) sendFile(h *sendHandle) error {
	f, err := s.fs.Open(h.path)
	if err == nil {
		defer f.Close()
		buf := bufPool.Get().(*[]byte)
		defer bufPool.Put(buf)
		if _, err := io.CopyBuffer(&fileSender{sender: s, id: h.id}, struct{ io.Reader }{f}, *buf); err != nil {
			return err
		}
	}
	return s.conn.SendMsg(&types.Packet{ID: h.id, Type: types.PACKET_DATA})
}

func (s *sender) walk(ctx context.Context) error {
	var i uint32 = 0
	target := string(filepath.Separator)
	err := s.fs.Walk(ctx, target, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := fi.Sys().(*types.Stat)
		if !ok {
			return errors.WithStack(&os.PathError{Path: path, Err: syscall.EBADMSG, Op: "fileinfo without stat info"})
		}
		stat.Path = filepath.ToSlash(stat.Path)
		stat.Linkname = filepath.ToSlash(stat.Linkname)
		p := &types.Packet{
			Type: types.PACKET_STAT,
			Stat: stat,
		}
		if fileCanRequestData(os.FileMode(stat.Mode)) {
			s.mu.Lock()
			s.files[i] = stat.Path
			s.mu.Unlock()
		}
		i++
		s.updateProgress(p.Size(), false)
		return errors.Wrapf(s.conn.SendMsg(p), "failed to send stat %s", path)
	})
	if err != nil {
		return err
	}
	return errors.Wrapf(s.conn.SendMsg(&types.Packet{Type: types.PACKET_STAT}), "failed to send last stat")
}

func fileCanRequestData(m os.FileMode) bool {
	// avoid updating this function as it needs to match between sender/receiver.
	// version if needed
	return m&os.ModeType == 0
}

type fileSender struct {
	sender *sender
	id     uint32
}

func (fs *fileSender) Write(dt []byte) (int, error) {
	if len(dt) == 0 {
		return 0, nil
	}
	p := &types.Packet{Type: types.PACKET_DATA, ID: fs.id, Data: dt}
	if err := fs.sender.conn.SendMsg(p); err != nil {
		return 0, err
	}
	fs.sender.updateProgress(p.Size(), false)
	return len(dt), nil
}

type syncStream struct {
	Stream
	mu sync.Mutex
}

func (ss *syncStream) SendMsg(m any) error {
	ss.mu.Lock()
	err := ss.Stream.SendMsg(m)
	ss.mu.Unlock()
	return err
}
