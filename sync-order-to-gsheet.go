package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
	"github.com/joho/godotenv"
)

const MAGENTO_BEARER_TOKEN string = "mrl88n4dylj68amehmp12sysno76jhn7"
const MAGENTO_BASE_REST_API string = "http://dev.precita.vn/rest/"

type billingAddress struct {
	Telephone string `json:"telephone"`
}
type shippingAddress struct {
	Company string `json:"company"`
}
type shipping struct {
	ShippingAddress shippingAddress `json:"address"`
}
type shippingAssignment struct {
	Shipping shipping `json:"shipping"`
}
type extensionAttributes struct {
	ShippingAssignments []shippingAssignment `json:"shipping_assignments"`
}
type payment struct {
	Method string `json:"method"`
}
type orderItem struct {
	SKU             string `json:"sku"'`
	Price           int    `json:"price"`
	QuantityOrdered int    `json:"qty_ordered"`
	ProductType     string `json:"product_type"`
}
type order struct {
	EntityId            int                 `json:"entity_id"`
	IncrementId         string              `json:"increment_id"`
	CreatedAt           string              `json:"created_at"`
	GrandTotal          int                 `json:"grand_total"`
	TotalPaid           int                 `json:"total_paid"`
	TotalDue            int                 `json:"total_due"`
	CustomerFirstName   string              `json:"customer_firstname"`
	CustomerLastName    string              `json:"customer_lastname"`
	CustomerEmail       string              `json:"customer_email"`
	Address             billingAddress      `json:"billing_address"`
	DiscountCode        string              `json:"coupon_code"`
	OrderItems          []orderItem         `json:"items"`
	Payment             payment             `json:"payment"`
	ExtensionAttributes extensionAttributes `json:"extension_attributes"`
}
type responseOrder struct {
	Total int     `json:"total_count"`
	Items []order `json:"items"`
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
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
func getSheet() {

}

func getListOrder() responseOrder {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	res := responseOrder{}
	request, _ := http.NewRequest("GET", MagentoBaseRestApi+"V1/orders?searchCriteria", nil)
	request.Header.Set("Authorization", "Bearer "+MagentoBearer)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("The http request failed")
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &res)
	}
	return res
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	spreadsheetId := os.Getenv("GOOGLE_SHEET_ID")

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved client_secret.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}


	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	//readRange := "Sheet 1!A2:E"
	//resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	//if err != nil {
	//	log.Fatalf("Unable to retrieve data from sheet: %v", err)
	//}
	//
	//if len(resp.Values) == 0 {
	//	fmt.Println("No data found.")
	//} else {
	//	fmt.Println("Mechanic, BTJ code, Vendor Code:")
	//	for _, row := range resp.Values {
	//		// Print columns A and E, which correspond to indices 0 and 4.
	//		fmt.Printf("%s, %s, %s\n", row[0], row[1], row[2])
	//	}
	//}
	var vr sheets.ValueRange

	//row := []interface{}{"Khoa"}
	//vr.Values = append(vr.Values, row)
	res := getListOrder()
	if res.Total > 0 {
		for i := 0; i < res.Total; i++ {
			//fmt.Printf(res.Items[i].CustomerFirstName);
			item := res.Items[i]
			skus := make([]string, 0)
			prices := make([]string, 0)
			quantities := make([]string, 0)

			for j := 0; j < len(item.OrderItems); j++ {
				if item.OrderItems[j].Price > 0 {
					skus = append(skus, item.OrderItems[j].SKU)
					prices = append(prices, strconv.Itoa(item.OrderItems[j].Price))
					quantities = append(quantities, strconv.Itoa(item.OrderItems[j].QuantityOrdered))
				}
			}
			//fmt.Printf("%+v\n", item.ExtensionAttributes.ShippingAssignments);
			company := ""
			if len(item.ExtensionAttributes.ShippingAssignments) > 0 {
				shipping := item.ExtensionAttributes.ShippingAssignments[0].Shipping
				company = shipping.ShippingAddress.Company
			}

			row := []interface{}{
				item.CreatedAt,
				item.IncrementId,
				item.CustomerFirstName,
				item.CustomerLastName,
				item.CustomerEmail,
				item.Address.Telephone,
				strings.Join(skus, ","),
				strings.Join(quantities, ","),
				strings.Join(prices, ","),
				company,
				item.DiscountCode,
				item.GrandTotal,
				item.TotalPaid,
				item.TotalDue,
				"",
				item.Payment.Method,
			}
			vr.Values = append(vr.Values, row)
		}
	}
	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, "Sheet1!A2", &vr).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}

}
