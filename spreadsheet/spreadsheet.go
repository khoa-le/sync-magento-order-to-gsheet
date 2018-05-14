package spreadsheet

import (
	"net/http"
	"fmt"
	"log"
	"os"
	"encoding/json"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/sheets/v4"
	"io/ioutil"
	"golang.org/x/oauth2/google"
)

var Service *sheets.Service

// Retrieve a token, saves the token, then returns the generated client.
func GetClient(config *oauth2.Config) *http.Client {
	pwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	tokFile := pwd + "/token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func CreateNewSpreadsheet() (*sheets.Spreadsheet, error) {
	property := sheets.SpreadsheetProperties{
		Title: "New Orders",
	}
	spread := &sheets.Spreadsheet{
		Properties: &property,
	}
	srv, _ := NewService()
	return srv.Spreadsheets.Create(spread).Do()
}

func CheckExistSheet(SpreadSheetID string, SheetTitle string) bool {
	result := true
	srv, _ := NewService()
	_ , err := srv.Spreadsheets.Values.Get(SpreadSheetID, SheetTitle+"!A1").Do()
	if err != nil {
		result=false
	}
	return result
}
func CreateNewSheet(SpreadSheetID string, SheetTitle string) *sheets.Spreadsheet {
	property := sheets.SheetProperties{
		Title: SheetTitle,
	}
	addS := &sheets.AddSheetRequest{
		//SpreadsheetId:spreadsheetId,
		Properties: &property,
	}

	requests := []*sheets.Request{} // TODO: Update placeholder value.
	request := sheets.Request{
		AddSheet: addS,
	}
	requests = append(requests, &request)

	rb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	srv, _ := NewService()
	resp, err := srv.Spreadsheets.BatchUpdate(SpreadSheetID, rb).Do()
	if err != nil {
		log.Fatal(err)
	}
	return resp.UpdatedSpreadsheet
}

func NewService() (*sheets.Service, error) {
	pwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	b, err := ioutil.ReadFile(pwd + "/client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved client_secret.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := GetClient(config)
	service, error := sheets.New(client)
	if error != nil {
		log.Fatalf("Unable to create new client: %v", error)
	}
	Service = service
	return service, error
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}
