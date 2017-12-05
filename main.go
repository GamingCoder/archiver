package main

import (
	"archive/tar"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const defaultFormat = "workspace-" + time.RFC3339 + ".tar"

func main() {
	src := flag.String("src", "", "Path to directory to archive")
	target := flag.String("target", "", "Path to storage directory")
	deleteOld := flag.Bool("delete-old", false, "Delete older archives with same content")
	format := flag.String("format", defaultFormat, "Archive name format")
	flag.Parse()
	if *src == "" {
		log.Fatalln("Path to src directory must not be empty")
	}
	if *target == "" {
		log.Fatalln("Path to storage directory must not be empty")
	}

	// Determine absolute paths
	srcPath, err := filepath.Abs(*src)
	if err != nil {
		log.Fatalf("%+v\n", errors.Wrapf(err, "error determining absolute path of path %q", *src))
	}
	targetPath, err := filepath.Abs(*target)
	if err != nil {
		log.Fatalf("%+v\n", errors.Wrapf(err, "error determining absolute path of path %q", *target))
	}

	// Create new archive
	tarPath := filepath.Join(targetPath, time.Now().Format(*format))
	err = writeTar(srcPath, tarPath)
	if err != nil {
		log.Fatalf("%+v\n", errors.Wrap(err, "error writing tar archive"))
	}

	// Delete older archives with same content if delete-old flag is TRUE
	if *deleteOld {
		err = delTarsWithSameContent(targetPath, tarPath)
		if err != nil {
			log.Fatalf("%+v\n", errors.Wrap(err, "error deleting old archives with same content"))
		}
	}
}

func writeTar(srcPath string, tarPath string) error {
	f, err := os.OpenFile(tarPath, os.O_WRONLY|os.O_CREATE, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening file ")
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	defer tw.Close()

	err = filepath.Walk(srcPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error with dir walker")
		}
		// The os.FileInfo provided to the walkFn function cannot be used for our use case.
		// Since internally filepath.Walk uses the os.Lstat method which does not follow symlinks
		// and therefore we cannot tell if the file is a symlink to a file or a dir.
		fi, err := os.Stat(path)
		if err != nil {
			return errors.Wrapf(err, "error getting os.FileInfo for path %q", path)
		}
		if fi.IsDir() {
			return nil
		}
		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return errors.Wrapf(err, "error converting os.FileInfo to tar.Header %+v\n", err)
		}
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return errors.Wrapf(err, "error converting path %q to a relative path")
		}
		hdr.Name = relPath

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "error opening file %q", path)
		}
		defer f.Close()
		tw.WriteHeader(hdr)

		_, err = io.Copy(tw, f)
		if err != nil {
			return errors.Wrap(err, "error writing file contents")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error walking src directory")
	}
	return nil
}

func delTarsWithSameContent(targetPath string, tarPath string) error {
	type File struct {
		Path string
		Fi   os.FileInfo
	}
	sums := make(map[string]File)
	err := filepath.Walk(targetPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error with dir walker")
		}
		// Skip dirs
		if fi.IsDir() {
			return nil
		}
		// Generate checksum of file
		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "error opening file %q", path)
		}
		sum := md5.New()
		_, err = io.Copy(sum, f)
		if err != nil {
			return errors.Wrap(err, "error generating checksum")
		}
		f.Close()
		// Test if a file with the same content as the current file exists
		// If not store info about the current file
		if file, ok := sums[fmt.Sprintf("%x", sum.Sum(nil))]; ok {
			// Delete the older file
			if file.Fi.ModTime().Before(fi.ModTime()) {
				err = os.Remove(file.Path)
				if err != nil {
					return errors.Wrapf(err, "error deleting file %q", file.Path)
				}
				sums[fmt.Sprintf("%x", sum.Sum(nil))] = File{path, fi}
			} else {
				err = os.Remove(path)
				if err != nil {
					return errors.Wrapf(err, "error deleting file %q", path)
				}
			}
		} else {
			sums[fmt.Sprintf("%x", sum.Sum(nil))] = File{path, fi}
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error walking target directory")
	}
	return nil
}
