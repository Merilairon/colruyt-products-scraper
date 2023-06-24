package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	shared "shared/pkg"
	"sync"
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
	queryParams.Set("sort", "new desc")
	requestUrl.RawQuery = queryParams.Encode()

	request, requestErr := http.NewRequest("GET", requestUrl.String(), nil)
	if requestErr != nil {
		return APIResponse{}, requestErr
	}

	request.Header.Set("Host", ColruytAPIHost)

	if page == 1 && size == 1 {
		fmt.Println("- Doing initial API call")
	}

	response, responseErr := shared.UseProxy(request)
	if responseErr != nil {
		return DoAPICall(page, size)
	}
	defer response.Body.Close()

	// fmt.Printf("[%d] Status code: %d\n", page, response.StatusCode)
	if response.StatusCode != 200 {
		return DoAPICall(page, size)
	}

	body, bodyErr := io.ReadAll(response.Body)
	if bodyErr != nil {
		return DoAPICall(page, size)
	}

	var apiResponse APIResponse
	unmarshalErr := json.Unmarshal(body, &apiResponse)
	if unmarshalErr != nil {
		return DoAPICall(page, size)
	}

	fmt.Printf("[%d] Call successfull\n", page)

	return apiResponse, nil
}

func GetAllProducts() (
	products []shared.Product,
	err error,
) {

	pageSize := 250
	limit := 50
	percentageRequired := 100.0 / 100.0

	initResp, err := DoAPICall(1, 1)
	if err != nil {
		return []shared.Product{}, err
	}
	fmt.Printf("Should retrieve %d products\n", initResp.ProductsFound)

	pages := initResp.ProductsFound/pageSize + 1

	// Limit to 5 concurrent requests, limit set by ScraperAPI Free plan
	limiter := make(chan int, limit)
	defer close(limiter)
	wg := sync.WaitGroup{}

	productsMutex := sync.Mutex{}
	alreadyAdded := map[string]bool{}

	productsRequired := int(float64(initResp.ProductsFound) * percentageRequired)

	// For some absolute bonkers reason the API likes to go wild and return
	// different objects for the same page.
	// It seems as if it sometimes just doesn't care about parameters passed along.
	//
	// Go to the `assortiment` page and order by `new`, refresh a couple of
	// times and you'll see different results, like it somehow doesn't list some
	// products. I am proper mad about this tbh.
waitTillWeGotEnoughProducts:
	for {
		for i := 1; i <= pages; i++ {
			limiter <- 1
			wg.Add(1)
			fmt.Printf(
				"--- Acc: %d / %d (%d%s)\n",
				len(products),
				productsRequired,
				int((float32(len(products))/float32(productsRequired))*100),
				"%",
			)
			if len(products) >= int(float64(initResp.ProductsFound)*percentageRequired) {
				<-limiter
				wg.Done()
				fmt.Println("==========      Got enough products, breaking (pending processes will still finish)")
				break waitTillWeGotEnoughProducts
			}
			go func(page int) {
				defer wg.Done()
				defer func() { <-limiter }()
				responseObject, err := DoAPICall(page, pageSize)
				if err != nil {
					fmt.Println(err)
				}

				for _, product := range responseObject.Products {
					productsMutex.Lock()
					if !alreadyAdded[product.ProductID] {
						alreadyAdded[product.ProductID] = true
						products = append(products, product)
					}
					productsMutex.Unlock()
				}

			}(i)
		}
	}

	wg.Wait()

	return products, nil

}