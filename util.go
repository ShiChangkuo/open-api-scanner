package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	apiEndpoint = "https://apiexplorer.developer.huaweicloud.com"
	MaxPageSize = 100
)

type ListAPIOpts struct {
	Offset       int
	Limit        int
	ProductShort string
	Version      string
}

// ToListQuery formats a ListAPIOpts into a query string.
func (opts ListAPIOpts) ToListQuery() string {
	params := url.Values{}

	params.Add("offset", strconv.Itoa(opts.Offset))
	params.Add("limit", strconv.Itoa(opts.Limit))

	if opts.ProductShort != "" {
		params.Add("product_short", opts.ProductShort)
	}
	if opts.Version != "" {
		params.Add("info_version", opts.Version)
	}

	return params.Encode()
}

type APIGroup struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Products []APIProduct `json:"products"`
}

type APIProduct struct {
	Name         string `json:"name"`
	ProductShort string `json:"productshort"`
	Link         string `json:"link"`
	APICount     int    `json:"api_count"`
	HasDate      bool   `json:"has_data"`
	Recommend    bool   `json:"is_recommend"`
	Global       bool   `json:"is_global"`
}

type APIBasicInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Alias        string `json:"alias_name"`
	Method       string `json:"method"`
	Summary      string `json:"summary"`
	Tags         string `json:"tags"`
	ProductShort string `json:"product_short"`
	Version      string `json:"info_version"`
}

type ProductVersion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func getAllProducts() ([]APIGroup, error) {
	url := fmt.Sprintf("%s/v4/products", apiEndpoint)
	fmt.Printf("[DEBUG] list products url: %s\n", url)

	body, err := httpRequest("GET", url, nil, nil)
	if err != nil {
		fmt.Printf("[ERROR] request %s failed, reason: %s\n", url, err)
		return nil, err
	}
	//fmt.Printf("[DEBUG] api response: %v\n", string(body))

	response := struct {
		Groups []APIGroup `json:"groups"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("[ERROR] Unmarshal failed: %s\n", err)
		return nil, err
	}
	return response.Groups, nil
}

func getProductVersions(product string) ([]string, error) {
	result := []string{}
	url := fmt.Sprintf("%s/v2/versions?productshort=%s", apiEndpoint, product)

	body, err := httpRequest("GET", url, nil, nil)
	if err != nil {
		fmt.Printf("[ERROR] request %s failed, reason: %s\n", url, err)
		return nil, err
	}

	response := struct {
		Count      int              `json:"count"`
		IsMultiple bool             `json:"is_multiple_version"`
		Versions   []ProductVersion `json:"versions"`
	}{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("[ERROR] Unmarshal failed: %s\n", err)
		return nil, err
	}

	for _, v := range response.Versions {
		result = append(result, v.Name)
	}

	// add an empty version if not found
	if len(result) == 0 {
		return []string{""}, nil
	}
	return result, nil
}

func getProductAPIs(product, version string) ([]APIBasicInfo, error) {
	result := []APIBasicInfo{}
	baseURL := fmt.Sprintf("%s/v3/apis", apiEndpoint)

	pageSize := MaxPageSize
	offset := 0
	listOpts := ListAPIOpts{
		Offset:       offset,
		Limit:        MaxPageSize,
		ProductShort: product,
		Version:      version,
	}

	for pageSize == MaxPageSize {
		query := listOpts.ToListQuery()
		url := fmt.Sprintf("%s?%s", baseURL, query)
		//fmt.Printf("[DEBUG] list url: %s\n", url)

		body, err := httpRequest("GET", url, nil, nil)
		if err != nil {
			fmt.Printf("[ERROR] request %s failed, reason: %s\n", url, err)
			return nil, err
		}

		response := struct {
			Count int            `json:"count"`
			APIs  []APIBasicInfo `json:"api_basic_infos"`
		}{}

		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("[ERROR] Unmarshal failed: %s\n", err)
			return nil, err
		}

		result = append(result, response.APIs...)
		pageSize = len(response.APIs)
		listOpts.Offset += pageSize
		if listOpts.Offset == response.Count {
			break
		}
	}
	return result, nil
}

func getAPIDetail(product, apiName, apiVersion, region string) ([]byte, error) {
	url := fmt.Sprintf("%s/v4/apis/detail?product_short=%s&name=%s&region_id=%s",
		apiEndpoint, product, apiName, region)
	if apiVersion != "" {
		url += fmt.Sprintf("&info_version=%s", apiVersion)
	}

	body, err := httpRequest("GET", url, nil, nil)
	if err != nil {
		fmt.Printf("[ERROR] request %s failed, reason: %s\n", url, err)
		return nil, err
	}

	return body, nil
}

func httpRequest(method, url string, jsonBody interface{}, headers map[string]string) ([]byte, error) {
	var body io.Reader
	var contentType string

	if jsonBody != nil {
		rendered, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(rendered)
		contentType = "application/json; charset=UTF-8"
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	authToken := os.Getenv("HW_TOKEN")
	if authToken != "" {
		request.Header.Set("X-Auth-Token", authToken)
	}

	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	if eLanguage {
		request.Header.Set("X-Language", "en-us")
	}

	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	for k, v := range headers {
		request.Header.Add(k, v)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	rst, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	defer client.CloseIdleConnections()

	if response.StatusCode != http.StatusOK {
		return rst, fmt.Errorf("Response Code %d", response.StatusCode)
	}
	return rst, nil
}
