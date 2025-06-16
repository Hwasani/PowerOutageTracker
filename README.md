# Power Outage Tracker
Duke Energy power outage tracker. WIP  
  
Make sure to have a .env with the following variables:  
API_KEY=12345 (Might not even use the geocode.maps.co API soon)  
SERVICE_AREA=County0,County1,County2  
JURISDICTION=DEF (This is for the Florida jursidiction)  
CGO_ENABLED=1  
(If you're on ARM add these to the .env as well. This is a requirement from the sqlite3 driver I used)  
CC=arm-linux-gnueabihf-gcc  
CXX=arm-linux-gnueabihf-g++  
GOOS=linux  
GOARCH=arm  
GOARM=7  
  
Requires these imports and their dependencies  
github.com/joho/godotenv  
github.com/mattn/go-sqlite3  
