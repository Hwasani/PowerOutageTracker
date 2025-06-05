package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config_t struct {
	Emp                 string `json:"emp"`
	Consumer_key_emp    string `json:"consumer_key_emp"`
	Consumer_secret_emp string `json:"consumer_secret_emp"`
	Eport_outage_url    string `json:"report_outage_url"`
	Get_outage_alerts   string `json:"get_outage_alerts"`
	Req_light_repair    string `json:"req_light_repair"`
	Res_your_power      string `json:"res_your_power"`
	Genr_safety         string `json:"genr_safety"`
	Meter_damage        string `json:"meter_damage"`
	Faq                 string `json:"faq"`
	Contact_us          string `json:"contact_us"`
	Outages_url         string `json:"outages_url"`
}

type County_t struct {
	Data []struct {
		AreaOfInterestID            int       `json:"areaOfInterestId"`
		AreaOfInterestName          string    `json:"areaOfInterestName"`
		CustomersServed             int       `json:"customersServed"`
		EtrOverride                 any       `json:"etrOverride"`
		CauseCodeOverride           any       `json:"causeCodeOverride"`
		CrewStatusOverride          any       `json:"crewStatusOverride"`
		CustomersAffectedOverride   any       `json:"customersAffectedOverride"`
		LastUpdated                 time.Time `json:"lastUpdated"`
		ServiceAreaOverrideIsActive bool      `json:"serviceAreaOverrideIsActive"`
		Jurisdiction                string    `json:"jurisdiction"`
		ServiceAreas                []any     `json:"serviceAreas"`
		State                       string    `json:"state"`
		CountyName                  string    `json:"countyName"`
		AreaOfInterestSummary       struct {
			AreaOfInterestID      int `json:"areaOfInterestId"`
			MaxCustomersAffected  int `json:"maxCustomersAffected"`
			ActiveEventsCount     int `json:"activeEventsCount"`
			RestoredEventsCount   any `json:"restoredEventsCount"`
			LatestRestorationDate any `json:"latestRestorationDate"`
			FirstReportedDate     any `json:"firstReportedDate"`
		} `json:"areaOfInterestSummary"`
	} `json:"data"`
	ErrorMessages []any `json:"errorMessages"`
}

type Outage_t struct {
	Data []struct {
		SourceEventNumber       string  `json:"sourceEventNumber"`
		DeviceLatitudeLocation  float64 `json:"deviceLatitudeLocation"`
		DeviceLongitudeLocation float64 `json:"deviceLongitudeLocation"`
		CustomersAffectedNumber int     `json:"customersAffectedNumber"`
		ConvexHull              []struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"convexHull"`
		OutageCause string `json:"outageCause"`
	} `json:"data"`
	ErrorMessages []any `json:"errorMessages"`
}

const ConfigUrl = "https://outagemap.duke-energy.com/config/config.prod.json"
const CountiesUrl = "https://prod.apigee.duke-energy.app/outage-maps/v1/counties?jurisdiction=DEF"
const OutageUrl = "https://prod.apigee.duke-energy.app/outage-maps/v1/outages?jurisdiction=DEF"

func FetchAndUnmarshal[T any](url string, authHeader string) (*T, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", authHeader)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var jsonBytes T
	if err := json.Unmarshal(body, &jsonBytes); err != nil {
		return nil, err
	}

	return &jsonBytes, nil
}

func main() {
	// config_url := "https://outagemap.duke-energy.com/config/config.prod.json"
	// counties_url := "https://prod.apigee.duke-energy.app/outage-maps/v1/counties?jurisdiction=DEF"
	// outage_url := "https://prod.apigee.duke-energy.app/outage-maps/v1/outages?jurisdiction=DEF"

	var config Config_t
	// var county County_t

	request, err := http.Get(ConfigUrl)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Println("Error 2: ", err)
		return
	}

	err = json.Unmarshal(body, &config)
	if err != nil {
		fmt.Print(err)
		return
	}

	authKey := []byte(config.Consumer_key_emp + ":" + config.Consumer_secret_emp)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString(authKey)

	// this is where we will repeat the get counties/outages functions

	countiesResponse, err := http.NewRequest(http.MethodGet, CountiesUrl, nil)
	if err != nil {
		fmt.Print(err)
		return
	}

	countiesResponse.Header.Add("Authorization", authHeader)

	outagesResponse, err := http.NewRequest(http.MethodGet, OutageUrl, nil)
	if err != nil {
		fmt.Print(err)
		return
	}

	outagesResponse.Header.Add("Authorization", authHeader)
	// the plan is to get the JSON data from this GET request, then get the lo la of the outages and map them in google maps or duke.
	// That way if theres an outage we should be aware of we can just click that and know at a glance

	// response, err := http.DefaultClient.Do(countiesResponse)
	// if err != nil {
	// 	print(err)
	// 	return
	// }
	// responseBody, err := io.ReadAll(response.Body)
	// if err != nil {
	// 	fmt.Print(err)
	// 	return
	// }

	county, err := FetchAndUnmarshal[County_t](CountiesUrl, authHeader)
	if err != nil {
		print(err)
	}

	outage, err := FetchAndUnmarshal[Outage_t](OutageUrl, authHeader)
	if err != nil {
		print(err)
	}

	for _, c := range county.Data {

		if c.AreaOfInterestSummary.ActiveEventsCount > 0 {
			fmt.Printf("%s, Customers Served: %d, Active Outage Count: %d, Customers Affected: %d\n", c.AreaOfInterestName, c.CustomersServed, c.AreaOfInterestSummary.ActiveEventsCount, c.AreaOfInterestSummary.MaxCustomersAffected)
		} else {
			fmt.Printf("%s, Customers Served: %d No Active Outages\n", c.AreaOfInterestName, c.CustomersServed)
		}

	}
	for _, o := range outage.Data {

		fmt.Printf("\nLatitude %f\nLongitude %f\nCustomers Affected: %d\nEvent ID: %s\nOutage Type: %s\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation, o.CustomersAffectedNumber, o.SourceEventNumber, o.OutageCause)
		fmt.Printf("Outage Location URL: https://www.google.com/maps/search/%f,+%f?sa=X&ved=1t:242&ictx=111\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation)
	}
	// https: //www.google.com/maps/search/41.36595626517665,+-108.70481231362976?sa=X&ved=1t:242&ictx=111

}
