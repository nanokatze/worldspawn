package main

import (
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"flag"
	"io"
	"log"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"worldspawn/cook/internal/cookers/blend"
	"worldspawn/cook/internal/work"

	"golang.org/x/sync/errgroup"
)

func main() {
	log.SetFlags(0)

	cook(os.Args[1:])
}

type Content struct {
	Files map[string][]string `json:"files"`
}

// TODO: have a higher level Context object. Also,
type Action = work.Action[*Artifacts]

// TODO: pass a huge context object
type Cooker func(srcDir *os.Root, file string) (*Action, error)

type Artifacts struct {
	dir string
}

// TODO: keep track of created artifacts
func (a *Artifacts) Create(file string) (*os.File, error) {
	// We don't need to check for os.MkdirAll failure, os.Create will fail if MkdirAll fails.
	os.MkdirAll(filepath.Join(a.dir, path.Dir(file)), 0755)
	return os.Create(filepath.Join(a.dir, file))
}

func copyCooker(srcDir *os.Root, file string) (*Action, error) {
	// TODO: we should have a more ergonomic hash helper, this one sucks
	stat, err := srcDir.Stat(file)
	if err != nil {
		return nil, err
	}

	return &Action{
		Hash: hash(stat.ModTime()),
		Exec: func(outDir *Artifacts) error {
			out := strings.TrimSuffix(file, path.Ext(file))

			// TODO: use filesystem features for copy when possible

			srcf, err := srcDir.Open(file)
			if err != nil {
				return err
			}
			defer srcf.Close()

			outf, err := outDir.Create(out)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outf, srcf); err != nil {
				return err
			}
			return outf.Close()
		},
	}, nil
}

var cookers = map[string]Cooker{
	"blend": func(srcDir *os.Root, file string) (*Action, error) {
		// TODO: in some cases we might need to go through the blend and append
		// all of the outputs.

		stat, err := srcDir.Stat(file)
		if err != nil {
			return nil, err
		}

		return &Action{
			Hash: hash(stat.ModTime()),
			Exec: func(outDir *Artifacts) error {
				return blend.Cook(outDir.dir, srcDir.Name(), file)
			},
		}, nil
	},
	"wav":  copyCooker,
	"ktx2": copyCooker,
}

func probe(root *os.Root, file string) (Cooker, error) {
	cooker, ok := cookers[path.Ext(file)[1:]]
	if !ok {
		return nil, errors.New("could not figure out the cooker")
	}
	return cooker, nil
}

func cook(args []string) {
	var flagSet flag.FlagSet
	flagSet.Int("j", runtime.NumCPU(), "")
	outDir := flagSet.String("o", "", "")
	flagSet.Parse(args)

	// TODO: just os.Chdir into srcRoot actually instead. We'll want to fixup
	// outDir though.
	srcRoot, err := os.OpenRoot(flagSet.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	// if err := os.Chdir(flagSet.Arg(0)); err != nil {
	// 	log.Fatal(err)
	// }

	f, err := srcRoot.Open(".content.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var content Content
	if err := json.UnmarshalRead(f, &content); err != nil {
		log.Fatal(err)
	}

	actions := make(map[*Action]struct{})
	for _, file := range slices.Sorted(maps.Keys(content.Files)) {
		cooker, err := probe(srcRoot, file)
		if err != nil {
			log.Fatalf("%v: %v", file, err)
		}

		// TODO: pass this to the cooker
		// args := project.Files[file]

		action, err := cooker(srcRoot, file)
		if err != nil {
			log.Fatalf("%v: %v", file, err)
		}

		// TODO: flatten actions
		actions[action] = struct{}{}
	}

	artifacts := Artifacts{*outDir}

	// TODO: introduce executor abstraction which will figure out dependencies
	// TODO: cache

	var wg errgroup.Group
	wg.SetLimit(runtime.NumCPU() + 2)

	// TODO: if running deterministically, sort by hash first
	for action := range actions {
		wg.Go(func() error {
			// TODO: wrap the error if it occured so that we can inform the user
			// of the action that failed
			return action.Exec(&artifacts)
		})
	}

	if err := wg.Wait(); err != nil {
		log.Fatal(err)
	}
}

func hash(stuff ...any) [work.HashSize]byte {
	hasher := sha256.New()
	if err := json.MarshalWrite(hasher, &stuff, json.Deterministic(true)); err != nil {
		panic(err)
	}
	return [work.HashSize]byte(hasher.Sum(nil))
}
