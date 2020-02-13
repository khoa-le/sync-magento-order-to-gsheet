package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/khoa-le/sync-magento-order-to-gsheet/spreadsheet"
	"google.golang.org/api/sheets/v4"
)

type billingAddress struct {
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	City      string   `json:"city"`
	Region    string   `json:"region"`
	Street    []string `json:"street"`
	Telephone string   `json:"telephone"`
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
	FulfillmentStatus   string               `json:"fulfillment_status"`
	StockChecking       string               `json:"stock_checking"`
}
type payment struct {
	Method        string `json:"method"`
	TransactionId string `json:"last_trans_id"`
}
type orderItem struct {
	SKU             string `json:"sku"`
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
	ShippingDescription string              `json:"shipping_description"`
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
type StoreNoteRequest struct {
	StoreNote StoreNote `json:"storeNote"`
}

type StoreNote struct {
	SalesOrderId          int    `json:"sales_order_id"`
	SalesOrderIncrementId string `json:"sales_order_increment_id"`
	Note                  string `json:"note"`
	Status                string `json:"status"`
	ErplyInvoiceIds       string `json:"erply_invoice_ids"`
}

func updateStoreNote(orderId string, note string, status string, erplyInvoiceIds string) StoreNote {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	storeNoteRequest := &StoreNoteRequest{
		StoreNote: StoreNote{
			Note:            note,
			Status:          status,
			ErplyInvoiceIds: erplyInvoiceIds,
		},
	}

	// marshal storeNote to json
	storeNoteRequestJson, err := json.Marshal(storeNoteRequest)
	if err != nil {
		panic(err)
	}
	request, _ := http.NewRequest("PUT", MagentoBaseRestApi+"V1/orderManagement/orderId/"+orderId+"/updateStoreNote", bytes.NewBuffer(storeNoteRequestJson))
	request.Header.Set("Authorization", "Bearer "+MagentoBearer)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	response, err := client.Do(request)
	res := StoreNote{}
	if err != nil {
		fmt.Printf("The http request failed")
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal([]byte(data), &res)
		fmt.Printf("data: %v", res)
	}
	return res
}

func getListOrder(currentMonth string) responseOrder {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	MagentoBearer := os.Getenv("MAGENTO_BEARER_TOKEN")
	MagentoBaseRestApi := os.Getenv("MAGENTO_BASE_REST_API")

	res := responseOrder{}

	condition := "searchCriteria[filter_groups][0][filters][0][field]=created_at&searchCriteria[filter_groups][0][filters][0][value]=" + currentMonth + "-01%2000:00:00&searchCriteria[filter_groups][0][filters][0][condition_type]=from&searchCriteria[filter_groups][1][filters][1][field]=created_at&searchCriteria[filter_groups][1][filters][1][value]=" + currentMonth + "-31%2023:59:59&searchCriteria[filter_groups][1][filters][1][condition_type]=to"
	// fmt.Printf(MagentoBaseRestApi + "V1/orders?" + condition)
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
	// fmt.Printf("result get list order: %v", res)
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
			res.BtjCode = attr.Value
			break
		}
	}

	return res
}
func getAllStoreNote(srv *sheets.Service, spreadsheetId string, currentMonth string, dataRange string) ([]string, []StoreNote) {
	var listStoreNote []StoreNote
	var listHash []string
	//Do get
	readRange := currentMonth + dataRange
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}
	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range resp.Values {

			var hash string
			if len(row) > 0 {
				hash = fmt.Sprintf("%v", row[0])
			}
			listHash = append(listHash, hash)

			var status string
			if len(row) > 1 {
				status = fmt.Sprintf("%v", row[1])
			}

			var note string
			if len(row) > 2 {
				note = fmt.Sprintf("%v", row[2])
			}

			var erplyInvoiceIds string
			if len(row) > 3 {
				erplyInvoiceIds = fmt.Sprintf("%v", row[3])
			}

			listStoreNote = append(listStoreNote, StoreNote{Note: note, Status: status, ErplyInvoiceIds: erplyInvoiceIds})
		}
	}
	return listHash, listStoreNote
}

func getHash(currentHash string, storeNote StoreNote) (bool, string) {
	hashtring := storeNote.Status + storeNote.Note + storeNote.ErplyInvoiceIds
	hasher := md5.New()
	hasher.Write([]byte(hashtring))
	hashStoreUpdateString := hex.EncodeToString(hasher.Sum(nil))
	fmt.Printf("New Hash %v", hashStoreUpdateString)
	fmt.Printf("Compare:::%v", currentHash != hashStoreUpdateString)
	return currentHash != hashStoreUpdateString, hashStoreUpdateString
}
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env.bak file")
	}
	spreadsheetId := os.Getenv("GOOGLE_SHEET_ID")
	storeNoteDataRange := os.Getenv("STORE_NOTE_DATA_RANGE")

	srv, err := spreadsheet.NewService()
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	currentMonth := time.Now().Format("2006-01")
	// currentMonth = "2019-12"
	// updateStoreNote("5842")
	// return

	existedSheet := spreadsheet.CheckExistSheet(spreadsheetId, currentMonth)
	if !existedSheet {
		_ = spreadsheet.CreateNewSheet(spreadsheetId, currentMonth)
	}

	listHash, allStoreNotes := getAllStoreNote(srv, spreadsheetId, currentMonth, storeNoteDataRange)
	fmt.Printf("TOtoal store note:%v", len(allStoreNotes))

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{
		"Order Date (time zone GST)",
		"Order ID",
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
		"Fulfillment Status",
		"Stock Checking",
	})

	res := getListOrder(currentMonth)
	if res.Total > 0 {
		for i := 0; i < res.Total; i++ {
			item := res.Items[i]
			skus := make([]string, 0)
			prices := make([]string, 0)
			quantities := make([]string, 0)

			for j := 0; j < len(item.OrderItems); j++ {
				if item.OrderItems[j].ParentItemId > 0 || (item.OrderItems[j].ParentItemId == 0 && item.OrderItems[j].ProductType == "simple") {
					//productSaleOrderItem := getProduct(item.OrderItems[j].SKU)
					// if productSaleOrderItem.ID > 0 {
					// 	skus = append(skus, item.OrderItems[j].SKU+"("+productSaleOrderItem.BtjCode+")")
					// } else {
					skus = append(skus, item.OrderItems[j].SKU)
					// }

					prices = append(prices, strconv.Itoa(item.OrderItems[j].Price))
					quantities = append(quantities, strconv.Itoa(item.OrderItems[j].QuantityOrdered))
				}
			}
			address := ""
			if len(item.ExtensionAttributes.ShippingAssignments) > 0 {
				shipping := item.ExtensionAttributes.ShippingAssignments[0].Shipping

				if shipping.Method == "smilestoredelivery_smilestoredelivery" {
					address = shipping.ShippingAddress.Company
				} else {
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

			fmt.Printf("Update New Store i:%v", i)
			//Check i exist key in allStoreNotes
			var newHash string
			var hasNewStoreUpdate bool
			if len(allStoreNotes) > i {
				hasNewStoreUpdate, newHash = getHash(listHash[i], allStoreNotes[i])
				if hasNewStoreUpdate {
					fmt.Printf("Update New Store Update")
					//Do update store note
					updateStoreNote(strconv.FormatInt(int64(item.EntityId), 10), allStoreNotes[i].Note, allStoreNotes[i].Status, allStoreNotes[i].ErplyInvoiceIds)
				}
			}

			row := []interface{}{
				item.CreatedAt,
				item.EntityId,
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
				item.ShippingDescription,
				address,
				item.DiscountCode,
				item.GrandTotal,
				item.TotalPaid,
				item.TotalDue,
				item.Payment.TransactionId,
				item.Payment.Method,
				giftInfo,
				item.ExtensionAttributes.FulfillmentStatus,
				item.ExtensionAttributes.StockChecking,
				newHash,
			}
			vr.Values = append(vr.Values, row)
		}
	}

	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, currentMonth+"!A1", &vr).ValueInputOption("RAW").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}

}
