package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
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

type Geocode_t struct {
	PlaceID     int    `json:"place_id"`
	Licence     string `json:"licence"`
	OsmType     string `json:"osm_type"`
	OsmID       int64  `json:"osm_id"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Address     struct {
		HouseNumber  string `json:"house_number"`
		Road         string `json:"road"`
		Town         string `json:"town"`
		County       string `json:"county"`
		State        string `json:"state"`
		ISO31662Lvl4 string `json:"ISO3166-2-lvl4"`
		Postcode     string `json:"postcode"`
		Country      string `json:"country"`
		CountryCode  string `json:"country_code"`
	} `json:"address"`
	Boundingbox []string `json:"boundingbox"`
}

const ConfigUrl = "https://outagemap.duke-energy.com/config/config.prod.json"
const CountiesUrl = "https://prod.apigee.duke-energy.app/outage-maps/v1/counties?jurisdiction="
const OutageUrl = "https://prod.apigee.duke-energy.app/outage-maps/v1/outages?jurisdiction="

func FetchAndUnmarshal[T any](url string, authHeader string) (*T, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	if authHeader != "" {
		request.Header.Add("Authorization", authHeader)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jsonBytes T
	if err := json.Unmarshal(body, &jsonBytes); err != nil {
		log.Fatal(err)
	}

	return &jsonBytes, nil
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	// Init env var
	serviceArea := strings.Split(os.Getenv("SERVICE_AREA"), ",")
	apiKey := os.Getenv("API_KEY")
	jurisdiction := os.Getenv("JURISDICTION")
	c_url := CountiesUrl + jurisdiction
	o_url := OutageUrl + jurisdiction

	// Init sqlite table
	db, err := sql.Open("sqlite3", "./outages.db")
	if err != nil {
		print("1")
	}
	defer db.Close()

	initTable := `
	CREATE TABLE IF NOT EXISTS outages (
	id INTEGER NOT NULL PRIMARY KEY,
	county TEXT,
	customers_affected INTEGER NOT NULL
	);`

	_, err = db.Exec(initTable)
	if err != nil {
		print("2")
	}

	var config Config_t

	request, err := http.Get(ConfigUrl)
	if err != nil {
		print("3")
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		print("4")
	}

	err = json.Unmarshal(body, &config)
	if err != nil {
		print("5")
	}

	authKey := []byte(config.Consumer_key_emp + ":" + config.Consumer_secret_emp)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString(authKey)

	countiesResponse, err := http.NewRequest(http.MethodGet, c_url, nil)
	if err != nil {
		print("6")
	}

	countiesResponse.Header.Add("Authorization", authHeader)

	outagesResponse, err := http.NewRequest(http.MethodGet, o_url, nil)
	if err != nil {
		print("7")
	}

	outagesResponse.Header.Add("Authorization", authHeader)

	county, err := FetchAndUnmarshal[County_t](c_url, authHeader)
	if err != nil {
		print("8")
	}

	outage, err := FetchAndUnmarshal[Outage_t](o_url, authHeader)
	if err != nil {
		print("9")
	}

	for _, c := range county.Data {
		for _, x := range serviceArea {
			if c.CountyName == x {
				if c.AreaOfInterestSummary.ActiveEventsCount > 0 {
					fmt.Printf("%s, Customers Served: %d, Active Outage Count: %d, Customers Affected: %d\n", c.AreaOfInterestName, c.CustomersServed, c.AreaOfInterestSummary.ActiveEventsCount, c.AreaOfInterestSummary.MaxCustomersAffected)
				} else {
					fmt.Printf("%s, Customers Served: %d No Active Outages\n", c.AreaOfInterestName, c.CustomersServed)
				}
			} else {
				continue
			}
		}
	}
	for _, o := range outage.Data {
		geocodeUrl := fmt.Sprintf("https://geocode.maps.co/reverse?lat=%f&lon=%f&api_key=%s", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation, apiKey)

		rGeocodedData, err := FetchAndUnmarshal[Geocode_t](geocodeUrl, "")
		if err != nil {
			print("10")
		}
		time.Sleep(1 * time.Second)

		list := strings.Fields(rGeocodedData.Address.County)
		for _, x := range serviceArea {
			if list[0] == x {
				print(rGeocodedData.Address.County)
				_, err := db.Exec("INSERT INTO outages (id, county, customers_affected) VALUES(?, ?, ?)", o.SourceEventNumber, list[0], o.CustomersAffectedNumber)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("\nLatitude %f\nLongitude %f\nCustomers Affected: %d\nEvent ID: %s\nOutage Type: %s\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation, o.CustomersAffectedNumber, o.SourceEventNumber, o.OutageCause)
				outageUrl := fmt.Sprintf("https://www.google.com/maps/search/%f,+%f?sa=X&ved=1t:242&ictx=111\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation)
				fmt.Printf("Outage Location URL: " + outageUrl)
				fmt.Printf("Address of Outage: %s %s, %s\n", rGeocodedData.Address.HouseNumber, rGeocodedData.Address.Road, rGeocodedData.Address.Town)
				for _, y := range o.ConvexHull {
					fmt.Printf("%f\n", y)
				}

			} else {
				continue
			}
		}

	}

}
