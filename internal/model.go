package main

import (
	"time"

	fb "github.com/huandu/facebook/v2"
	"go.uber.org/zap/zapcore"
)

// KeyMetadata describes the important fields to extract from Graph API response
type KeyMetadata struct {
	CreatedTime time.Time `json:"created_time"`
	FacebookID  string    `json:"id"`
	UpdatedTime time.Time `json:"updated_time"`
}

// MarshalLogObject for KeyMetadata type
func (k KeyMetadata) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddTime("created_time", k.CreatedTime)
	encoder.AddString("id", k.FacebookID)
	encoder.AddTime("updated_time", k.UpdatedTime)
	return nil
}

// PostData describes the data to be inserted into DB
type PostData struct {
	CreatedTime  time.Time `json:"created_time"`
	FacebookID   string    `json:"facebook_id"`
	FacebookPost fb.Result `json:"post"`
	IsParsed     string    `json:"is_parsed"`
}

// MarshalLogObject for PostData type
func (p PostData) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddTime("created_time", p.CreatedTime)
	encoder.AddString("facebook_id", p.FacebookID)
	encoder.AddString("is_parsed", p.IsParsed)
	return nil
}

// C3poRequest describes the request body sent to C-3PO
type C3poRequest struct {
	FacebookPost fb.Result `json:"facebook_post"`
}

// C3poResponse describes the response from C-3PO POST request
type C3poResponse struct {
	Success bool `json:"success"`
}
