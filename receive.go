// +build linux

package fsutil

import (
	"io"
	"os"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func Receive(ctx context.Context, conn Stream, dest string, notifyHashed ChangeFunc) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &receiver{
		ctx: ctx,
		// cancel: cancel,
		conn:         &syncStream{Stream: conn},
		dest:         dest,
		files:        make(map[string]uint32),
		pipes:        make(map[uint32]*io.PipeWriter),
		walkChan:     make(chan *currentPath, 128),
		walkDone:     make(chan struct{}),
		notifyHashed: notifyHashed,
	}
	return r.run()
}

type receiver struct {
	dest         string
	ctx          context.Context
	conn         Stream
	files        map[string]uint32
	pipes        map[uint32]*io.PipeWriter
	mu           sync.RWMutex
	muPipes      sync.RWMutex
	walkChan     chan *currentPath
	walkDone     chan struct{}
	notifyHashed ChangeFunc
}

func (r *receiver) readStat(ctx context.Context, pathC chan<- *currentPath) error {
	for {
		select {
		case p, ok := <-r.walkChan:
			if !ok {
				return nil
			}
			pathC <- p
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (r *receiver) run() error {
	dw := DiskWriter{
		asyncDataFunc: r.getAsyncDataFunc(),
		dest:          r.dest,
		notifyHashed:  r.notifyHashed,
	}
	//todo: add errgroup
	go func() {
		err := doubleWalkDiff(r.ctx, dw.HandleChange, GetWalkerFn(r.dest), r.readStat)
		if err != nil {
			logrus.Errorf("walkerr %s", err)
		}
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
		} else if err != nil {
			logrus.Error(err)
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
