package blend

import (
	"io"
	"os"
	"os/exec"
)

// TODO: rename to bpy or something
type blender struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func openBlender() (*blender, error) {
	// TODO: take the command + flags from the envvar
	cmd := exec.Command("blender", "-b", "-q", "--python-console")
	// cmd := exec.Command("python3.13")
	if true { // TODO: gate on envvar or similar
		cmd.Stdout = os.Stdout
	}
	if true {
		cmd.Stderr = os.Stderr
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &blender{
		cmd:   cmd,
		stdin: stdin,
	}, nil
}

// TODO: make a pool of blenders
// var getBlender = sync.OnceValues(openBlender)
