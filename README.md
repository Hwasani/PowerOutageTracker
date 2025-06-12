# Power Outage Tracker
Duke Energy power outage tracker. WIP  
  
Make sure to have a .env with the following variables:  
API_KEY=12345 (Might not even use the geocode.maps.co API soon)  
SERVICE_AREA=County0,County1,County2  
JURISDICTION=DEF (This is for the Florida jursidiction)  
CGO_ENABLED=1  
  
Requires these imports and their dependencies  
github.com/joho/godotenv  
github.com/mattn/go-sqlite3  