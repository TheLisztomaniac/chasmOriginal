package main

import (
	"log"
	"os"

	"gopkg.in/fsnotify.v1"
)

// StartWatching a path indefinitely.
func StartWatching(path string, subDirs map[string]bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	for sub, _ := range subDirs {
		watcher.Add(sub)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				isDir := isDir(event.Name)

				if event.Op&fsnotify.Create == fsnotify.Create {
					AddFile(event.Name)
					if isDir {
						watcher.Add(event.Name)
					}
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					AddFile(event.Name)
					if isDir {
						watcher.Add(event.Name)
					}
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					DeleteFile(event.Name)
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					DeleteFile(event.Name)
				}

			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	<-done
}

func isDir(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}

	fi, err := file.Stat()
	if err != nil {
		return false
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	default:
		return false
	}
}
