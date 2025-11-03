package cas

import (
	"errors"
	"os"
	"path"
)

// TODO: translate addr to path as addr[0]/addr[1]/addr[2:]. For that we'll need
// to create directories transitively (like mkdir -p)

func ReadContent(dir string, addr Address) ([]byte, error) {
	content, err := os.ReadFile(path.Join(dir, addr.String()))
	if err != nil {
		return nil, err
	}
	if AddressOf(content) != addr {
		return nil, errors.New("bad")
	}
	return content, nil
}

func WriteContent(dir string, content []byte) error {
	return writeContentAt(dir, AddressOf(content), content)
}

// TODO: make public?
func writeContentAt(dir string, addr Address, content []byte) error {
	return os.WriteFile(path.Join(dir, addr.String()), content, 0666)
}
