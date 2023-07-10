package main

import (
	"encoding/json"
	"errors"
	api2captcha "github.com/2captcha/2captcha-go"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var defaultGetHeaders = map[string]string{
	"accept":          "application/json, text/plain, */*",
	"accept-language": "en-GB,en-US;q=0.9,en;q=0.8,lt;q=0.7",
	"cache-control":   "no-cache",
	"pragma":          "no-cache",
	"referer":         "https://skinport.com/item/",
	"user-agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36",
}

const SKINPORT_IMAGE_URL = "https://community.cloudflare.steamstatic.com/economy/image/class/730/"
const SKINPORT_PURCHASE_URL = "https://skinport.com/item/"
const MANUAL_LOGIN = true

var GBPinUSD float64

var jar, _ = cookiejar.New(nil)
var checkoutClient = &http.Client{
	Jar: jar,
}

func ConnectWs() (*websocket.Conn, error) {
	InfoLogger.Println("Connecting to SkinPort ws...")

	client, _, err := websocket.DefaultDialer.Dial("wss://skinport.com/socket.io/?EIO=4&transport=websocket", nil)
	if err != nil {
		ReportError(err)
		return nil, err
	}

	client.ReadMessage()
	client.WriteMessage(websocket.TextMessage, []byte("40"))
	client.ReadMessage()
	client.WriteMessage(websocket.TextMessage, []byte("42[\"saleFeedJoin\",{\"appid\":730,\"currency\":\"USD\",\"locale\":\"en\"}]"))
	client.ReadMessage()
	client.ReadMessage()
	client.WriteMessage(websocket.TextMessage, []byte("3"))

	return client, nil
}

func RunSkinport() {
	client, err := ConnectWs()
	if err != nil {
		panic(err)
	}

	attempts := 0
	for {
		_, message, err := client.ReadMessage()
		if err != nil {
			if attempts >= 5 {
				panic(err)
			}

			ErrorLogger.Println("Reconnecting to SkinPort ws...")
			client.Close()
			client, err = ConnectWs()
			attempts++
			continue
		}

		attempts = 0
		strMessage := string(message)
		if strMessage == "2" {
			InfoLogger.Println("Ponging SkinPort WS")
			client.WriteMessage(websocket.TextMessage, []byte("3"))
		} else if strings.Contains(strMessage, "42") {
			var response SkinportPayload
			strMessage = strings.TrimSuffix(strMessage[14:], "]")
			json.Unmarshal([]byte(strMessage), &response)

			for _, item := range response.Sales {
				InfoLogger.Println("Found product " + item.MarketName)
				buffPrice := GetBuffPrice(item.MarketName, item.Version)
				numPrice := float64(item.SalePrice) / 100

				if (PercentageDifference(numPrice, buffPrice) >= config.MinimumProfitPercentage) && numPrice >= config.MinimumPrice && numPrice <= config.MaximumPrice && !strings.Contains(item.MarketName, "StatTrak") {
					SendSkinportProduct(item.MarketName, SKINPORT_IMAGE_URL+item.Classid, item.Link, numPrice, SKINPORT_PURCHASE_URL+item.URL+"/"+strconv.Itoa(item.SaleID), buffPrice, "SkinPort")
					gbp := convertToGbp(item.SalePrice)
					go addToCart(strconv.Itoa(item.SaleID), int(math.Round(gbp*100)))
				}
			}
		} else {
			ReportError(errors.New(strMessage))
		}
	}
}

func login() error {
	checkoutClient.Get("https://skinport.com/")

	if !MANUAL_LOGIN {
		captcha := generateCaptcha("https://skinport.com/signin")
		payload := url.Values{}
		payload.Set("email", config.SkinportUsername)
		payload.Set("password", config.SkinportPassword)
		payload.Set("g-recaptcha-response", captcha)
		payload.Set("_csrf", getCsrfToken())

		checkoutClient.Get("https://skinport.com/api/home")
		loginReq, _ := http.NewRequest(http.MethodPost, "https://skinport.com/api/auth/login", strings.NewReader(payload.Encode()))
		loginReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		for header, value := range defaultGetHeaders {
			loginReq.Header.Add(header, value)
		}

		loginResponse, err := checkoutClient.Do(loginReq)

		if err != nil {
			ReportError(err)
			return err
		}

		body, err := ioutil.ReadAll(loginResponse.Body)
		loginResponse.Body.Close()

		var loginResponseObject LoginResponse
		json.Unmarshal(body, &loginResponseObject)

		if !loginResponseObject.Success {
			ReportError(errors.New(loginResponseObject.Message))
			return errors.New(loginResponseObject.Message)
		}

		if (loginResponseObject.State == 8) && (loginResponseObject.Key != "") {
			authUrl := GetUserInput("Waiting for auth URL (Check Email)...")
			authResponse, err := checkoutClient.Get(authUrl)

			if err != nil {
				ReportError(err)
				return err
			}

			body, err := ioutil.ReadAll(authResponse.Body)
			authResponse.Body.Close()

			var authResponseObject AuthResponse
			json.Unmarshal(body, &authResponseObject)
		}

		return nil
	}

	authCookie := GetUserInput("Waiting for connect.sid cookie...")
	site, _ := url.Parse("https://skinport.com")
	cookie := &http.Cookie{
		Name:   "connect.sid",
		Value:  authCookie,
		Path:   "/",
		Domain: ".skinport.com",
	}

	checkoutClient.Jar.SetCookies(site, []*http.Cookie{cookie})

	return nil
}

func getCsrfToken() string {
	dataReq, _ := http.NewRequest("GET", "https://skinport.com/api/data?v=939402949c4961a7af31&t="+time.Now().UTC().String(), nil)
	for header, value := range defaultGetHeaders {
		dataReq.Header.Add(header, value)
	}

	response, err := checkoutClient.Do(dataReq)

	if err != nil {
		ReportError(err)
		return ""
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if response.StatusCode != 200 {
		ErrorLogger.Println("Erroneous response received")
		return ""
	}

	var apiData APIDataResponse
	json.Unmarshal(body, &apiData)

	GBPinUSD = apiData.Rates.USD
	return apiData.Csrf
}

func addToCart(saleId string, price int) error {
	payload := url.Values{}
	payload.Set("sales[0][id]", saleId)
	payload.Set("sales[0][price]", strconv.Itoa(price))
	payload.Set("_csrf", getCsrfToken())

	atcReq, _ := http.NewRequest(http.MethodPost, "https://skinport.com/api/cart/add", strings.NewReader(payload.Encode()))
	atcReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	for header, value := range defaultGetHeaders {
		atcReq.Header.Add(header, value)
	}

	atcResponse, err := checkoutClient.Do(atcReq)

	if err != nil {
		ReportError(err)
		return err
	}

	body, err := ioutil.ReadAll(atcResponse.Body)
	atcResponse.Body.Close()

	var atcResponseObject ATCResponse
	json.Unmarshal(body, &atcResponseObject)

	if !atcResponseObject.Success {
		if atcResponseObject.Message == "MUST_LOGIN" {
			err = errors.New("Login expired at ATC")
			login()
		} else if atcResponseObject.Message == "ITEM_NOT_LISTED" {
			err = errors.New(saleId + " is now OOS")
		} else {
			err = errors.New(atcResponseObject.Message)
		}

		ReportError(err)
		return err
	}

	ReportATC()
	return nil
}

// Broken due to updated captcha
/*func submitOrder(saleId string) error {
	captcha := generateCaptcha("https://skinport.com/cart")
	payload := url.Values{}
	payload.Set("sales[0]", saleId)
	payload.Set("g-recaptcha-response", captcha)
	payload.Set("_csrf", getCsrfToken())

	submitReq, _ := http.NewRequest(http.MethodPost, "https://skinport.com/api/checkout/create-order", strings.NewReader(payload.Encode()))
	submitReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	for header, value := range defaultGetHeaders {
		submitReq.Header.Add(header, value)
	}

	submitResponse, err := checkoutClient.Do(submitReq)

	if err != nil {
		ReportError(err)
		return err
	}

	body, err := ioutil.ReadAll(submitResponse.Body)
	submitResponse.Body.Close()

	return nil
}*/

func convertToGbp(usd int) float64 {
	return (float64(usd) / 100) / GBPinUSD
}

func generateCaptcha(url string) string {
	captchaClient := api2captcha.NewClient(config.TwoCaptchaKey)
	cap := api2captcha.ReCaptcha{
		SiteKey: "6Ldo-yEgAAAAAIBUo13yCs0Pjek0XuIKUIS6lHFJ",
		Url:     url,
		Version: "v3",
		Score:   0.3,
	}
	req := cap.ToRequest()
	code, err := captchaClient.Solve(req)
	if err != nil {
		ReportError(err)
	}

	return code
}

type ATCResponse struct {
	RequestID    string      `json:"requestId"`
	Success      bool        `json:"success"`
	Message      string      `json:"message"`
	Notification interface{} `json:"notification"`
}

type AuthResponse struct {
	RequestID string `json:"requestId"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	User      struct {
		ID                  int    `json:"id"`
		Username            string `json:"username"`
		Avatar              string `json:"avatar"`
		SteamAccounts       int    `json:"steamAccounts"`
		Balance             int    `json:"balance"`
		WithdrawAble        int    `json:"withdrawAble"`
		Trusted             bool   `json:"trusted"`
		TwoFactor           bool   `json:"twoFactor"`
		Password            bool   `json:"password"`
		VoucherAccess       bool   `json:"voucherAccess"`
		APIAccess           bool   `json:"apiAccess"`
		BillingAddress      bool   `json:"billingAddress"`
		Affiliate           bool   `json:"affiliate"`
		UnreadNotifications int    `json:"unreadNotifications"`
	} `json:"user"`
}

type LoginResponse struct {
	RequestID string `json:"requestId"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	State     int    `json:"state"`
	Email     string `json:"email"`
	Key       string `json:"key"`
}

type APIDataResponse struct {
	RequestID string      `json:"requestId"`
	Success   bool        `json:"success"`
	Message   interface{} `json:"message"`
	Csrf      string      `json:"csrf"`
	Country   string      `json:"country"`
	Currency  string      `json:"currency"`
	Rate      float64     `json:"rate"`
	Rates     struct {
		EUR float64 `json:"EUR"`
		DKK float64 `json:"DKK"`
		HRK float64 `json:"HRK"`
		CZK float64 `json:"CZK"`
		NOK float64 `json:"NOK"`
		PLN float64 `json:"PLN"`
		SEK float64 `json:"SEK"`
		USD float64 `json:"USD"`
		CAD float64 `json:"CAD"`
		CHF float64 `json:"CHF"`
		AUD float64 `json:"AUD"`
		BRL float64 `json:"BRL"`
		GBP int     `json:"GBP"`
		CNY float64 `json:"CNY"`
		RUB float64 `json:"RUB"`
		TRY float64 `json:"TRY"`
		SGD float64 `json:"SGD"`
		NZD float64 `json:"NZD"`
		HKD float64 `json:"HKD"`
	} `json:"rates"`
	Locale string `json:"locale"`
	Tags   []struct {
		Tag   string `json:"tag"`
		Appid int    `json:"appid"`
	} `json:"tags"`
	Limits struct {
		MinOrderValue     int `json:"minOrderValue"`
		KycTier1PayoutMax int `json:"kycTier1PayoutMax"`
		MinSaleValue      int `json:"minSaleValue"`
		SaleFeeReduced    int `json:"saleFeeReduced"`
		MaxOrderValue     int `json:"maxOrderValue"`
		MinPayoutValue    int `json:"minPayoutValue"`
		KycTier2PayoutMax int `json:"kycTier2PayoutMax"`
	} `json:"limits"`
	PaymentMethods []string      `json:"paymentMethods"`
	Following      []interface{} `json:"following"`
}

type SkinportPayload struct {
	EventType string            `json:"eventType"`
	Sales     []SkinportProduct `json:"sales"`
}

type SkinportProduct struct {
	ID                   int           `json:"id"`
	SaleID               int           `json:"saleId"`
	ProductID            int           `json:"productId"`
	AssetID              int           `json:"assetId"`
	ItemID               int           `json:"itemId"`
	Appid                int           `json:"appid"`
	Steamid              string        `json:"steamid"`
	URL                  string        `json:"url"`
	Family               string        `json:"family"`
	FamilyLocalized      string        `json:"family_localized"`
	Name                 string        `json:"name"`
	Title                string        `json:"title"`
	Text                 string        `json:"text"`
	MarketName           string        `json:"marketName"`
	MarketHashName       string        `json:"marketHashName"`
	Color                string        `json:"color"`
	BgColor              interface{}   `json:"bgColor"`
	Image                string        `json:"image"`
	Classid              string        `json:"classid"`
	Assetid              string        `json:"assetid"`
	Lock                 time.Time     `json:"lock"`
	Version              string        `json:"version"`
	VersionType          string        `json:"versionType"`
	StackAble            bool          `json:"stackAble"`
	SuggestedPrice       int           `json:"suggestedPrice"`
	SalePrice            int           `json:"salePrice"`
	Currency             string        `json:"currency"`
	SaleStatus           string        `json:"saleStatus"`
	SaleType             string        `json:"saleType"`
	Category             string        `json:"category"`
	CategoryLocalized    string        `json:"category_localized"`
	SubCategory          string        `json:"subCategory"`
	SubCategoryLocalized string        `json:"subCategory_localized"`
	Pattern              int           `json:"pattern"`
	Finish               int           `json:"finish"`
	CustomName           interface{}   `json:"customName"`
	Wear                 float64       `json:"wear"`
	Link                 string        `json:"link"`
	Type                 string        `json:"type"`
	Exterior             string        `json:"exterior"`
	Quality              string        `json:"quality"`
	Rarity               string        `json:"rarity"`
	RarityLocalized      string        `json:"rarity_localized"`
	RarityColor          string        `json:"rarityColor"`
	Collection           interface{}   `json:"collection"`
	CollectionLocalized  interface{}   `json:"collection_localized"`
	Stickers             []interface{} `json:"stickers"`
	CanHaveScreenshots   bool          `json:"canHaveScreenshots"`
	Screenshots          []interface{} `json:"screenshots"`
	Souvenir             bool          `json:"souvenir"`
	Stattrak             bool          `json:"stattrak"`
	Tags                 []struct {
		Name          string `json:"name"`
		NameLocalized string `json:"name_localized"`
	} `json:"tags"`
	OwnItem bool `json:"ownItem"`
}
