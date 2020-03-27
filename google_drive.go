package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"time"

	"github.com/fatih/color"
	"github.com/toqueteos/webbrowser"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type GDriveStore struct {
	Config     oauth2.Config `json:"oauth_config"`
	OAuthToken oauth2.Token  `json:"oauth_token"`
	UserID     string        `json:"user_id"`
}

// Setup GDrive
func (g *GDriveStore) Setup() bool {
	config, err := getConfig()

	if err != nil {
		color.Red("Unable to parse client secret file to config: %v", err)
		return false
	}

	tok, err := getGDriveTokenFromWeb(config)
	if err != nil {
		color.Red("Unable to get client token: %v", err)
		return false
	}

	// set the oauth info
	g.Config = *config
	g.OAuthToken = *tok

	ctx := context.Background()
	client := config.Client(ctx, &g.OAuthToken)

	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return false
	}

	account, err := svc.About.Get().Fields("user").Do()
	if err != nil {
		color.Red("Unable to retrieve drive information %v", err)
		return false
	}
	userID := account.User.PermissionId

	for _, gds := range preferences.GDriveStores {
		if gds.UserID == userID {
			color.Red("Google Drive Account for %v (%v) already exists.", account.User.DisplayName, account.User.EmailAddress)
			return false
		}
	}

	g.UserID = userID
	return true
}

func (g GDriveStore) Upload(share Share) {
	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)

	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return
	}

	fmt.Print(color.MagentaString("Uploading GoogleDrive/%s...", share.SID))

	// delete existing share
	deleteFilesForShareID(share.SID, svc)

	// now create and upload share
	file := drive.File{}
	now, err := time.Now().MarshalText()
	file.ModifiedTime = string(now)
	file.Name = string(share.SID)
	file.Parents = []string{"appDataFolder"}

	_, err = svc.Files.Create(&file).Media(bytes.NewReader(share.Data)).Do()
	if err != nil {
		color.Red("GoogleDrive/%s upload failed: %v", share.SID, err)
	} else {
		//print check mark
		fmt.Print(color.MagentaString("\u2713\n"))
	}
}

func (g GDriveStore) Delete(sid ShareID) {
	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)

	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return
	}

	fmt.Print(color.YellowString("Deleting GoogleDrive/%s...", sid))
	// delete existing share
	deleteFilesForShareID(sid, svc)

	//print check mark
	fmt.Print(color.YellowString("\u2713\n"))
}

//Restore downloads shares to local restore path
func (g GDriveStore) Restore() string {
	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)

	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return ""
	}

	restoreDir, err := ioutil.TempDir("", "chasm_gdrive_restore")
	if err != nil {
		color.Red("Error cannot create temp dir: %v", err)
		return ""
	}

	// download all files
	// get all chasm files from drive
	r, err := svc.Files.List().Spaces("appDataFolder").Do()
	if err != nil {
		color.Red("Unable to iterate names %v", err)
		return ""
	}

	color.Yellow("Downloading shares from Google Drive...")

	for _, i := range r.Files {

		// export file
		resp, err := svc.Files.Get(i.Id).Download()
		if err != nil {
			color.Yellow("Error downloading file %s: %v", i.Name, err)
			continue
		}
		defer resp.Body.Close()

		// read file bytes
		fileBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			color.Yellow("Error reading downloaded bytes %s: %v", i.Name, err)
			continue
		}

		// write file to temp dir
		ioutil.WriteFile(path.Join(restoreDir, i.Name), fileBytes, 0770)
		fmt.Println("\t - got share ", i.Name)
	}

	return restoreDir
}

func (g GDriveStore) Description() string {
	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)
	label := "Google Drive Store"
	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return label
	}

	// get all chasm files from drive
	r, err := svc.Files.List().Spaces("appDataFolder").Do()
	if err != nil {
		color.Red("Unable to iterate names %v", err)
		return label
	}

	account, err := svc.About.Get().Fields("user").Do()
	if err != nil {
		color.Red("Unable to retrieve drive information %v", err)
		return label
	}

	label = fmt.Sprintf("Google Drive Store: %v (%v)", account.User.DisplayName, account.User.EmailAddress)

	for _, i := range r.Files {
		label += fmt.Sprintf("\n\t%s %s", color.YellowString("-"), i.Name)
	}

	return label
}

func (g GDriveStore) ShortDescription() string {
	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)
	label := "Google Drive Store"
	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return label
	}

	account, err := svc.About.Get().Fields("user").Do()
	if err != nil {
		color.Red("Unable to retrieve drive information %v", err)
		return label
	}

	return fmt.Sprintf("Google Drive Store: %v (%v)", account.User.DisplayName, account.User.EmailAddress)
}

// Clean deletes all shares from the folder store
func (g GDriveStore) Clean() {

	ctx := context.Background()
	config := &g.Config
	client := config.Client(ctx, &g.OAuthToken)

	svc, err := drive.New(client)
	if err != nil {
		color.Red("Unable to retrieve drive Client %v", err)
		return
	}

	r, err := svc.Files.List().Spaces("appDataFolder").Do()
	if err != nil {
		color.Red("Unable to search for files to delete: %v", err)
		return
	}

	for _, i := range r.Files {
		color.Yellow("Removing Google Drive: %v", i.Name)
		svc.Files.Delete(i.Id).Do()
	}
}

/// MARK: Helper Methods ///
func getConfig() (*oauth2.Config, error) {
	json, err := ioutil.ReadFile(GoogleDriveClientSecret)
	if err != nil {
		log.Fatalf("Unable to read google client secret file: %v", err)
	}
	return google.ConfigFromJSON(json, drive.DriveAppdataScope)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getGDriveTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	webbrowser.Open(authURL)

	color.Yellow("Enter Auth Code: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		color.Red("Unable to read authorization code %v", err)
		return nil, err
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		color.Red("Unable to retrieve token from web %v", err)
		return tok, err
	}

	return tok, nil
}

func deleteFilesForShareID(sid ShareID, svc *drive.Service) {
	// get all chasm files from drive
	q := fmt.Sprintf("name = '%s'", string(sid))

	r, err := svc.Files.List().Spaces("appDataFolder").Q(q).Do()
	if err != nil {
		color.Red("Unable to search for files to delete: %v", err)
		return
	}

	for _, i := range r.Files {
		svc.Files.Delete(i.Id).Do()
	}
}
