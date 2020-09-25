package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

func writeTo(w io.Writer, val string) error {
	_, err := w.Write([]byte(val))
	return err
}

func genIndents(level, countDir int) (res string) {
	for i := 0; i < countDir; i++ {
		res += "│\t"
	}
	countIndents := level - countDir
	for i := 0; i < countIndents; i++ {
		res += "\t"
	}
	return
}

func printFileName(w io.Writer, name, prefix string, level, countLast int, size int64) error {
	var sizeOrEmpty string
	if size == 0 {
		sizeOrEmpty = "empty"
	} else {
		sizeOrEmpty = fmt.Sprintf("%db", size)
	}
	fileRepr := fmt.Sprintf("%s%s%s (%s)\n", genIndents(level, countLast), prefix, name, sizeOrEmpty)
	err := writeTo(w, fileRepr)
	if err != nil {
		return err
	}
	return nil
}

func printDirName(w io.Writer, name, prefix string, level, countLast int) error {
	str := fmt.Sprintf("%s%s%s\n", genIndents(level, countLast), prefix, name)
	err := writeTo(w, str)
	if err != nil {
		return err
	}
	return nil
}

func printDir(w io.Writer, path string, level int, lastCount int, isLast bool, withFiles bool) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	lastDirOrFileIndex := len(files) - 1
	if !withFiles {
		for i, file := range files {
			if file.IsDir() {
				lastDirOrFileIndex = i
			}
		}
	}

	for index, file := range files {
		last := index == lastDirOrFileIndex

		var prefix string
		if last {
			prefix = "└───"
		} else {
			prefix = "├───"
		}

		countIndents := level
		if isLast {
			countIndents -= lastCount
		}

		if file.IsDir() {
			err := printDirName(w, file.Name(), prefix, level, countIndents)
			if err != nil {
				return err
			}
			level++
			if last {
				lastCount++
			}
			err = printDir(w, path+string(filepath.Separator)+file.Name(), level, lastCount, last, withFiles)
			if err != nil {
				return err
			}
			level--
			continue
		}

		if withFiles {
			err := printFileName(w, file.Name(), prefix, level, countIndents, file.Size())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func dirTree(w io.Writer, path string, withFiles bool) error {
	err := printDir(w, path, 0, 0, false, withFiles)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
