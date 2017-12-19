package compression

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
)

func xzDecompress(archive io.Reader) (io.ReadCloser, error) {
	args := []string{"xz", "-d", "-c", "-q"}

	pipeR, pipeW := io.Pipe()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = archive
	cmd.Stdout = pipeW
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	// Run the command and return the pipe
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	chdone := make(chan struct{})
	chkill := make(chan struct{})

	go func() {
		if err := cmd.Wait(); err != nil {
			pipeW.CloseWithError(fmt.Errorf("%s: %s", err, errBuf.String()))
		} else {
			pipeW.Close()
		}
		close(chdone)
	}()

	go func() {
		select {
		case <-chkill:
			cmd.Process.Kill()
		case <-chdone:
		}
	}()

	return &readCloserWrapper{
		ioutil.NopCloser(pipeR),
		func() error {
			close(chkill)
			<-chdone
			return pipeR.Close()
		},
	}, nil

}
