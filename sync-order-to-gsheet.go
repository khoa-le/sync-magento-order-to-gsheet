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
	"go-sheet/spreadsheet"

	"github.com/joho/godotenv"
	"google.golang.org/api/sheets/v4"
	"time"
)

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
	Method        string `json:"method"`
	TransactionId string `json:"last_trans_id"`
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
	Status              string              `json:"status"`
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

func getListOrder() responseOrder {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	res := responseOrder{}

	currentMonth := time.Now().Format("2006-01")

	condition := "searchCriteria[filter_groups][0][filters][0][field]=created_at&searchCriteria[filter_groups][0][filters][0][value]=" + currentMonth + "-01%2000:00:00&searchCriteria[filter_groups][0][filters][0][condition_type]=from&searchCriteria[filter_groups][1][filters][1][field]=created_at&searchCriteria[filter_groups][1][filters][1][value]=" + currentMonth + "-31%2023:59:59&searchCriteria[filter_groups][1][filters][1][condition_type]=to"
	request, _ := http.NewRequest("GET", MagentoBaseRestApi+"V1/orders?"+condition, nil)
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

	srv, err := spreadsheet.NewService()
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	currentMonth := time.Now().Format("2006-01")
	existedSheet := spreadsheet.CheckExistSheet(spreadsheetId, currentMonth)
	if !existedSheet {
		_ = spreadsheet.CreateNewSheet(spreadsheetId, currentMonth)
	}
	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{
		"Order Date (time zone GST)",
		"Order number",
		"Status",
		"First Name",
		"Last Name",
		"Email",
		"Phone",
		"SKU",
		"Quantity",
		"Item Price",
		"Store",
		"Discount code",
		"Order Value",
		"Paid Amount",
		"Due Amount",
		"Payment Transaction ID",
		"Payment method",
	})

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
			company := ""
			if len(item.ExtensionAttributes.ShippingAssignments) > 0 {
				shipping := item.ExtensionAttributes.ShippingAssignments[0].Shipping
				company = shipping.ShippingAddress.Company
			}
			row := []interface{}{
				item.CreatedAt,
				item.IncrementId,
				item.Status,
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
				item.Payment.TransactionId,
				item.Payment.Method,
			}
			vr.Values = append(vr.Values, row)
		}
	}
	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, currentMonth+"!A1", &vr).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}

}
