package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
    "net/http"
	"io"

	"github.com/gobwas/glob"
)

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, ErrEnvVarEmpty
	}
	return v, nil
}

func getenvInt(key string, def int) (int, error) {
	s, err := getenvStr(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func getenvBool(key string) (bool, error) {
	s, err := getenvStr(key)
	if err != nil {
		return true, err
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return true, err
	}
	return v, nil
}

func check(e error) {
	if e != nil {
		log.Fatalf("error: %v", e)
	}
}

func listFiles(include string, exclude string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		if doesFileMatch(path, include, exclude) {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList, err
}

func doesFileMatch(path string, include string, exclude string) bool {
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		includeGlob := glob.MustCompile(include)
		excludeGlob := glob.MustCompile(exclude)
		return includeGlob.Match(path) && !excludeGlob.Match(path)
	}
	return false
}


func downloadFile(filepath string, url string) (err error) {
	out, err := os.Create(filepath)
	if err != nil  {
	  return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
	  return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
	  return fmt.Errorf("bad status: %s", resp.Status)
	}
  
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
	  return err
	}
  
	return nil
}

func findAndReplace(path string, find string, target string, replace bool) (bool, error) {
	if find != target {
		read, readErr := ioutil.ReadFile(path)
		check(readErr)

		newContents := ""
		re := regexp.MustCompile(find)

		targetDir := filepath.Dir(path)
		matches := re.FindAllString(string(read), -1)
		for _, v := range matches {
			newTarget := filepath.Join(targetDir, re.ReplaceAllString(v, target))
			fmt.Println(v + "-->" + newTarget)
			downloadErr := downloadFile(newTarget, v)
			check(downloadErr)
		}

		if replace {
			newContents = re.ReplaceAllString(string(read), target)
			if newContents != string(read) {
				writeErr := ioutil.WriteFile(path, []byte(newContents), 0)
				check(writeErr)
				return true, nil
			}
		}
		
		return len(matches)>0, nil
	}

	return false, nil
}

func main() {
	include, _ := getenvStr("INPUT_INCLUDE")
	exclude, _ := getenvStr("INPUT_EXCLUDE")
	find, findErr := getenvStr("INPUT_FIND")
	target, targetErr := getenvStr("INPUT_TARGET")
	replace, replaceErr := getenvBool("INPUT_REPLACE")

	if findErr != nil {
		panic(errors.New("gha-download-images: expected with.find to be a string"))
	}

	if targetErr != nil {
		panic(errors.New("gha-download-images: expected with.replace to be a string"))
	}

	if replaceErr != nil {
		replace = true
	}

	files, filesErr := listFiles(include, exclude)
	check(filesErr)

	modifiedCount := 0

	for _, path := range files {
		modified, findAndReplaceErr := findAndReplace(path, find, target, replace)
		check(findAndReplaceErr)

		if modified {
			modifiedCount++
		}
	}

	fmt.Println(fmt.Sprintf(`::set-output name=modifiedFiles::%d`, modifiedCount))
}
