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
			log.Fatalf("%+v\n", errors.Wrap(err, "error deleting old archives with same checksum"))
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

	err = filepath.Walk(srcPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error with dir walker")
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
	// Generate checksum of newly generated archive
	base := md5.New()
	f, err := os.Open(tarPath)
	if err != nil {
		return errors.Wrapf(err, "error opening tar file %q", tarPath)
	}
	defer f.Close()
	_, err = io.Copy(base, f)
	if err != nil {
		return errors.Wrapf(err, "error generating base checksum")
	}
	// Compare every archive in target path to checksum of new archive
	// Delete archives with same checksum
	err = filepath.Walk(targetPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error with dir walker")
		}
		// Skip dirs and the new archive
		if fi.IsDir() || path == tarPath {
			return nil
		}
		// Generate checksum of file
		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "error opening file %q", path)
		}
		defer f.Close()
		sum := md5.New()
		_, err = io.Copy(sum, f)
		if err != nil {
			return errors.Wrap(err, "error generating checksum")
		}
		// Delete file if its checksum matches the checksum of the new archive
		if fmt.Sprintf("%x", base.Sum(nil)) == fmt.Sprintf("%x", sum.Sum(nil)) {
			err = os.Remove(path)
			if err != nil {
				return errors.Wrapf(err, "error deleting file %q", path)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error walking target directory")
	}
	return nil
}
