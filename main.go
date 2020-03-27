package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"sync"

	"github.com/codegangsta/cli"
	"github.com/fatih/color"
)

/// chasm commands ///

func loadChasm(c *cli.Context) error {
	CreateOrLoadChasmDir(chasmRoot)
	return nil
}

func startChasm(c *cli.Context) error {
	loadChasm(c)

	if preferences.NeedSetup() {
		color.Red("Warning: not enough services.")
		return nil
	}

	// start the watcher
	color.Green("Starting chasm. Listening on %s", preferences.root)
	StartWatching(preferences.root, preferences.DirMap)

	return nil
}

func statusChasm(c *cli.Context) error {
	loadChasm(c)

	color.Green("Cloud stores:")
	for i, cs := range preferences.AllCloudStores() {
		fmt.Println(color.GreenString("%v)", i+1), cs.Description())
	}
	if preferences.NeedSetup() {
		color.Red("Warning: not enough services.")
	}

	return nil
}

func restoreChasm(c *cli.Context) error {
	loadChasm(c)

	if preferences.NeedSetup() {
		color.Red("Warning: not enough services. Cannot Restore.")
		return nil
	}

	color.Green("Preparing to restore chasm to %s", preferences.root)
	Restore()

	return nil
}

func removeChasm(c *cli.Context) error {
	loadChasm(c)

	numStores := preferences.RegisteredServices()
	if numStores == 0 {
		color.Red("There are no cloud stores to delete.")
		return nil
	}

	color.Green("Cloud stores:")
	for i, cs := range preferences.AllCloudStores() {
		fmt.Println(color.GreenString("%v)", i+1), cs.ShortDescription())
	}
	color.Cyan("Enter the number of the store you would like to remove:")

	var d int
	for true {
		_, err := fmt.Scanf("%d", &d)
		if err != nil || d < 0 || d > numStores {
			color.Red("Please enter a number between %v and %v", 1, numStores)
		} else {
			break
		}
	}

	if d <= len(preferences.FolderStores) {
		ind := d - 1
		preferences.FolderStores[ind].Clean()
		preferences.FolderStores = append(preferences.FolderStores[:ind], preferences.FolderStores[ind+1:]...)
		color.Yellow("Deleting Folder Store...")
	} else if d <= len(preferences.FolderStores)+len(preferences.GDriveStores) {
		ind := d - 1 - len(preferences.FolderStores)
		preferences.GDriveStores[ind].Clean()
		preferences.GDriveStores = append(preferences.GDriveStores[:ind], preferences.GDriveStores[ind+1:]...)
		color.Yellow("Deleting Google Drive Store...")
	}

	preferences.Save()
	syncChasm(c)

	return nil
}

func cleanChasm(c *cli.Context) error {
	loadChasm(c)
	var wg sync.WaitGroup
	for _, cs := range preferences.AllCloudStores() {
		wg.Add(1)
		go func(c CloudStore) {
			defer wg.Done()
			c.Clean()
			color.Green("Done cleaning %v", c.ShortDescription())
		}(cs)
	}
	wg.Wait()

	return nil
}

func syncChasm(c *cli.Context) error {
	color.Green("Clean:")
	cleanChasm(c)
	color.Green("Done cleaning.\nBeginning sync:")

	if preferences.NeedSetup() {
		color.Red("Error: not enough services. Cannot sync.")
		return nil
	}

	files, _ := ioutil.ReadDir(preferences.root)
	currentFileMap := make(map[string]bool)
	for _, f := range files {
		if f.Name() == chasmPrefFile {
			continue
		}
		path := path.Join(preferences.root, f.Name())
		currentFileMap[path] = true
		AddFile(path)
	}

	// remove invalid entries in existing file map
	for filePath, _ := range preferences.FileMap {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			delete(preferences.FileMap, filePath)
		}
	}

	preferences.Save()
	AddFile(path.Join(preferences.root, chasmPrefFile))

	color.Green("Done syncing.")

	return nil
}

//MARK: Add Handlers

func addFolder(c *cli.Context) error {
	loadChasm(c)
	var folderStore FolderStore

	if len(c.Args()) < 1 {
		color.Red("Error: missing folder path")
		return nil
	}

	folderStore.Path = c.Args()[0]
	if !folderStore.Setup() {
		color.Red("(Cloud Store) Folder Store: setup incomplete.")
		return nil
	}

	preferences.FolderStores = append(preferences.FolderStores, folderStore)
	preferences.Save()

	color.Green("Success! Added folder store: %s", folderStore.Path)
	return nil
}

func addDrive(c *cli.Context) error {
	loadChasm(c)
	var gdrive GDriveStore

	if (&gdrive).Setup() == false {
		color.Red("(Cloud Store) Google Drive: setup incomplete.")
		return nil
	}

	// only 1 gdrive store
	preferences.GDriveStores = append(preferences.GDriveStores, gdrive)
	preferences.Save()

	color.Green("Success! Added Google Drive Store.")

	return nil
}

/// Cli toolchain ///
var chasmRoot string

func main() {
	app := cli.NewApp()

	app.Name = color.GreenString("chasm")
	app.Usage = color.GreenString("A secret-sharing based secure cloud backup solution.")
	app.EnableBashCompletion = true
	app.Version = "0.1"

	usr, _ := user.Current()
	defaultRoot := path.Join(usr.HomeDir, "Chasm")
	chasmRoot = defaultRoot

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "root, r",
			Value:       defaultRoot,
			Usage:       "Destination of the Chasm secure folder.",
			Destination: &chasmRoot,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "start",
			Aliases: nil,
			Usage:   "Start running chasm.",
			Action:  startChasm,
		},
		{
			Name:    "status",
			Aliases: nil,
			Usage:   "Prints out the current Chasm setup.",
			Action:  statusChasm,
		},
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "Add a new cloud store to chasm.",
			Subcommands: []cli.Command{
				{
					Name:   "folder",
					Usage:  "add folder",
					Action: addFolder,
				},
				{
					Name:   "gdrive",
					Usage:  "add google drive",
					Action: addDrive,
				},
			},
		},
		{
			Name:    "restore",
			Aliases: nil,
			Usage:   "Restores chasm after repeating setup.",
			Action:  restoreChasm,
		},
		{
			Name:    "remove",
			Aliases: nil,
			Usage:   "Removes a cloud store.",
			Action:  removeChasm,
		},
		{
			Name:    "clean",
			Aliases: nil,
			Usage:   "Deletes all shares in cloud stores",
			Action:  cleanChasm,
		},
		{
			Name:    "sync",
			Aliases: nil,
			Usage:   "Clean cloud stores, sync all items in Chasm folder by secret-sharing.",
			Action:  syncChasm,
		},
	}

	app.Run(os.Args)
}
