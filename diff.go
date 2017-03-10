package fsutil

import (
	"context"
	"os"
)

type walkerFn func(ctx context.Context, pathC chan<- *currentPath) error

func Changes(ctx context.Context, a, b walkerFn, changeFn ChangeFunc) error {
	return nil
}

func GetWalkerFn(root string) walkerFn {
	return func(ctx context.Context, pathC chan<- *currentPath) error {
		return Walk(ctx, root, nil, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			p := &currentPath{
				path: path,
				f:    f,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case pathC <- p:
				return nil
			}
		})
	}
}
