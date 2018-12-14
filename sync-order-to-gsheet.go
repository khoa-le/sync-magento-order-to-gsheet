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
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	City      string   `json:"city"`
	Region    string   `json:"region"`
	Street    []string `json:"street"`
	Telephone string   `json"telephone"`
}
type shippingAddress struct {
	City      string   `json:"city"`
	Company   string   `json:"company"`
	Country   string   `json:"country"`
	Email     string   `json:"email"`
	Region    string   `json:"region"`
	Street    []string `json:"street"`
	Telephone string   `json:"telephone"`
}
type shipping struct {
	ShippingAddress shippingAddress `json:"address"`
	Method          string          `json:"method"`
}
type shippingAssignment struct {
	Shipping shipping `json:"shipping"`
}
type giftMessage struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}
type extensionAttributes struct {
	ShippingAssignments []shippingAssignment `json:"shipping_assignments"`
	GiftMessage         *giftMessage         `json:"gift_message"`
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
	ParentItemId    int    `json:"parent_item_id"`
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
	DiscountCode        string              `json:"coupon_code"`
	OrderItems          []orderItem         `json:"items"`
	Payment             payment             `json:"payment"`
	ExtensionAttributes extensionAttributes `json:"extension_attributes"`
	BillingAddress      billingAddress      `json:"billing_address"`
}
type responseOrder struct {
	Total int     `json:"total_count"`
	Items []order `json:"items"`
}
type customAttribute struct {
	AttributeCode string `json:"attribute_code"`
	Value         string `json:"value"`
}
type product struct {
	ID               int               `json:"id"`
	Sku              string            `json:"sku"`
	CustomAttributes []customAttribute `json:"custom_attributes"`
	BtjCode          string
}

func getListOrder(currentMonth string) responseOrder {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	res := responseOrder{}

	//currentMonth := time.Now().Format("2006-01")

	condition := "searchCriteria[filter_groups][0][filters][0][field]=created_at&searchCriteria[filter_groups][0][filters][0][value]=" + currentMonth + "-01%2000:00:00&searchCriteria[filter_groups][0][filters][0][condition_type]=from&searchCriteria[filter_groups][1][filters][1][field]=created_at&searchCriteria[filter_groups][1][filters][1][value]=" + currentMonth + "-31%2023:59:59&searchCriteria[filter_groups][1][filters][1][condition_type]=to"
	//fmt.Printf(MagentoBaseRestApi + "V1/orders?" + condition)
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

func getProduct(sku string) product {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	res := product{}

	request, _ := http.NewRequest("GET", MagentoBaseRestApi+"V1/products/"+sku, nil)
	request.Header.Set("Authorization", "Bearer "+MagentoBearer)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("The http request failed")
	} else {
		data, _ := ioutil.ReadAll(response.Body)

		json.Unmarshal([]byte(data), &res)
	}

	for i := 0; i < len(res.CustomAttributes); i++ {
		attr := res.CustomAttributes[i]
		if attr.AttributeCode == "btj_code" {
			res.BtjCode=attr.Value
			break
		}
	}

	return res
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	spreadsheetId := os.Getenv("GOOGLE_SHEET_ID")

	srv, err := spreadsheet.NewService()
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	currentMonth := time.Now().Format("2006-01")
	//currentMonth = "2018-05"

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
		"Billing Address",
		"Shipping Method",
		"Shipping Address",
		"Discount code",
		"Order Value",
		"Paid Amount",
		"Due Amount",
		"Payment Transaction ID",
		"Payment method",
		"Gift Message",
	})

	res := getListOrder(currentMonth)
	if res.Total > 0 {
		for i := 0; i < res.Total; i++ {
			//fmt.Printf(res.Items[i].CustomerFirstName);
			item := res.Items[i]
			skus := make([]string, 0)
			prices := make([]string, 0)
			quantities := make([]string, 0)

			for j := 0; j < len(item.OrderItems); j++ {
				if item.OrderItems[j].Price > 0 || (item.OrderItems[j].Price == 0 && item.OrderItems[j].ParentItemId == 0) {
					productSaleOrderItem := getProduct(item.OrderItems[j].SKU)
					if productSaleOrderItem.ID > 0 {
						skus = append(skus, item.OrderItems[j].SKU+"("+productSaleOrderItem.BtjCode+")")
					} else {
						skus = append(skus, item.OrderItems[j].SKU)
					}

					prices = append(prices, strconv.Itoa(item.OrderItems[j].Price))
					quantities = append(quantities, strconv.Itoa(item.OrderItems[j].QuantityOrdered))
				}
			}
			address := ""
			shippingMethod := ""
			if len(item.ExtensionAttributes.ShippingAssignments) > 0 {
				shipping := item.ExtensionAttributes.ShippingAssignments[0].Shipping

				if shipping.Method == "smilestoredelivery_smilestoredelivery" {
					shippingMethod = "Nhận Tại Cửa Hàng"
					address = shipping.ShippingAddress.Company
				} else {
					shippingMethod = "Giao Hàng Tận Nơi"
					address = strings.Join(shipping.ShippingAddress.Street, ", ")
					address += ", " + shipping.ShippingAddress.City + ", " + shipping.ShippingAddress.Region + " (Phone: " + shipping.ShippingAddress.Telephone + ")"
				}
			}

			billingAddress := "First Name: " + item.BillingAddress.FirstName + ", Last Name: " + item.BillingAddress.LastName + ", Phone: " + item.BillingAddress.Telephone + " \n"
			billingAddress += "Address: " + strings.Join(item.BillingAddress.Street, ", ")
			billingAddress += ", " + item.BillingAddress.City + ", " + item.BillingAddress.Region

			giftInfo := ""
			if item.ExtensionAttributes.GiftMessage != nil {
				giftMessage := item.ExtensionAttributes.GiftMessage
				giftInfo += "From: " + giftMessage.Sender + "\n"
				giftInfo += "To: " + giftMessage.Recipient + "\n"
				giftInfo += "Message: " + giftMessage.Message + "\n"
			}
			row := []interface{}{
				item.CreatedAt,
				item.IncrementId,
				item.Status,
				item.CustomerFirstName,
				item.CustomerLastName,
				item.CustomerEmail,
				item.BillingAddress.Telephone,
				strings.Join(skus, ","),
				strings.Join(quantities, ","),
				strings.Join(prices, ","),
				billingAddress,
				shippingMethod,
				address,
				item.DiscountCode,
				item.GrandTotal,
				item.TotalPaid,
				item.TotalDue,
				item.Payment.TransactionId,
				item.Payment.Method,
				giftInfo,
			}
			vr.Values = append(vr.Values, row)
		}
	}
	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, currentMonth+"!A1", &vr).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}

}
