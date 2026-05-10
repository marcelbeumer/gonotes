package gonotes

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var reIDPrefix = regexp.MustCompile(`^(\d{8})-(\d+)`)

// readDirBatch controls how many directory entries are read per ReadDir call.
// 256 is a reasonable batch size that balances memory use and syscall count.
const readDirBatch = 256

func MaxNumFromDir(dir string, now time.Time) (int, error) {
	prefix := now.Format("20060102")

	f, err := os.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("open dir: %w", err)
	}
	defer f.Close()

	maxNum := 0
	for {
		entries, err := f.ReadDir(readDirBatch)
		for _, e := range entries {
			m := reIDPrefix.FindStringSubmatch(e.Name())
			if m == nil {
				continue
			}
			if m[1] != prefix {
				continue
			}
			n, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			if n > maxNum {
				maxNum = n
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, fmt.Errorf("read dir: %w", err)
		}
	}

	return maxNum, nil
}

func idPrefix(t time.Time) string {
	return t.Format("20060102")
}

func fmtID(prefix string, num int) string {
	return fmt.Sprintf("%s-%d", prefix, num)
}

func NextID(dir string, now time.Time) (string, error) {
	maxNum, err := MaxNumFromDir(dir, now)
	if err != nil {
		return "", fmt.Errorf("max id: %w", err)
	}

	prefix := idPrefix(now)
	return fmtID(prefix, maxNum+1), nil
}

func NoteFilename(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "-" + slug + ".md"
}

func FolderName(id, slug string) string {
	if slug == "" {
		return id
	}
	return id + "-" + slug
}

// IDFromFilename extracts the note ID from a filename. The parsed return
// value is true when the filename starts with the yyyymmdd-N format.
func IDFromFilename(name string) (id string, parsed bool) {
	if !strings.HasSuffix(name, ".md") {
		return "", false
	}

	stem := strings.TrimSuffix(name, ".md")

	m := reIDPrefix.FindStringSubmatch(stem)
	if m != nil {
		return m[1] + "-" + m[2], true
	}

	return stem, false
}
