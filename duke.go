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

// Kind of an ambitious way to update these dbs. I didnt have it squared away how I was going to mark these outages as active or not yet

// func ActiveScanLoop(outageDataBase *sql.DB, cordDataBase *sql.DB, tableName string, sourceEvent string) {
// 	for {
// 		var eventID string
// 		err := outageDataBase.QueryRow("SELECT event_id FROM outages WHERE event_id = ?", sourceEvent).Scan(&eventID)
// 		if err == sql.ErrNoRows {
// 			_, execErr := outageDataBase.Exec("UPDATE outages SET active = 0 WHERE event_id = ?", sourceEvent)
// 			if execErr != nil {
// 				log.Printf("Insert failed: %v\n", execErr)
// 			} else {
// 				log.Printf("Deactivated event: %s\n", sourceEvent)
// 				_, err := cordDataBase.Exec("UPDATE coordinates SET active = 0 WHERE event_id = ?", sourceEvent)
// 				if err != nil {
// 					log.Printf("Error updating active field in coordinates table %e", err)
// 				}
// 			}
// 		} else if err != nil {
// 			log.Printf("Query failed: %v\n", err)
// 		}
// 		time.Sleep(1 * time.Second)
// 	}
// }

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

	// Init outage table
	outageDb, err := sql.Open("sqlite3", "./outages.db")
	if err != nil {
		log.Fatal("Failed to create table", err)
	}
	defer outageDb.Close()

	initOutageTable := `
	CREATE TABLE IF NOT EXISTS outages (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	active INTEGER NOT NULL,
	event_id INTEGER NOT NULL,
	county TEXT,
	customers_affected INTEGER NOT NULL
	);`

	_, err = outageDb.Exec(initOutageTable)
	if err != nil {
		log.Fatal("Failed to create table", err)
	}
	_, err = outageDb.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal("Failed to enable foreign keys:", err)
	}

	_, err = outageDb.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatal("Failed to set WAL mode:", err)
	}

	// Init outage coordinates table

	coordDb, err := sql.Open("sqlite3", "./coordinates.db")
	if err != nil {
		log.Fatal(err)
	}
	defer coordDb.Close()

	initCordsTable := `
	CREATE TABLE IF NOT EXISTS coordinates (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	event_id INTEGER NOT NULL,
	lat REAL NOT NULL,
	lon REAL NOT NULL,
	active INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY (event_id) REFERENCES outages(event_id) ON DELETE CASCADE
	);`

	_, err = coordDb.Exec(initCordsTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = coordDb.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Fatal("Failed to set WAL mode:", err)
	}

	var config Config_t

	request, err := http.Get(ConfigUrl)
	if err != nil {
		log.Fatal("Failed to reach config url. Check internet connection", err)
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Fatal("Failed to read the request body", err)
	}

	err = json.Unmarshal(body, &config)
	if err != nil {
		log.Fatal("Failed to unmarshal json bytes into data struct")
	}

	authKey := []byte(config.Consumer_key_emp + ":" + config.Consumer_secret_emp)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString(authKey)

	countiesResponse, err := http.NewRequest(http.MethodGet, c_url, nil)
	if err != nil {
		log.Fatal("Failed to reach counties url Check internet connection", err)
	}

	countiesResponse.Header.Add("Authorization", authHeader)

	outagesResponse, err := http.NewRequest(http.MethodGet, o_url, nil)
	if err != nil {
		log.Fatal("Failed to reach outages url. Check internet connection", err)
	}

	outagesResponse.Header.Add("Authorization", authHeader)

	county, err := FetchAndUnmarshal[County_t](c_url, authHeader)
	if err != nil {
		log.Fatal("Failed to run FetchAndUnmarshal on county url", err)
	}

	outage, err := FetchAndUnmarshal[Outage_t](o_url, authHeader)
	if err != nil {
		log.Fatal("Failed to run FetchAndUnmarshal on outage url", err)
	}

	// This entire bottom section will ultimately change in it's use case.
	// This was just a proof of concept that I could get the data and use it.

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

	parsedOutage := make(map[string]bool)
	for _, o := range outage.Data {
		geocodeUrl := fmt.Sprintf("https://geocode.maps.co/reverse?lat=%f&lon=%f&api_key=%s", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation, apiKey)

		rGeocodedData, err := FetchAndUnmarshal[Geocode_t](geocodeUrl, "")
		if err != nil {
			log.Fatal("Failed to run FetchAndUnmarshal on geocode url. Check your API key.", err)
		}
		time.Sleep(1 * time.Second)
		countyList := strings.Fields(rGeocodedData.Address.County)
		for _, x := range serviceArea {
			if countyList[0] == x {
				println(rGeocodedData.Address.County)
				parsedOutage[o.SourceEventNumber] = true
				_, err := outageDb.Exec("INSERT OR REPLACE INTO outages (event_id, active, county, customers_affected) VALUES(?, ?, ?, ?)", o.SourceEventNumber, 1, countyList[0], o.CustomersAffectedNumber)
				if err != nil {
					log.Fatal(err)
				}
				var convexHull = o.ConvexHull
				for _, j := range convexHull {
					_, err := coordDb.Exec("INSERT OR REPLACE INTO coordinates (event_id, lat, lon) VALUES(?,?,?)", o.SourceEventNumber, j.Lat, j.Lng)
					if err != nil {
						log.Fatal(err)
					}
				}

				fmt.Printf("\nLatitude %f\nLongitude %f\nCustomers Affected: %d\nEvent ID: %s\nOutage Type: %s\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation, o.CustomersAffectedNumber, o.SourceEventNumber, o.OutageCause)
				outageUrl := fmt.Sprintf("https://www.google.com/maps/search/%f,+%f?sa=X&ved=1t:242&ictx=111\n", o.DeviceLatitudeLocation, o.DeviceLongitudeLocation)
				fmt.Printf("Outage Location URL: %s", outageUrl)
				// fmt.Printf("Address of Outage: %s %s, %s\n", rGeocodedData.Address.HouseNumber, rGeocodedData.Address.Road, rGeocodedData.Address.Town) I thought this would work but this really sucks at giving addresses.
				for _, y := range o.ConvexHull {
					fmt.Printf("%f\n", y)
				}
			} else {
				continue
			}
		}
	}
	outageRows, err := outageDb.Query("SELECT event_id FROM outages WHERE active = 1")
	if err != nil {
		log.Fatal("Event_ID not in outageDB", err)
	}
	defer outageRows.Close()
	var count = 0
	fmt.Printf("Starting check for cleared outages.\n")
	for outageRows.Next() {
		var dbEventId string
		if err := outageRows.Scan(&dbEventId); err != nil {
			log.Fatal("Failed to scan", err)
		}

		if !parsedOutage[dbEventId] {
			log.Printf("Deactivating outage: %s\n", dbEventId)

			_, err := outageDb.Exec("UPDATE outages SET active = 0 WHERE event_id = ?", dbEventId)
			if err != nil {
				log.Fatalf("Failed to deactivate event %s from outageDb %v\n", dbEventId, err)
			}

			_, err = coordDb.Exec("UPDATE coordinates SET active = 0 WHERE event_id = ?", dbEventId)
			if err != nil {
				log.Fatalf("Failed to deactivate event %s from coordDb %v\n", dbEventId, err)
			}
			count += 1
		}
	}
	fmt.Printf("Finished checking for cleared outages.\n")

	fmt.Printf("Total count of outages cleared: %d\n", count)
}
