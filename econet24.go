package goeconet24

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Params struct {
	PumpCOWorks   bool    `json:"pumpCOWorks"`
	BoilerPower   int     `json:"boilerPower"`
	BoilerPowerKW float32 `json:"boilerPowerKW"`
	TempCOSet     float32 `json:"tempCOSet"`
	TempCO        float32 `json:"tempCO"`
	TempCWUSet    float32 `json:"tempCWUSet"`
	TempCWU       float32 `json:"tempCWU"`
	TempFeeder    float32 `json:"tempFeeder"`
	FanWorks      bool    `json:"fanWorks"`
	FuelStream    float32 `json:"fuelStream"`
	FuelLevel     int     `json:"fuelLevel"`
	OperationMode int     `json:"mode"`
}

type Econet24 interface {
	getRequest(cmd string) (*http.Request, error)
	setParam(element interface{}, value int, command CommandType) error
	ChangeHUWStatus(status int) error
	ChangeBoilerStatus(status BoilerStatus) error
	GetDeviceRegParams() (Params, error)
}

type econet struct {
	client              *http.Client
	hostname            string
	uid                 string
	logger              *slog.Logger
	csrdmiddlewaretoken string
}

func (e econet) ChangeBoilerStatus(status BoilerStatus) error {
	if err := e.setParam("BOILER_STATUS", int(status), NewParamName); err != nil {
		return fmt.Errorf("change boiler status: %w", err)
	}
	return nil
}

func (e econet) ChangeHUWStatus(status int) error {
	if err := e.setParam(HUWHeater, status, NewParamIndex); err != nil {
		return fmt.Errorf("change HUW status failed: %w", err)
	}
	return nil
}

func (e econet) setParam(element any, value int, command CommandType) error {
	var cmd, service string

	switch command {
	case NewParamKey:
		cmd = "newParamKey"
		service = "rmCurrNewParam"
	case NewParamIndex:
		cmd = "newParamIndex"
		service = "rmNewParam"
	case NewParamName:
		cmd = "newParamName"
		service = "newParam"
	}

	now := time.Now()
	c := fmt.Sprintf("%s?uid=%s&%s=%v&newParamValue=%d&_=%d", service, e.uid, cmd, element, value, now.Unix())
	req, err := e.getRequest(c)
	if err != nil {
		return err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("test")
	}
	fmt.Println(string(body))
	return nil
}
func (e econet) GetDeviceRegParams() (Params, error) {
	type Response struct {
		Param Params `json:"curr"`
	}
	r := Response{}
	cmd := fmt.Sprintf("getDeviceParams?uid=%s&_=%d", e.uid, time.Now().Unix())
	req, err := e.getRequest(cmd)
	if err != nil {
		return Params{}, fmt.Errorf("get device reg params: %w", err)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return Params{}, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Params{}, err
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return Params{}, err
	}
	return r.Param, nil
}

func (e econet) getRequest(cmd string) (*http.Request, error) {
	req, err := http.NewRequest("GET", e.hostname+"/service/"+cmd, nil)
	e.logger.Debug("getRequest", "URL", req.URL)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func NewEconet24(username, password, uid, hostname string, logger *slog.Logger) Econet24 {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		logger.Error("Failed to create cookie jar")
	}
	client := http.Client{
		Jar: jar,
	}
	resp, err := client.Get(hostname)
	if err != nil {
		logger.Error("error with opening econet24.com page")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logger.Error("Błąd strony: ", resp.StatusCode, resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("Unable to load document: ", err)
	}

	csrfToken, exists := doc.Find("input[name='csrfmiddlewaretoken']").Attr("value")
	if exists {
		logger.Debug("get CSRF token", "CSRF Token: ", csrfToken)
	} else {
		logger.Error("Nie znaleziono CSRF Token")
	}
	formData := url.Values{
		"csrfmiddlewaretoken": []string{csrfToken},
		"username":            []string{username},
		"password":            []string{password},
	}

	var payload = bytes.NewBufferString(formData.Encode())
	request, err := http.NewRequest("POST", hostname+"/login/?next=main", payload)
	if err != nil {
		logger.Error("Unable to create request: ", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(request)
	if err != nil {
		logger.Error("Unable to make request: ", err)
	}
	if resp.StatusCode != 200 {
		logger.Error("Unable to log in econet24.com")
	}

	return &econet{
		client:              &client,
		uid:                 uid,
		logger:              logger,
		hostname:            hostname,
		csrdmiddlewaretoken: csrfToken,
	}
}
