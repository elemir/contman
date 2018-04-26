package docker

import (
	"archive/tar"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func md5sum(reader io.Reader) (result string, err error) {
	hash := md5.New()

	_, err = io.Copy(hash, reader)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

func mkdir(target string) (err error) {
	_, err = os.Stat(target)
	if err == nil {
		return
	}

	err = os.MkdirAll(target, 0755)
	return
}

func hasEntryChanged(tr io.Reader, target string) (src io.Reader, changed bool, fail error) {
	src = tr
	changed = true
	fail = nil

	_, err := os.Stat(target)
	if err != nil {
		// Target likely doesn't exists
		return
	}

	f, err := os.Open(target)
	if err != nil {
		fail = err
		return
	}
	defer f.Close()

	targetHash, err := md5sum(f)
	if err != nil {
		fail = err
		return
	}

	// Duplicate archive stream to read it twice:
	// during hash calculation and when saving it to disk
	var buf bytes.Buffer
	tee := io.TeeReader(tr, &buf)

	sourceHash, err := md5sum(tee)
	if err != nil {
		fail = err
		return
	}

	// if our hashes are equal - we assume file is not changed
	if sourceHash == targetHash {
		changed = false
		return
	}

	src = &buf
	return
}

func extractEntryToFile(src io.Reader, target string, mode os.FileMode) error {
	f, errOpen := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if os.IsPermission(errOpen) {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			return errOpen
		}
		if err := os.Chmod(target, mode|os.FileMode(0200)); err != nil {
			return err
		}
		f, errOpen = os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	}
	if errOpen != nil {
		return errOpen
	}
	defer f.Close()
	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	return os.Chmod(target, mode)
}

func extractTarFromReader(r io.Reader, dest string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := mkdir(target); err != nil {
				return err
			}
		case tar.TypeReg:
			src, changed, err := hasEntryChanged(tr, target)
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			logrus.WithField("target", target).Debug("extracting entry")
			if err := extractEntryToFile(src, target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

func createTarToWriter(src string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(file, string(filepath.Separator))

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return nil
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})

}
