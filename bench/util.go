package bench

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

func createTestDir(n int) (string, error) {
	const nesting = 1.0 / 3.0
	rootDir, err := ioutil.TempDir(os.Getenv("BENCH_BASE_DIR"), "diffcopy")
	if err != nil {
		return "", err
	}

	dirs := int(math.Ceil(math.Pow(float64(n), nesting)))
	if err := fillTestDir(rootDir, dirs, n); err != nil {
		os.RemoveAll(rootDir)
		return "", err
	}
	return rootDir, nil
}

func fillTestDir(root string, items, n int) error {
	if n <= items {
		var size int64 = 64 * 1024
		if s, err := strconv.ParseInt(os.Getenv("BENCH_FILE_SIZE"), 10, 64); err == nil {
			size = s
		}
		b := make([]byte, size)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		for i := 0; i < items; i++ {
			fp := filepath.Join(root, randomID())
			tf, err := os.Create(fp)
			if err != nil {
				return err
			}
			if _, err := tf.Write(b); err != nil {
				return err
			}
			tf.Close()
		}
	} else {
		sub := n / items
		for n > 0 {
			fp := filepath.Join(root, randomID())
			if err := os.MkdirAll(fp, 0700); err != nil {
				return err
			}
			if n < sub {
				sub = n
			}
			if err := fillTestDir(fp, items, sub); err != nil {
				return err
			}
			n -= sub
		}
	}
	return nil
}

func randomID() string {
	b := make([]byte, 10)
	rand.Read(b)
	return hex.EncodeToString(b)
}
