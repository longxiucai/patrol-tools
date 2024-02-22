package hash

import (
	"crypto/md5" // #nosec
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func MD5(body []byte) string {
	bytes := md5.Sum(body) // #nosec
	return hex.EncodeToString(bytes[:])
}

// FileMD5 count file md5
func FileMD5(path string) (string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}

	m := md5.New() // #nosec
	if _, err := io.Copy(m, file); err != nil {
		return "", err
	}

	fileMd5 := fmt.Sprintf("%x", m.Sum(nil))
	return fileMd5, nil
}
