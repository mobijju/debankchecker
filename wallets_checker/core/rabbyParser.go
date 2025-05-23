package core

import (
	"debank_checker_v3/customTypes"
	"debank_checker_v3/utils"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"net/url"
	"sort"
	"time"
)

func SortByChainBalance(data []customTypes.RabbyReturnData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].ChainBalance > data[j].ChainBalance
	})
}

func getTotalBalance(accountAddress string) (float64, []customTypes.RabbyReturnData) {
	baseURL := "https://api.rabby.io/v1/user/total_balance"
	params := url.Values{}
	params.Set("id", accountAddress)

	type chainList struct {
		Name       string  `json:"name"`
		Token      string  `json:"native_token_id"`
		UsdBalance float64 `json:"usd_value"`
	}

	type responseStruct struct {
		ErrorCode     int         `json:"error_code"`
		TotalUsdValue float64     `json:"total_usd_value"`
		ChainList     []chainList `json:"chain_list"`
		Message       string      `json:"message,omitempty"`
	}

	for {
		randomProxy := utils.GetProxy()
		client := GetClient(randomProxy)
		var result []customTypes.RabbyReturnData

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		req.SetRequestURI(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
		req.Header.SetMethod(fasthttp.MethodGet)
		req.Header.Set("accept", "application/json")
		req.Header.Set("accept-language", "en-US,en;q=0.9")
		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		if err := client.Do(req, resp); err != nil {
			log.Printf("%s | Request Error: %s", accountAddress, err)
			time.Sleep(3 * time.Second)
			continue
		}

		statusCode := resp.StatusCode()

		if statusCode == 429 || statusCode == 403 {
			log.Printf("%s | Rate Limited (HTTP %d) - Sleeping 15 seconds...", accountAddress, statusCode)
			time.Sleep(3 * time.Second)
			continue
		}

		responseData := &responseStruct{}

		if err := json.Unmarshal(resp.Body(), &responseData); err != nil {
			log.Printf("%s | Failed To Parse JSON Response: %s", accountAddress, err)
			time.Sleep(3 * time.Second)
			continue
		}

		if responseData.Message == "Too Many Requests" {
			log.Printf("%s | Rate Limited by Message - Sleeping 15 seconds...", accountAddress)
			time.Sleep(3 * time.Second)
			continue
		}

		totalUsdBalance := responseData.TotalUsdValue

		for _, currentChain := range responseData.ChainList {
			if currentChain.UsdBalance <= 0 {
				continue
			}
			result = append(result, customTypes.RabbyReturnData{
				ChainName:    currentChain.Name,
				ChainBalance: currentChain.UsdBalance,
			})
		}

		return totalUsdBalance, result
	}
}

func ParseRabbyAccount(accountData string) {
	accountAddress, _, _, err := utils.GetAccountData(accountData)
	if err != nil {
		log.Printf("%s", err)
		return
	}

	totalUsdBalance, chainBalances := getTotalBalance(accountAddress)
	SortByChainBalance(chainBalances)

	var formattedResult string
	formattedResult += fmt.Sprintf("Account Data: %s\nAddress: %s\nTotal Balance: %.2f $\n\n", accountData, accountAddress, totalUsdBalance)

	for _, currentChain := range chainBalances {
		formattedResult += fmt.Sprintf("%s | %.2f $\n", currentChain.ChainName, currentChain.ChainBalance)
	}

	formattedResult += "\n\n"

	log.Printf("%s | Total USD Balance: %.2f $", accountAddress, totalUsdBalance)

	var filePath string

	switch {
	case totalUsdBalance >= 0 && totalUsdBalance < 1:
		filePath = "0_1_rabby.txt"
	case totalUsdBalance >= 1 && totalUsdBalance < 10:
		filePath = "1_10_rabby.txt"
	case totalUsdBalance >= 10 && totalUsdBalance < 100:
		filePath = "10_100_rabby.txt"
	case totalUsdBalance >= 100 && totalUsdBalance < 500:
		filePath = "100_500_rabby.txt"
	case totalUsdBalance >= 500 && totalUsdBalance < 1000:
		filePath = "500_1000_rabby.txt"
	default:
		filePath = "1000_plus_rabby.txt"
	}

	utils.AppendFile("./results/"+filePath, formattedResult)
}
