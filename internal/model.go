package main

import (
	"time"

	fb "github.com/huandu/facebook/v2"
)

// KeyMetadata describes the important fields to extract from Graph API response
type KeyMetadata struct {
	CreatedTime time.Time `json:"created_time"`
	FacebookID  string    `json:"id"`
	UpdatedTime time.Time `json:"updated_time"`
}

// PostData describes the data to be inserted into DB
type PostData struct {
	CreatedTime  time.Time `json:"created_time"`
	FacebookID   string    `json:"facebook_id"`
	FacebookPost fb.Result `json:"post"`
	IsParsed     string    `json:"is_parsed"`
}

// C3poRequest describes the request body sent to C-3PO
type C3poRequest struct {
	FacebookPost fb.Result `json:"facebook_post"`
}

// C3poResponse describes the response from C-3PO POST request
type C3poResponse struct {
	Success bool `json:"success"`
}
