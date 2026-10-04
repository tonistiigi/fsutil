package fsutil

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSenderQueueCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		s := &sender{
			files:        map[uint32]string{1: "pending"},
			sendpipeline: make(chan *sendHandle, 1),
		}
		s.sendpipeline <- &sendHandle{id: 0, path: "queued"}

		done := make(chan struct{})
		var err error
		t.Cleanup(func() {
			cancel(context.Canceled)
			<-s.sendpipeline
			<-done
		})
		go func() {
			err = s.queue(ctx, 1)
			close(done)
		}()
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("queue returned while the pipeline was full")
		default:
		}

		cause := errors.New("send failed")
		cancel(cause)
		synctest.Wait()
		select {
		case <-done:
			require.ErrorIs(t, err, cause)
		default:
			t.Fatal("queue remained blocked after cancellation")
		}
	})
}
