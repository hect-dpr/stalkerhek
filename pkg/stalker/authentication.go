package stalker

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

// Handshake reserves a offered token in Portal. If offered token is not available - new one will be issued by stalker portal, reservedMAG254 and Stalker's config will be updated.
func (p *Portal) handshake() error {
	// This HTTP request has different headers from the rest of HTTP requests, so perform it manually
	type tmpStruct struct {
		Js map[string]interface{} `json:"js"`
	}
	var tmp tmpStruct

	req, err := http.NewRequest("GET", p.Location+"?type=stb&action=handshake&JsHttpRequest=1-xml", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C)")
	req.Header.Set("X-User-Agent", "Model: "+p.Model+"; Link: Ethernet")
	req.Header.Set("Cookie", "mac="+p.MAC+"; stb_lang=en; timezone="+p.TimeZone)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(contents, &tmp); err != nil {
		log.Println(string(contents))
		return err
	}

	token, ok := tmp.Js["token"]

	if !ok || token == "" {
		// Token accepted. Using accepted token
		return nil
	}
	// Server provided new token. Using new provided token
	p.Token = token.(string)
	return nil
}

// Authenticate associates credentials with token. In other words - logs you in
func (p *Portal) authenticate() (err error) {
	// This HTTP request has different headers from the rest of HTTP requests, so perform it manually
	type tmpStruct struct {
		Js bool `json:"js"`
	}
	var tmp tmpStruct

	content, err := p.httpRequest(p.Location + "?type=stb&action=do_auth&login=" + p.Username + "&password=" + p.Password + "&device_id=" + p.DeviceID + "&device_id2=" + p.DeviceID2 + "&JsHttpRequest=1-xml")

	if err = json.Unmarshal(content, &tmp); err != nil {
		return err
	}

	if tmp.Js {
		return nil
	}
	return errors.New("invalid credentials")
}
