package main

import (
    "log"
	"fmt"
)

// processSearch handles the search request, querying Shopify and sending on_search response
func processSearch(req ONDCSearchRequest) {
	city := req.Context.City
	category := req.Message.Intent.Category.ID
	log.Printf("Processing search for city: %s and category: %s", city, category)

	// Check for incremental catalog refresh request.
	for _, tag := range req.Message.Intent.Tags {
		if tag.Code == "catalog_inc" {
			var mode, startTime, endTime string
			for _, item := range tag.List {
				switch item.Code {
				case "mode":
					mode = item.Value
				case "start_time":
					startTime = item.Value
				case "end_time":
					endTime = item.Value
				}
			}
			log.Printf("Incremental catalog refresh requested - Mode: %s, Start Time: %s, End Time: %s", mode, startTime, endTime)
			// For now, proceed with a full Shopify query.
		}
	}

	// Query Shopify for products using both city and category filters.
	products := queryShopify(city, category)
	if len(products) == 0 {
		log.Printf("No products found for city: %s and category: %s", city, category)
		return
	}

	// Transform the Shopify products into an ONDC catalog format.
	catalog := transformToONDCCatalog(products)

	// Send on_search response.
	if err := sendOnSearch(req.Context.BapURI, catalog, req); err != nil {
		log.Printf("Failed to send on_search for message ID %s: %v", req.Context.MessageID, err)
	}
}

func processSelect(req ONDCSelectRequest) {
	city := req.Context.City
	category := req.Message.Intent.Category.ID
	log.Printf("Processing select for transaction: %s, city: %s, category: %s", req.Context.TransactionID, city, category)

	// Query Shopify for product details with both city and category filters.
	shopifyProducts := queryShopifyForSelect(req)

	// Build a ShopifyResponse from the returned products.
	shopifyResp := ShopifyResponse{
		Data: struct {
			Products struct {
				Edges []struct {
					Node struct {
						ID       string `json:"id"`
						Title    string `json:"title"`
						Variants struct {
							Edges []struct {
								Node struct {
									Price string `json:"price"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"variants"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"products"`
		}{
			Products: struct {
				Edges []struct {
					Node struct {
						ID       string `json:"id"`
						Title    string `json:"title"`
						Variants struct {
							Edges []struct {
								Node struct {
									Price string `json:"price"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"variants"`
					} `json:"node"`
				} `json:"edges"`
			}{
				Edges: make([]struct {
					Node struct {
						ID       string `json:"id"`
						Title    string `json:"title"`
						Variants struct {
							Edges []struct {
								Node struct {
									Price string `json:"price"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"variants"`
					} `json:"node"`
				}, len(shopifyProducts)),
			},
		},
	}

	// Populate the ShopifyResponse.
	for i, product := range shopifyProducts {
		shopifyResp.Data.Products.Edges[i].Node.ID = product.ID
		shopifyResp.Data.Products.Edges[i].Node.Title = product.Title
		shopifyResp.Data.Products.Edges[i].Node.Variants.Edges = []struct {
			Node struct {
				Price string `json:"price"`
			} `json:"node"`
		}{
			{
				Node: struct {
					Price string `json:"price"`
				}{
					Price: product.Price,
				},
			},
		}
	}

	if len(shopifyProducts) == 0 {
		log.Printf("No products found for selection for city: %s and category: %s", city, category)
		return
	}

	// Transform into ONDC select response format.
	selectResponse := transformToONDCCSelectResponse(req, shopifyResp)

	// Send on_select response.
	if err := sendOnSelect(req.Context.BapURI, selectResponse, req); err != nil {
		log.Printf("Failed to send on_select for transaction %s: %v", req.Context.TransactionID, err)
	}
}

// processInit handles the init request by gathering the selected product meta IDs,
// querying Shopify, and constructing the on_init response using the fetched product data.
func processInit(req ONDCInitRequest) {
	log.Printf("Processing init for transaction: %s", req.Context.TransactionID)

	if len(req.Message.Order.Items) == 0 {
		log.Printf("No product selected in init request")
		return
	}

	// Collect all selected product IDs from the init request.
	var selectedIDs []string
	for _, item := range req.Message.Order.Items {
		selectedIDs = append(selectedIDs, item.ID)
	}

	// Build a custom query string to search for any of the selected product IDs.
	// For example: "tag:I1 OR tag:I2"
	queryStr := ""
	for i, id := range selectedIDs {
		if i > 0 {
			queryStr += " OR "
		}
		queryStr += fmt.Sprintf("tag:%s", id)
	}

	// Query Shopify using the custom query string.
	selectedProducts := queryShopifyWithCustomQuery(queryStr)
	if len(selectedProducts) == 0 {
		log.Printf("No matching product found for on_init with provided product ids")
		return
	}

	// Build a ShopifyResponse from the selected products.
	var shopifyResp ShopifyResponse
	shopifyResp.Data.Products.Edges = make([]struct {
		Node struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Variants struct {
				Edges []struct {
					Node struct {
						Price string `json:"price"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"variants"`
		} `json:"node"`
	}, len(selectedProducts))

	for i, p := range selectedProducts {
		shopifyResp.Data.Products.Edges[i].Node.ID = p.ID
		shopifyResp.Data.Products.Edges[i].Node.Title = p.Title
		shopifyResp.Data.Products.Edges[i].Node.Variants.Edges = []struct {
			Node struct {
				Price string `json:"price"`
			} `json:"node"`
		}{
			{
				Node: struct {
					Price string `json:"price"`
				}{
					Price: p.Price,
				},
			},
		}
	}

	// Use the existing transformer to build the on_init response.
	initResponse := transformToONDCInitResponse(req, shopifyResp)
	if err := sendOnInit(req.Context.BapURI, initResponse, req); err != nil {
		log.Printf("Failed to send on_init for transaction %s: %v", req.Context.TransactionID, err)
	}
}

