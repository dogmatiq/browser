package githubx

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

// GetArchive downloads and uncompresses a source archive from GitHub.
//
// It returns the name of a temporary directory containing the root of the
// repository.
func GetArchive(
	ctx context.Context,
	c *http.Client,
	url string,
) (_ string, err error) {
	if c == nil {
		c = http.DefaultClient
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	// Remove the temporary directory if there is a panic or an error is
	// returned by this function.
	defer func() {
		if r := recover(); r != nil {
			os.RemoveAll(dir)
			panic(r)
		}

		if err != nil {
			os.RemoveAll(dir)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	uncompressed, err := gzip.NewReader(res.Body)
	if err != nil {
		return "", err
	}
	defer uncompressed.Close()

	archive := tar.NewReader(uncompressed)
	trimPrefix := ""

	for {
		header, err := archive.Next()
		if err == io.EOF {
			return dir, nil
		}
		if err != nil {
			return "", err
		}

		header.Name = strings.TrimPrefix(header.Name, trimPrefix)
		entryPath := path.Join(dir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if trimPrefix == "" {
				// GitHub puts the repository content inside another top-level
				// directory within the archive. We don't want this directory,
				// so we move the content of the archive "up one level" by
				// trimming this directory name from the start of every file.
				trimPrefix = header.Name
				continue
			}

			if err := os.Mkdir(entryPath, 0700); err != nil {
				return "", err
			}

		case tar.TypeReg:
			f, err := os.OpenFile(entryPath, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(f, archive); err != nil {
				return "", err
			}
		}
	}
}
