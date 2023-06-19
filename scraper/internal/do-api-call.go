package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func DoAPICall(
	page int,
	size int,
) (
	responseObject APIResponse,
	err error,
) {

	requestUrl, urlErr := url.ParseRequestURI(ColruytAPIEndpoint)
	if urlErr != nil {
		return APIResponse{}, urlErr
	}
	queryParams := requestUrl.Query()
	queryParams.Set("clientCode", "CLP")
	queryParams.Set("page", fmt.Sprint(page))
	queryParams.Set("size", fmt.Sprint(size))
	queryParams.Set("placeId", ColruytPlaceID)
	requestUrl.RawQuery = queryParams.Encode()

	scraperRequestUrl, scraperUrlErr := url.ParseRequestURI(ScraperAPIUrl)
	if scraperUrlErr != nil {
		return APIResponse{}, scraperUrlErr
	}
	scraperQueryParams := requestUrl.Query()
	scraperQueryParams.Set("api_key", ScraperAPIKey)
	scraperQueryParams.Set("keep_headers", "true")
	scraperQueryParams.Set("url", requestUrl.String())
	scraperRequestUrl.RawQuery = scraperQueryParams.Encode()

	request, requestErr := http.NewRequest("GET", scraperRequestUrl.String(), nil)
	if requestErr != nil {
		return APIResponse{}, requestErr
	}

	request.Header.Set("Host", ColruytAPIHost)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("x-cg-apikey", APIKey)
	request.Header.Set("User-Agent", UserAgent)

	fmt.Printf("[%d] Doing API call\n", page)

	response, responseErr := http.DefaultClient.Do(request)
	if responseErr != nil {
		return APIResponse{}, responseErr
	}
	defer response.Body.Close()

	if response.StatusCode == 456 {
		fmt.Printf("[%d] Status code 456, retrying in 10 sec...\n", page)
		time.Sleep(10000 * time.Millisecond)
		return DoAPICall(page, size)
	}

	fmt.Printf("[%d] Status code: %d\n", page, response.StatusCode)

	body, bodyErr := io.ReadAll(response.Body)
	if bodyErr != nil {
		return APIResponse{}, bodyErr
	}

	var apiResponse APIResponse
	unmarshalErr := json.Unmarshal(body, &apiResponse)
	if unmarshalErr != nil {
		return APIResponse{}, unmarshalErr
	}

	return apiResponse, nil
}

func GetAllProducts() (
	products []Product,
	err error,
) {

	initResp, err := DoAPICall(1, 1)
	if err != nil {
		return []Product{}, err
	}

	pages := initResp.ProductsFound/250 + 1
	// TODO remove
	pages = 10

	// Limit to 5 concurrent requests, limit set by ScraperAPI Free plan
	limiter := make(chan int, 5)
	defer close(limiter)
	wg := sync.WaitGroup{}
	wg.Add(pages)

	for i := 1; i <= pages; i++ {
		limiter <- 1
		go func(page int) {
			defer wg.Done()
			defer func() { <-limiter }()
			responseObject, err := DoAPICall(page, 250)
			if err != nil {
				fmt.Println(err)
				return
			}
			products = append(products, responseObject.Products...)
		}(i)
	}

	wg.Wait()
	return products, nil

}