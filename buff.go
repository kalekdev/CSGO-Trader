package main

import "strings"

const BUFF_SUGGESTION_URL = "https://buff.163.com/api/market/search/suggest?game=csgo&text="
const BUFF_ITEM_URL = "https://buff.163.com/goods/"

type SuggestionResponse struct {
	Code string `json:"code"`
	Data struct {
		Suggestions []struct {
			GoodsIds string `json:"goods_ids"`
			Option   string `json:"option"`
		} `json:"suggestions"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func GetBuffPrice(itemName string, phase string) float64 {
	prices := marketPrices[itemName]

	if phase != "" && phase != "default" {
		return prices.Buff163.HighestOrder.Doppler[phase]
	}

	return prices.Buff163.HighestOrder.Price
}

func GetBuffUrl(name string) string {
	// The exact item URL can only be fetched after logged in now.
	/*fmt.Println(time.Now().Format(timeLayout), "Fetching buff item suggestions...")

	client := &http.Client{}
	req, _ := http.NewRequest("GET", BUFF_SUGGESTION_URL+url.QueryEscape(name), nil)
	for header, value := range defaultGetHeaders {
		req.Header.Add(header, value)
	}

	response, err := client.Do(req)

	if err != nil || (response.StatusCode != 200) {
		fmt.Println(time.Now().Format(timeLayout), "Erroneous response received")
		return ""
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var suggestionResponse SuggestionResponse

	json.Unmarshal(body, &suggestionResponse)

	return BUFF_ITEM_URL + suggestionResponse.Data.Suggestions[0].GoodsIds*/
	return "https://buff.163.com/market/csgo#tab=selling&page_num=1&search=" + strings.ReplaceAll(name, " ", "%20")
}
