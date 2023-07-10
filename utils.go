package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DMARKET_API_URL = "https://api.dmarket.com"
const PRICES_URL = "https://prices.csgotrader.app/latest/prices_v6.json"

var marketPrices = make(map[string]MarketPrices)

func GetPrivateKey(s string) *[64]byte {
	b, _ := hex.DecodeString(s)
	var privateKey [64]byte
	copy(privateKey[:], b[:64])

	return &privateKey
}

func Sign(pk, msg string) string {
	b := GetPrivateKey(pk)
	return hex.EncodeToString(ed25519.Sign((*b)[:], []byte(msg)))
}

func SendSignedDmarketRequest(method string, path string, body string) (*http.Response, error) {
	timestamp := strconv.Itoa(int(time.Now().UTC().Unix()))
	unsigned := method + path + body + timestamp
	signature := Sign(config.DmarketPrivateKey, unsigned)

	req, _ := http.NewRequest(method, DMARKET_API_URL+path, ioutil.NopCloser(strings.NewReader(body)))
	req.Header.Set("X-Sign-Date", timestamp)
	req.Header.Set("X-Request-Sign", "dmar ed25519 "+signature)
	req.Header.Set("X-Api-Key", config.DmarketPublicKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return http.DefaultClient.Do(req)
}

func HandleError(response *http.Response) {
	ErrorLogger.Println("Erroneous response received: " + strconv.Itoa(response.StatusCode))

	body, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var errorObj DmarketError
	json.Unmarshal(body, &errorObj)

	ReportError(errors.New(errorObj.Message))
}

func fetchPrices() {
	response, _ := http.DefaultClient.Get(PRICES_URL)

	if response.StatusCode != 200 {
		panic("Failed to fetch prices")
	}

	body, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	json.Unmarshal(body, &marketPrices)
}

func PercentageDifference(cost float64, revenue float64) float64 {
	return ((revenue - cost) / revenue) * 100
}

type DmarketError struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details []struct {
		TypeURL string `json:"type_url"`
		Value   string `json:"value"`
	} `json:"details"`
}

type MarketPrices struct {
	Steam struct {
		Last24H float64 `json:"last_24h"`
		Last7D  float64 `json:"last_7d"`
		Last30D float64 `json:"last_30d"`
		Last90D float64 `json:"last_90d"`
	} `json:"steam"`
	Bitskins struct {
		Price            string      `json:"price"`
		InstantSalePrice interface{} `json:"instant_sale_price"`
	} `json:"bitskins"`
	Lootfarm float64 `json:"lootfarm"`
	Csgotm   string  `json:"csgotm"`
	Csmoney  struct {
		Price   float64 `json:"price"`
		Doppler struct {
			Phase1     float64 `json:"Phase 1"`
			Ruby       int     `json:"Ruby"`
			Phase3     float64 `json:"Phase 3"`
			Phase4     float64 `json:"Phase 4"`
			Phase2     int     `json:"Phase 2"`
			Sapphire   float64 `json:"Sapphire"`
			BlackPearl int     `json:"Black Pearl"`
			FactoryNew int     `json:"(Factory New)"`
		} `json:"doppler"`
	} `json:"csmoney"`
	Skinport struct {
		SuggestedPrice float64 `json:"suggested_price"`
		StartingAt     float64 `json:"starting_at"`
	} `json:"skinport"`
	Csgotrader struct {
		Price   float64 `json:"price"`
		Doppler struct {
			Phase1     float64     `json:"Phase 1"`
			Ruby       float64     `json:"Ruby"`
			Phase3     float64     `json:"Phase 3"`
			Phase4     float64     `json:"Phase 4"`
			Phase2     float64     `json:"Phase 2"`
			Sapphire   float64     `json:"Sapphire"`
			BlackPearl float64     `json:"Black Pearl"`
			FactoryNew interface{} `json:"(Factory New)"`
		} `json:"doppler"`
	} `json:"csgotrader"`
	Csgoempire float64 `json:"csgoempire"`
	Swapgg     float64 `json:"swapgg"`
	Csgoexo    float64 `json:"csgoexo"`
	Cstrade    struct {
		Doppler struct {
			Phase2 float64 `json:"Phase 2"`
			Phase4 float64 `json:"Phase 4"`
			Phase1 float64 `json:"Phase 1"`
			Phase3 float64 `json:"Phase 3"`
		} `json:"doppler"`
		Price float64 `json:"price"`
	} `json:"cstrade"`
	Skinwallet float64 `json:"skinwallet"`
	Buff163    struct {
		StartingAt struct {
			Price   float64            `json:"price"`
			Doppler map[string]float64 `json:"doppler"`
		} `json:"starting_at"`
		HighestOrder struct {
			Price   float64            `json:"price"`
			Doppler map[string]float64 `json:"doppler"`
		} `json:"highest_order"`
	} `json:"buff163"`
}

type ItemPrices struct {
	Steam struct {
		Last24H float64 `json:"last_24h"`
		Last7D  float64 `json:"last_7d"`
		Last30D float64 `json:"last_30d"`
		Last90D float64 `json:"last_90d"`
	} `json:"steam"`
	Bitskins struct {
		Price            string      `json:"price"`
		InstantSalePrice interface{} `json:"instant_sale_price"`
	} `json:"bitskins"`
	Lootfarm float64 `json:"lootfarm"`
	Csgotm   string  `json:"csgotm"`
	Csmoney  struct {
		Price   float64 `json:"price"`
		Doppler struct {
			Phase1     float64 `json:"Phase 1"`
			Ruby       int     `json:"Ruby"`
			Phase3     float64 `json:"Phase 3"`
			Phase4     float64 `json:"Phase 4"`
			Phase2     int     `json:"Phase 2"`
			Sapphire   float64 `json:"Sapphire"`
			BlackPearl int     `json:"Black Pearl"`
			FactoryNew int     `json:"(Factory New)"`
		} `json:"doppler"`
	} `json:"csmoney"`
	Skinport struct {
		SuggestedPrice float64 `json:"suggested_price"`
		StartingAt     float64 `json:"starting_at"`
	} `json:"skinport"`
	Csgotrader struct {
		Price   float64 `json:"price"`
		Doppler struct {
			Phase1     float64     `json:"Phase 1"`
			Ruby       float64     `json:"Ruby"`
			Phase3     float64     `json:"Phase 3"`
			Phase4     float64     `json:"Phase 4"`
			Phase2     float64     `json:"Phase 2"`
			Sapphire   float64     `json:"Sapphire"`
			BlackPearl float64     `json:"Black Pearl"`
			FactoryNew interface{} `json:"(Factory New)"`
		} `json:"doppler"`
	} `json:"csgotrader"`
	Csgoempire float64 `json:"csgoempire"`
	Swapgg     float64 `json:"swapgg"`
	Csgoexo    float64 `json:"csgoexo"`
	Cstrade    struct {
		Doppler struct {
			Phase2 float64 `json:"Phase 2"`
			Phase4 float64 `json:"Phase 4"`
			Phase1 float64 `json:"Phase 1"`
			Phase3 float64 `json:"Phase 3"`
		} `json:"doppler"`
		Price float64 `json:"price"`
	} `json:"cstrade"`
	Skinwallet float64 `json:"skinwallet"`
	Buff163    struct {
		StartingAt struct {
			Price   float64            `json:"price"`
			Doppler map[string]float64 `json:"doppler"`
		} `json:"starting_at"`
		HighestOrder struct {
			Price   float64            `json:"price"`
			Doppler map[string]float64 `json:"doppler"`
		} `json:"highest_order"`
	} `json:"buff163"`
}
