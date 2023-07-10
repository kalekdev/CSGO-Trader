package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var apiUrl = "https://api.dmarket.com/exchange/v1/market/items?side=market&orderBy=updated&orderDir=desc&title=&priceFrom=" + fmt.Sprintf("%f", config.MinimumPrice) + "&priceTo=" + fmt.Sprintf("%f", config.MinimumPrice) + "&treeFilters=&gameId=a8db&cursor=&limit=100&currency=USD&platform=browser&isLoggedIn=false&types="

var balance float64

func RunDmarket(delayMs int, marketType string) {
	ticker := time.Tick(time.Duration(delayMs) * time.Millisecond)

	firstTime := true
	var newProducts []DmarketProduct
	var products []DmarketProduct

	for range ticker {
		InfoLogger.Println("Fetching new dmarket items (" + marketType + ")")

		response, err := http.DefaultClient.Get(apiUrl + marketType)

		if err != nil {
			fmt.Println(err)
			continue
		}

		if response.StatusCode != 200 {
			ErrorLogger.Println("Erroneous response received: " + strconv.Itoa(response.StatusCode))
			continue
		}

		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		var productsObj DmarketProductsResponse
		var toSend []DmarketProduct
		json.Unmarshal(body, &productsObj)

		newProducts = productsObj.Objects

		for _, product := range productsObj.Objects {
			send := true

			for _, oldProduct := range products {
				if oldProduct.ProductID == product.ProductID {
					send = false
				}
			}

			if send {
				toSend = append(toSend, product)
			}
		}

		for _, product := range toSend {
			InfoLogger.Println("Found product", product.Title, marketType)
			buffPrice := GetBuffPrice(product.Title, product.Extra.PhaseTitle)
			numPrice, _ := strconv.ParseFloat(product.Price.USD, 32)
			numPrice = numPrice / 100
			if !firstTime && numPrice <= balance && (PercentageDifference(numPrice, buffPrice) >= config.MinimumProfitPercentage) && !strings.Contains(product.Title, "StatTrak") {
				go PurchaseProduct(&product, marketType)
			}
		}

		products = newProducts
		firstTime = false
	}
}

func PurchaseProduct(product *DmarketProduct, marketType string) {
	payload := fmt.Sprintf("{\"offers\": [{\"offerId\": \"%s\",\"price\": {\"amount\": \"%s\",\"currency\": \"USD\"},\"type\": \"%s\"}]}", product.Extra.OfferID, product.Price.USD, marketType)
	response, err := SendSignedDmarketRequest(http.MethodPatch, "/exchange/v1/offers-buy", payload)

	if err != nil {
		fmt.Println(err)
		ReportError(err)
		return
	}

	if response.StatusCode != 200 {
		HandleError(response)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var orderObj DmarketOrderResponse
	json.Unmarshal(body, &orderObj)

	if orderObj.Status == "TxSuccess" {
		// Successful dmarket order
		SendDmarketPurchase(product, marketType, orderObj.OrderID)
	} else if orderObj.Status == "" && strings.Contains(string(body), "{\"started\":true}") {
		// Successful p2p
		SendDmarketPurchase(product, marketType, orderObj.OrderID)
	} else if orderObj.DmOffersFailReason.Code == "OfferNotFound" {
		// OOS
		ReportError(errors.New("The following product was OOS at the time of purchase: " + product.Title))
	} else {
		ReportError(errors.New("Unknown order response: " + string(body)))
	}

	UpdateAvailableBalance()
}

func UpdateAvailableBalance() {
	response, err := SendSignedDmarketRequest(http.MethodGet, "/account/v1/balance", "")

	if err != nil {
		fmt.Println(err)
		ReportError(err)
		return
	}

	if response.StatusCode != 200 {
		HandleError(response)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var balanceObj DmarketBalance
	json.Unmarshal(body, &balanceObj)

	balanceInt, _ := strconv.Atoi(balanceObj.Usd)
	balance = float64(balanceInt) / 100
}

type DmarketOrderResponse struct {
	TxID            string      `json:"txId"`
	Status          string      `json:"status"`
	OrderID         string      `json:"orderId"`
	P2POffersStatus interface{} `json:"p2pOffersStatus"`
	DmOffersStatus  struct {
	} `json:"dmOffersStatus"`
	DmOffersFailReason struct {
		Code string `json:"code"`
	} `json:"dmOffersFailReason"`
}

type DmarketBalance struct {
	Dmc                    string `json:"dmc"`
	DmcAvailableToWithdraw string `json:"dmcAvailableToWithdraw"`
	Usd                    string `json:"usd"`
	UsdAvailableToWithdraw string `json:"usdAvailableToWithdraw"`
}

type DmarketProduct struct {
	ItemID             string `json:"itemId"`
	Type               string `json:"type"`
	Amount             int    `json:"amount"`
	ClassID            string `json:"classId"`
	GameID             string `json:"gameId"`
	GameType           string `json:"gameType"`
	InMarket           bool   `json:"inMarket"`
	LockStatus         bool   `json:"lockStatus"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	Image              string `json:"image"`
	Slug               string `json:"slug"`
	Owner              string `json:"owner"`
	OwnersBlockchainID string `json:"ownersBlockchainId"`
	OwnerDetails       struct {
		ID     string `json:"id"`
		Avatar string `json:"avatar"`
		Wallet string `json:"wallet"`
	} `json:"ownerDetails"`
	Status   string `json:"status"`
	Discount int    `json:"discount"`
	Price    struct {
		DMC string `json:"DMC"`
		USD string `json:"USD"`
	} `json:"price"`
	InstantPrice struct {
		DMC string `json:"DMC"`
		USD string `json:"USD"`
	} `json:"instantPrice"`
	ExchangePrice struct {
		DMC string `json:"DMC"`
		USD string `json:"USD"`
	} `json:"exchangePrice"`
	InstantTargetID string `json:"instantTargetId"`
	SuggestedPrice  struct {
		DMC string `json:"DMC"`
		USD string `json:"USD"`
	} `json:"suggestedPrice"`
	RecommendedPrice struct {
		OfferPrice struct {
			DMC string `json:"DMC"`
			USD string `json:"USD"`
		} `json:"offerPrice"`
		D3 struct {
			DMC string `json:"DMC"`
			USD string `json:"USD"`
		} `json:"d3"`
		D7 struct {
			DMC string `json:"DMC"`
			USD string `json:"USD"`
		} `json:"d7"`
		D7Plus struct {
			DMC string `json:"DMC"`
			USD string `json:"USD"`
		} `json:"d7Plus"`
	} `json:"recommendedPrice"`
	Extra struct {
		NameColor         string   `json:"nameColor"`
		BackgroundColor   string   `json:"backgroundColor"`
		Tradable          bool     `json:"tradable"`
		OfferID           string   `json:"offerId"`
		IsNew             bool     `json:"isNew"`
		GameID            string   `json:"gameId"`
		Name              string   `json:"name"`
		CategoryPath      string   `json:"categoryPath"`
		LinkID            string   `json:"linkId"`
		Exterior          string   `json:"exterior"`
		Quality           string   `json:"quality"`
		Category          string   `json:"category"`
		TradeLockDuration int      `json:"tradeLockDuration"`
		ItemType          string   `json:"itemType"`
		InspectInGame     string   `json:"inspectInGame"`
		Collection        []string `json:"collection"`
		SaleRestricted    bool     `json:"saleRestricted"`
		InGameAssetID     string   `json:"inGameAssetID"`
		EmissionSerial    string   `json:"emissionSerial"`
		PhaseTitle        string   `json:"phaseTitle"`
	} `json:"extra"`
	CreatedAt     int `json:"createdAt"`
	DeliveryStats struct {
		Rate string `json:"rate"`
		Time string `json:"time"`
	} `json:"deliveryStats"`
	Fees struct {
		F2F struct {
			Sell struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"sell"`
			InstantSell struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"instantSell"`
			Exchange struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"exchange"`
		} `json:"f2f"`
		Dmarket struct {
			Sell struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"sell"`
			InstantSell struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"instantSell"`
			Exchange struct {
				Default struct {
					Percentage string `json:"percentage"`
					MinFee     struct {
						DMC string `json:"DMC"`
						USD string `json:"USD"`
					} `json:"minFee"`
				} `json:"default"`
			} `json:"exchange"`
		} `json:"dmarket"`
	} `json:"fees"`
	DiscountPrice struct {
		DMC string `json:"DMC"`
		USD string `json:"USD"`
	} `json:"discountPrice"`
	ProductID string `json:"productId"`
}

type DmarketProductsResponse struct {
	Objects []DmarketProduct `json:"objects"`
	Total   struct {
		Offers          int `json:"offers"`
		Targets         int `json:"targets"`
		Items           int `json:"items"`
		CompletedOffers int `json:"completedOffers"`
		ClosedTargets   int `json:"closedTargets"`
	} `json:"total"`
	Cursor string `json:"cursor"`
}
