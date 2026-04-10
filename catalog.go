package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"net/http"
	"os"
	"time"
)

// transformToONDCCatalog converts Shopify products into the ONDC catalog format.
func transformToONDCCatalog(products []shopifyProduct) ONDCCatalog {
	catalog := ONDCCatalog{
		Fulfillments: []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}{
			{ID: "1", Type: "Delivery"},
			{ID: "2", Type: "Self-Pickup"},
			{ID: "3", Type: "Delivery and Self-Pickup"},
		},
		Descriptor: struct {
			Name      string   `json:"name"`
			Symbol    string   `json:"symbol"`
			ShortDesc string   `json:"short_desc"`
			LongDesc  string   `json:"long_desc"`
			Images    []string `json:"images"`
			Tags      []struct {
				Code string `json:"code"`
				List []struct {
					Code  string `json:"code"`
					Value string `json:"value"`
				} `json:"list"`
			} `json:"tags"`
		}{
			Name:      "Seller NP",
			Symbol:    "https://sellerNP.com/images/np.png",
			ShortDesc: "Seller Marketplace",
			LongDesc:  "Seller Marketplace",
			Images:    []string{"https://sellerNP.com/images/np.png"},
			Tags: []struct {
				Code string `json:"code"`
				List []struct {
					Code  string `json:"code"`
					Value string `json:"value"`
				} `json:"list"`
			}{
				{
					Code: "bpp_terms",
					List: []struct {
						Code  string `json:"code"`
						Value string `json:"value"`
					}{
						{Code: "np_type", Value: "MSN"},
						{Code: "accept_bap_terms", Value: "Y"},
						{Code: "collect_payment", Value: "Y"},
					},
				},
			},
		},
		Providers: []struct {
			ID    string `json:"id"`
			Items []struct {
				ID         string `json:"id"`
				Descriptor struct {
					Name string `json:"name"`
				} `json:"descriptor"`
				Price struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				} `json:"price"`
			} `json:"items"`
		}{
			{
				ID:    "P1",
				Items: []struct {
					ID         string `json:"id"`
					Descriptor struct {
						Name string `json:"name"`
					} `json:"descriptor"`
					Price struct {
						Currency string `json:"currency"`
						Value    string `json:"value"`
					} `json:"price"`
				}{},
			},
		},
	}

	// Create items from Shopify products.
	for i, p := range products {
		itemID := fmt.Sprintf("I%d", i+1)
		item := struct {
			ID         string `json:"id"`
			Descriptor struct {
				Name string `json:"name"`
			} `json:"descriptor"`
			Price struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"price"`
		}{
			ID: itemID,
			Descriptor: struct {
				Name string `json:"name"`
			}{Name: p.Title},
			Price: struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			}{
				Currency: "INR",
				Value:    p.Price,
			},
		}
		catalog.Providers[0].Items = append(catalog.Providers[0].Items, item)
	}

	return catalog
}

// sendOnSearch sends the ONDC on_search response to the BAP's on_search endpoint.
func sendOnSearch(bapURI string, catalog ONDCCatalog, req ONDCSearchRequest) error {
	response := struct {
		Context struct {
			Domain        string `json:"domain"`
			Country       string `json:"country"`
			City          string `json:"city"`
			Action        string `json:"action"`
			CoreVersion   string `json:"core_version"`
			BapID         string `json:"bap_id"`
			BapURI        string `json:"bap_uri"`
			BppID         string `json:"bpp_id"`
			BppURI        string `json:"bpp_uri"`
			TransactionID string `json:"transaction_id"`
			MessageID     string `json:"message_id"`
			Timestamp     string `json:"timestamp"`
		} `json:"context"`
		Message struct {
			Catalog ONDCCatalog `json:"catalog"`
		} `json:"message"`
	}{}

	// Populate the context using the incoming request and environment variables.
	response.Context.Domain = req.Context.Domain
	response.Context.Country = req.Context.Country
	response.Context.City = req.Context.City
	response.Context.Action = "on_search"
	response.Context.CoreVersion = "1.2.0"
	response.Context.BapID = req.Context.BapID
	response.Context.BapURI = req.Context.BapURI
	response.Context.BppID = os.Getenv("BPP_ID")
	response.Context.BppURI = os.Getenv("BPP_URI")
	response.Context.TransactionID = req.Context.TransactionID
	response.Context.MessageID = req.Context.MessageID
	response.Context.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	response.Message.Catalog = catalog

	// Format the JSON payload with indentation.
	prettyPayload, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal on_search response: %v", err)
	}

	log.Printf("Sending on_search to %s with payload:\n%s", bapURI+"/on_search", string(prettyPayload))

	resp, err := http.Post(bapURI+"/on_search", "application/json", bytes.NewBuffer(prettyPayload))
	if err != nil {
		return fmt.Errorf("failed to send on_search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("on_search failed with status: %s", resp.Status)
	}

	log.Printf("Successfully sent on_search response for message ID: %s", req.Context.MessageID)
	return nil
}


// transformToONDCSelectResponse converts the Shopify response into an ONDC select response.
// transformToONDCCSelectResponse converts the Shopify response into an ONDC select response.
// transformToONDCCSelectResponse converts the Shopify response into an ONDC select response.
func transformToONDCCSelectResponse(req ONDCSelectRequest, shopifyResp ShopifyResponse) ONDCSelectResponse {
	response := ONDCSelectResponse{
		Context: ONDCContext{
			Domain:        "ONDC:RET11",
			Action:        "on_select",
			CoreVersion:   "1.2.0",
			BapID:         req.Context.BapID,
			BapURI:        req.Context.BapURI,
			BppID:         req.Context.BppID,
			BppURI:        req.Context.BppURI,
			TransactionID: req.Context.TransactionID,
			MessageID:     req.Context.MessageID,
			City:          req.Context.City,
			Country:       req.Context.Country,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Set provider ID.
	response.Message.Order.Provider.ID = "P1"

	// Transform items.
	var totalValue float64
	for _, edge := range shopifyResp.Data.Products.Edges {
		price := edge.Node.Variants.Edges[0].Node.Price
		priceFloat, _ := strconv.ParseFloat(price, 64)
		totalValue += priceFloat

		item := struct {
			ID            string `json:"id"`
			FulfillmentID string `json:"fulfillment_id"`
			Quantity      struct {
				Available int `json:"available"`
				Maximum   int `json:"maximum"`
			} `json:"quantity"`
			Price struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"price"`
			Breakup []struct {
				Title string `json:"title"`
				Price struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				} `json:"price"`
			} `json:"breakup"`
		}{
			ID:            edge.Node.ID,
			FulfillmentID: "F1617",
			Quantity: struct {
				Available int `json:"available"`
				Maximum   int `json:"maximum"`
			}{
				Available: 5,
				Maximum:   3,
			},
			Price: struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			}{
				Currency: "INR",
				Value:    price,
			},
			Breakup: []struct {
				Title string `json:"title"`
				Price struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				} `json:"price"`
			}{
				{
					Title: fmt.Sprintf("Base Item - %s", edge.Node.Title),
					Price: struct {
						Currency string `json:"currency"`
						Value    string `json:"value"`
					}{
						Currency: "INR",
						Value:    price,
					},
				},
			},
		}
		response.Message.Order.Items = append(response.Message.Order.Items, item)
	}

	// Process offers if available.
	var totalOfferDiscount float64
	// Initialize the quote breakup slice if not already.
	response.Message.Order.Quote.Breakup = []struct {
		Title string `json:"title"`
		Price struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
	}{}
	
	// Get discount percentage from environment
	discountPercentStr := os.Getenv("OFFER_DISCOUNT")
	if discountPercentStr == "" {
		discountPercentStr = "10.00" // Default 10% discount
	}
	discountPercent, err := strconv.ParseFloat(discountPercentStr, 64)
	if err != nil {
		discountPercent = 10.0 // Default to 10% if parsing fails
	}

	for _, offer := range req.Message.Order.Offers {
		for _, tag := range offer.Tags {
			for _, tagItem := range tag.List {
				if tagItem.Code == "apply" && tagItem.Value == "yes" {
					// Calculate discount amount based on percentage of total value
					discount := (totalValue * discountPercent) / 100.0
					totalOfferDiscount += discount

					// Append offer breakup line.
					response.Message.Order.Quote.Breakup = append(response.Message.Order.Quote.Breakup, struct {
						Title string `json:"title"`
						Price struct {
							Currency string `json:"currency"`
							Value    string `json:"value"`
						} `json:"price"`
					}{
						Title: fmt.Sprintf("Offer - %s (%g%%)", offer.ID, discountPercent),
						Price: struct {
							Currency string `json:"currency"`
							Value    string `json:"value"`
						}{
							Currency: "INR",
							Value:    fmt.Sprintf("-%.2f", discount),
						},
					})
				}
			}
		}
	}
	totalValue -= totalOfferDiscount

	// Get delivery fee from configuration.
	deliveryFeeStr := os.Getenv("DELIVERY_FEE")
	if deliveryFeeStr == "" {
		deliveryFeeStr = "30.00"
	}
	deliveryCharge, err := strconv.ParseFloat(deliveryFeeStr, 64)
	if err != nil {
		deliveryCharge = 30.0
	}
	totalValue += deliveryCharge

	// Add fulfillment (delivery charge).
	fulfillment := struct {
		ID            string `json:"id"`
		Type          string `json:"type"`
		ProviderName  string `json:"@ondc/org/provider_name"`
		Tracking      bool   `json:"tracking"`
		Category      string `json:"@ondc/org/category"`
		TAT           string `json:"@ondc/org/TAT"`
		State         struct {
			Descriptor struct {
				Code string `json:"code"`
			} `json:"descriptor"`
		} `json:"state"`
		Price struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
	}{
		ID:           "F1617",
		Type:         "Delivery",
		ProviderName: "LSP Delivery",
		Tracking:     false,
		Category:     "Immediate Delivery",
		TAT:          "PT60M",
		State: struct {
			Descriptor struct {
				Code string `json:"code"`
			} `json:"descriptor"`
		}{
			Descriptor: struct {
				Code string `json:"code"`
			}{
				Code: "Serviceable",
			},
		},
		Price: struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		}{
			Currency: "INR",
			Value:    fmt.Sprintf("%.2f", deliveryCharge),
		},
	}
	response.Message.Order.Fulfillments = append(response.Message.Order.Fulfillments, fulfillment)

	// Add quote.
	response.Message.Order.Quote = struct {
		Price struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
		Breakup []struct {
			Title string `json:"title"`
			Price struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"price"`
		} `json:"breakup"`
	}{
		Price: struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		}{
			Currency: "INR",
			Value:    fmt.Sprintf("%.2f", totalValue),
		},
		Breakup: []struct {
			Title string `json:"title"`
			Price struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"price"`
		}{
			{
				Title: "Product Total",
				Price: struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				}{
					Currency: "INR",
					Value:    fmt.Sprintf("%.2f", totalValue-deliveryCharge),
				},
			},
			{
				Title: "Delivery Charge",
				Price: struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				}{
					Currency: "INR",
					Value:    fmt.Sprintf("%.2f", deliveryCharge),
				},
			},
		},
	}

	return response
}


func sendOnSelect(bapURI string, response ONDCSelectResponse, req ONDCSelectRequest) error {
    log.Printf("Sending on_select response for transaction %s", req.Context.TransactionID)

    jsonResponse, err := json.MarshalIndent(response, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal on_select response: %v", err)
    }

    log.Printf("on_select Response: %s", string(jsonResponse))

    resp, err := http.Post(bapURI+"/on_select", "application/json", bytes.NewBuffer(jsonResponse))
    if err != nil {
        return fmt.Errorf("failed to send on_select: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("on_select failed with status: %s", resp.Status)
    }

    log.Printf("Successfully sent on_select response for transaction ID: %s", req.Context.TransactionID)
    return nil
}



// transformToONDCInitResponse constructs the /on_init response using selected Shopify product data.
// It builds items, quote, billing, cancellation_terms, and a dynamic payment section.
func transformToONDCInitResponse(req ONDCInitRequest, shopifyResp ShopifyResponse) ONDCInitResponse {
	var response ONDCInitResponse

	// Populate context.
	response.Context = req.Context
	response.Context.Action = "on_init"
	response.Context.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Provider details.
	response.Message.Order.Provider.ID = "P1"
	response.Message.Order.Provider.Locations = []struct {
		ID string `json:"id"`
	}{
		{ID: "L1"},
	}

	// Prepare accumulators.
	var totalProductPrice float64
	response.Message.Order.Items = []struct {
		ID            string `json:"id"`
		FulfillmentID string `json:"fulfillment_id"`
		Quantity      struct {
			Count int `json:"count"`
		} `json:"quantity"`
		ParentItemID string `json:"parent_item_id"`
		Tags         []struct {
			Code string `json:"code"`
			List []struct {
				Code  string `json:"code"`
				Value string `json:"value"`
			} `json:"list"`
		} `json:"tags"`
	}{}

	// Process each requested item by matching with Shopify product data.
	for _, reqItem := range req.Message.Order.Items {
		for _, productEdge := range shopifyResp.Data.Products.Edges {
			// Here, reqItem.ID is expected to match the product meta tag value from Shopify.
			if productEdge.Node.ID == reqItem.ID {
				// Parse product price.
				productPriceStr := productEdge.Node.Variants.Edges[0].Node.Price
				productPrice, _ := strconv.ParseFloat(productPriceStr, 64)
				quantity := reqItem.Quantity.Count
				if quantity == 0 {
					quantity = 1
				}
				totalProductPrice += productPrice * float64(quantity)

				// Append order item using the dynamic product ID.
				response.Message.Order.Items = append(response.Message.Order.Items, struct {
					ID            string `json:"id"`
					FulfillmentID string `json:"fulfillment_id"`
					Quantity      struct {
						Count int `json:"count"`
					} `json:"quantity"`
					ParentItemID string `json:"parent_item_id"`
					Tags         []struct {
						Code string `json:"code"`
						List []struct {
							Code  string `json:"code"`
							Value string `json:"value"`
						} `json:"list"`
					} `json:"tags"`
				}{
					ID:            productEdge.Node.ID,
					FulfillmentID: reqItem.FulfillmentID,
					Quantity:      struct{ Count int `json:"count"` }{Count: quantity},
					ParentItemID:  reqItem.ParentItemID,
					Tags:          reqItem.Tags,
				})
			}
		}
	}

	// Read delivery fee from environment (default to 100.00 if not set).
	deliveryFeeStr := os.Getenv("DELIVERY_FEE")
	if deliveryFeeStr == "" {
		deliveryFeeStr = "100.00"
	}
	deliveryFee, err := strconv.ParseFloat(deliveryFeeStr, 64)
	if err != nil {
		deliveryFee = 100.0
	}
	totalOrderPrice := totalProductPrice + deliveryFee

	// Build the quote breakup.
	var quoteBreakup []struct {
		ItemID       string `json:"@ondc/org/item_id"`
		ItemQuantity struct {
			Count int `json:"count"`
		} `json:"@ondc/org/item_quantity"`
		Title     string `json:"title"`
		TitleType string `json:"@ondc/org/title_type"`
		Price     struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
	}
	for _, reqItem := range req.Message.Order.Items {
		for _, productEdge := range shopifyResp.Data.Products.Edges {
			if productEdge.Node.ID == reqItem.ID {
				productPriceStr := productEdge.Node.Variants.Edges[0].Node.Price
				productPrice, _ := strconv.ParseFloat(productPriceStr, 64)
				quantity := reqItem.Quantity.Count
				if quantity == 0 {
					quantity = 1
				}
				quoteBreakup = append(quoteBreakup, struct {
					ItemID       string `json:"@ondc/org/item_id"`
					ItemQuantity struct {
						Count int `json:"count"`
					} `json:"@ondc/org/item_quantity"`
					Title     string `json:"title"`
					TitleType string `json:"@ondc/org/title_type"`
					Price     struct {
						Currency string `json:"currency"`
						Value    string `json:"value"`
					} `json:"price"`
				}{
					ItemID:       productEdge.Node.ID,
					ItemQuantity: struct{ Count int `json:"count"` }{Count: quantity},
					Title:        productEdge.Node.Title,
					TitleType:    "item",
					Price: struct {
						Currency string `json:"currency"`
						Value    string `json:"value"`
					}{
						Currency: "INR",
						Value:    fmt.Sprintf("%.2f", productPrice*float64(quantity)),
					},
				})
			}
		}
	}
	// Append a breakup entry for delivery charges.
	quoteBreakup = append(quoteBreakup, struct {
		ItemID       string `json:"@ondc/org/item_id"`
		ItemQuantity struct {
			Count int `json:"count"`
		} `json:"@ondc/org/item_quantity"`
		Title     string `json:"title"`
		TitleType string `json:"@ondc/org/title_type"`
		Price     struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
	}{
		ItemID:       "F1",
		ItemQuantity: struct{ Count int `json:"count"` }{Count: 1},
		Title:        "Delivery charges",
		TitleType:    "delivery",
		Price: struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		}{
			Currency: "INR",
			Value:    fmt.Sprintf("%.2f", deliveryFee),
		},
	})

	// Build the overall Quote.
	response.Message.Order.Quote = struct {
		Price struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		} `json:"price"`
		Breakup []struct {
			ItemID       string `json:"@ondc/org/item_id"`
			ItemQuantity struct {
				Count int `json:"count"`
			} `json:"@ondc/org/item_quantity"`
			Title     string `json:"title"`
			TitleType string `json:"@ondc/org/title_type"`
			Price     struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"price"`
		} `json:"breakup"`
	}{
		Price: struct {
			Currency string `json:"currency"`
			Value    string `json:"value"`
		}{
			Currency: "INR",
			Value:    fmt.Sprintf("%.2f", totalOrderPrice),
		},
		Breakup: quoteBreakup,
	}

	// Billing details: make email and phone dynamic.
	billingEmail := os.Getenv("BILLING_EMAIL")
	if billingEmail == "" {
		billingEmail = "nobody@nomail.com"
	}
	billingPhone := os.Getenv("BILLING_PHONE")
	if billingPhone == "" {
		billingPhone = "9886098860"
	}
	// Address and name can remain static or be updated similarly.
	response.Message.Order.Billing = struct {
		Name    string `json:"name"`
		Address struct {
			Name     string `json:"name"`
			Building string `json:"building"`
			Locality string `json:"locality"`
			City     string `json:"city"`
			State    string `json:"state"`
			Country  string `json:"country"`
			AreaCode string `json:"area_code"`
		} `json:"address"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}{
		Name: "ONDC buyer",
		Address: struct {
			Name     string `json:"name"`
			Building string `json:"building"`
			Locality string `json:"locality"`
			City     string `json:"city"`
			State    string `json:"state"`
			Country  string `json:"country"`
			AreaCode string `json:"area_code"`
		}{
			Name:     "my house or door or floor #",
			Building: "my building name or house #",
			Locality: "my street name",
			City:     "Bengaluru",
			State:    "Karnataka",
			Country:  "IND",
			AreaCode: "560037",
		},
		Email: billingEmail,
		Phone: billingPhone,
	}

	// Cancellation terms (static defaults).
	response.Message.Order.CancellationTerms = []struct {
		FulfillmentState struct {
			Descriptor struct {
				Code      string `json:"code"`
				ShortDesc string `json:"short_desc"`
			} `json:"descriptor"`
		} `json:"fulfillment_state"`
		CancellationFee struct {
			Percentage string `json:"percentage"`
			Amount     struct {
				Currency string `json:"currency"`
				Value    string `json:"value"`
			} `json:"amount"`
		} `json:"cancellation_fee"`
	}{
		{
			FulfillmentState: struct {
				Descriptor struct {
					Code      string `json:"code"`
					ShortDesc string `json:"short_desc"`
				} `json:"descriptor"`
			}{
				Descriptor: struct {
					Code      string `json:"code"`
					ShortDesc string `json:"short_desc"`
				}{
					Code:      "Pending",
					ShortDesc: "002",
				},
			},
			CancellationFee: struct {
				Percentage string `json:"percentage"`
				Amount     struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				} `json:"amount"`
			}{
				Percentage: "0.00",
				Amount: struct {
					Currency string `json:"currency"`
					Value    string `json:"value"`
				}{
					Currency: "INR",
					Value:    "0.00",
				},
			},
		},
	}

	// Payment: Build dynamically using environment variables.
	response.Message.Order.Payment = struct {
		Uri                      string `json:"uri"`
		BuyerAppFinderFeeType    string `json:"@ondc/org/buyer_app_finder_fee_type"`
		BuyerAppFinderFeeAmount  string `json:"@ondc/org/buyer_app_finder_fee_amount"`
		WithholdingAmount        string `json:"@ondc/org/withholding_amount"`
		Tags                     []struct {
			Code string `json:"code"`
			List []struct {
				Code  string `json:"code"`
				Value string `json:"value"`
			} `json:"list"`
		} `json:"tags"`
	}{
		Uri:                     os.Getenv("PAYMENT_URI"),                     // e.g., "https://snp.com/pg"
		BuyerAppFinderFeeType:   os.Getenv("BUYER_APP_FINDER_FEE_TYPE"),       // e.g., "percent"
		BuyerAppFinderFeeAmount: os.Getenv("BUYER_APP_FINDER_FEE_AMOUNT"),     // e.g., "3"
		WithholdingAmount:       os.Getenv("WITHHOLDING_AMOUNT"),              // e.g., "10.00"
		Tags: []struct {
			Code string `json:"code"`
			List []struct {
				Code  string `json:"code"`
				Value string `json:"value"`
			} `json:"list"`
		}{
			{
				Code: "bpp_collect",
				List: []struct {
					Code  string `json:"code"`
					Value string `json:"value"`
				}{
					{Code: "success", Value: os.Getenv("BPP_COLLECT_SUCCESS")}, // e.g., "Y"
					{Code: "error", Value: os.Getenv("BPP_COLLECT_ERROR")},     // e.g., ".."
				},
			},
		},
	}

	return response
}



// sendOnInit sends the ONDC on_init response to the BAP's on_init endpoint.
func sendOnInit(bapURI string, response ONDCInitResponse, req ONDCInitRequest) error {
    log.Printf("Sending on_init response for transaction %s", req.Context.TransactionID)

    jsonResponse, err := json.MarshalIndent(response, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal on_init response: %v", err)
    }

    log.Printf("on_init Response: %s", string(jsonResponse))

    resp, err := http.Post(bapURI+"/on_init", "application/json", bytes.NewBuffer(jsonResponse))
    if err != nil {
        return fmt.Errorf("failed to send on_init: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("on_init failed with status: %s", resp.Status)
    }

    log.Printf("Successfully sent on_init response for transaction ID: %s", req.Context.TransactionID)
    return nil
}
