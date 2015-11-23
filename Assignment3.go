package main
import (
"fmt"
"encoding/json"
  "github.com/julienschmidt/httprouter"
  "gopkg.in/mgo.v2"
  "net/http"
  "os"
  "strconv"
  "bytes"
  )
type UberPriceResponse struct {
   Prices []struct {
       CurrencyCode         string  `json:"currency_code"`
       DisplayName          string  `json:"display_name"`
       Distance             float64 `json:"distance"`
       Duration             int     `json:"duration"`
       Estimate             string  `json:"estimate"`
       HighEstimate         float64 `json:"high_estimate"`
       LocalizedDisplayName string  `json:"localized_display_name"`
       LowEstimate          int     `json:"low_estimate"`
       Minimum              int     `json:"minimum"`
       ProductID            string  `json:"product_id"`
       SurgeMultiplier      int     `json:"surge_multiplier"`
   } `json:"prices"`
}
type UberRideResponse struct {
  Driver          interface{} `json:"driver"`
  Eta             int         `json:"eta"`
  Location        interface{} `json:"location"`
  RequestID       string      `json:"request_id"`
  Status          string      `json:"status"`
  SurgeMultiplier interface{} `json:"surge_multiplier"`
  Vehicle         interface{} `json:"vehicle"`
}

type MongoResponse struct {
  ID         int    `json:"id" bson:"_id"`
  Name       string `json:"name"`
  Address    string `json:"address"`
  City       string `json:"city"`
  State      string `json:"state"`
  Zip        string `json:"zip"`
  Coordinate struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
  } `json:"coordinate"`
}
type TripRequest struct{
    LocationIds            []string `json:"location_ids"`
    StartingFromLocationID string   `json:"starting_from_location_id"`
  }
type TripResponse struct{
    BestRouteLocationIds   []int   `json:"best_route_location_ids"`
    ID                     int     `json:"id" bson:"_id"`
    StartingFromLocationID int     `json:"starting_from_location_id"`
    Status                 string  `json:"status"`
    TotalDistance          float64 `json:"total_distance"`
    TotalUberCosts         float64 `json:"total_uber_costs"`
    TotalUberDuration      int     `json:"total_uber_duration"`
  }
type TripPutResponse struct{
ID                        int   `json:"id"`
Status                    string   `json:"status"`// using boolean
StartingFromLocationID    int   `json:"starting_from_location_id"`
NextDestinationLocationID int   `json:"next_destination_location_id"`
BestRouteLocationIds      []int `json:"best_route_location_ids"`
TotalUberCosts            float64      `json:"total_uber_costs"`
TotalUberDuration         int      `json:"total_uber_duration"`
TotalDistance             float64  `json:"total_distance"`
UberWaitTimeEta           int      `json:"uber_wait_time_eta"`//All logic 

}
type UberRideRequestRequest struct {
ProductID      string  `json:"product_id"`
StartLatitude  float64 `json:"start_latitude"`
StartLongitude float64 `json:"start_longitude"`
EndLatitude    float64 `json:"end_latitude"`
EndLongitude   float64 `json:"end_longitude"`
}

var count int = 1121;
var countUber int = 0;
func createTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params ){
  tripReq:= TripRequest{}
  json.NewDecoder(req.Body).Decode(&tripReq)
  startLoc,_ := strconv.Atoi(tripReq.StartingFromLocationID)
  locIds :=stringConvert(tripReq.LocationIds)
  tripRes:= TripResponse{}
  tripRes.ID = getId(); ///---------------
  tripRes.Status = "planning"  //Doubtful
  tripRes.StartingFromLocationID = startLoc//Error
  fmt.Println("StartLoc set", startLoc)
  tripRes.BestRouteLocationIds, tripRes.TotalUberCosts, tripRes.TotalUberDuration, tripRes.TotalDistance  = getBestRoute( startLoc, locIds)//The call goes to get BestRoute
  
  tripResJson, _ := json.Marshal(tripRes)

  addTrip(tripRes)

  rw.Header().Set("Content-Type","application/json")
  rw.WriteHeader(201)
  fmt.Fprintf(rw, "%s", tripResJson)

}
func stringConvert(ab []string) (b []int){
  for _,i:= range ab{
    x,_:=strconv.Atoi(i)
    b= append(b, x)
  }
  return b
}

func addTrip(tripRes TripResponse){

  session, err1 := mgo.Dial(getTripDatabaseURL())

  
  if err1 != nil {
    fmt.Println("Error while connecting to database while creating")
    os.Exit(1)
  } else {
    fmt.Println("Session Created")
  }

  
  session.DB("trips").C("tripCollection").Insert(tripRes)

  
  session.Close()

}

func putTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params){
  id := p.ByName("tripId")
  tripId, _ := strconv.Atoi(id)
  getRes:= TripResponse{}
  getRes = getTripDb(tripId)
  putRes:= TripPutResponse{}
  putRes.ID = getRes.ID
  putRes.StartingFromLocationID = getRes.StartingFromLocationID
  putRes.BestRouteLocationIds = getRes.BestRouteLocationIds
  putRes.TotalDistance = getRes.TotalDistance
  putRes.TotalUberDuration = getRes.TotalUberDuration
  putRes.TotalUberCosts = getRes.TotalUberCosts
  cur:= putRes.StartingFromLocationID
  next:=0
  product_id:=""
  bestRoute := getRes.BestRouteLocationIds
  x:=len(bestRoute)

  
  
  if countUber>= x+1{
    putRes.Status="finished"
    putRes.NextDestinationLocationID = putRes.StartingFromLocationID


  }else if countUber==x{
    putRes.NextDestinationLocationID = putRes.StartingFromLocationID
    putRes.Status="requesting"
    next = putRes.NextDestinationLocationID
    cur = bestRoute[x-1]
    countUber++



  }else{
    putRes.Status="requesting"
    if countUber!=0{
      cur = bestRoute[countUber-1]
    }

  putRes.NextDestinationLocationID =  bestRoute[countUber]
  next = putRes.NextDestinationLocationID
  countUber++

}
if next!=0{
  result:= UberPriceResponse{}
  url:= getUrl(cur, next)
  response, err:= http.Get(url) 
  if err != nil {
    fmt.Println("Error while computing the route", err.Error())
    os.Exit(1)
  
  }

json.NewDecoder(response.Body).Decode(&result)
product_id = result.Prices[0].ProductID
fmt.Println(product_id)
 } 

  putRes.UberWaitTimeEta =getWait(cur,next, product_id)


  resJson, _ := json.Marshal(putRes)
  rw.Header().Set("Content-Type", "application/json")
  fmt.Fprintf(rw, "%s", resJson)

}

func getWait(curLoc int, nextLoc int, productId string) (int){
curLocLat, curLocLng:=getLatLng(curLoc)
nextLocLat, nextLocLng :=getLatLng(nextLoc)
url:= "https://sandbox-api.uber.com/v1/requests"
uberAuthorizationToken:="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiJhYzdiZWJmMS02YzhlLTQyOTgtOGJlNy1iYWE3YzdhODRhNjciLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjE2YjMzOWI3LWVlYWYtNDU0YS1iNTVhLWI5YjI1MWQ1YmU3NCIsImV4cCI6MTQ1MDc0MjcyNywiaWF0IjoxNDQ4MTUwNzI2LCJ1YWN0IjoidnlKSU1pY1VIQ2wwWVdvUkFBbVBZWmNYV0s4b2JUIiwibmJmIjoxNDQ4MTUwNjM2LCJhdWQiOiI2R282ZUxhZC1Kby1hQkI0WDhuWUJtaTJ5ckcwdFJ6TSJ9.Wr0zJQKBvxo-sZSx-L24PjSJfCNG0upAWdYmSqx7hw9e3oqK_sCu11CzzAd0jhZ_8mM1jE5bZ9fuo52f4Mi9D_lVQpUbGEVmaTTnE8jG741PCmq-MEiajFUJrQDn-EFFTJRvyU0Fv6m3dgylRVmoCmP3EMhIHINotMKlRAksjg-JWpIGLErM9ess2r9GsQVdL0uE3-jnOXcDC8PKZMAwugUQ6P6Jk2tme0kirQuIkE6mlZnnJVSBldiG2k4zSScmtPDUrmU1LOJA1Lnh1VIFZJtYj5Ke7BB54KYjKhRTHiEcqBhG22QdXymTkrzvSFxccS8NTH_QbevEB1uH9rbr-w"
urrreq:= UberRideRequestRequest{}
urrreq.ProductID= productId
urrreq.StartLatitude=curLocLat
urrreq.EndLatitude=nextLocLat
urrreq.StartLongitude=curLocLng
urrreq.EndLongitude=nextLocLng
jsonInputData, _ := json.Marshal(urrreq)
fmt.Println("Req sent",urrreq)
result:=UberRideResponse{}

response, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonInputData))
response.Header.Set("Content-Type", "application/json")
response.Header.Set("Authorization", "Bearer "+uberAuthorizationToken)

  client := &http.Client{}
  resp, err := client.Do(response)

  if err != nil {
    fmt.Println("Error while getting response from uber ride request api", err.Error())
    panic(err)
  }

  defer resp.Body.Close()
json.NewDecoder(resp.Body).Decode(&result)
fmt.Println(result)

waitTime:= result.Eta
return waitTime
  }
func getTrip(rw http.ResponseWriter, req *http.Request, p httprouter.Params){
  id := p.ByName("tripId")
  tripId, _ := strconv.Atoi(id)
  getRes:= TripResponse{}
  getRes = getTripDb(tripId)

  resJson, _ := json.Marshal(getRes)
  rw.Header().Set("Content-Type", "application/json")
  fmt.Fprintf(rw, "%s", resJson)
}

func getTripDb(tripId int) TripResponse{

  getTripRes:= TripResponse{}

session, err1 := mgo.Dial(getTripDatabaseURL())

  
  if err1 != nil {
    fmt.Println("Error while connecting to database while getting")
    os.Exit(1)
  } else {
    fmt.Println("Session Created")
  }

  
  err2 := session.DB("trips").C("tripCollection").FindId(tripId).One(&getTripRes)
  if err2 != nil {
    panic(err2)
  } else {
    fmt.Println("Trip Data retrieved from trip DB")
  }
  session.Close()
return getTripRes
}

func getBestRoute(startLoc int, locIds []int) ([]int, float64, int, float64){//Returns and array of best route
  fmt.Println("Control moves to getBestRoute, startLoc is", startLoc)
  allComb:=permutations(locIds)                  //Call to all permutation combinations returns 2d array
  var bestRoute []int
  var finalCost float64
  var finalDis float64
  var finalDur int

  for j,i:= range allComb {
    fmt.Println("j=",j)
      fmt.Println("Print all Permu",i)          //iterated over all possible combinations
    cost, dur, dist := getUberCostTimeDis(startLoc, i)   //call goes to this function for every combo//Giving 0
    fmt.Println("Current Cost", cost)
    if j ==0 {
      finalCost = cost
          }
     
      if cost<finalCost {
      finalCost = cost
      finalDur = dur
      finalDis = dist
      bestRoute = i
      fmt.Println("Final Cost for now is", finalCost)
    }
  }
return bestRoute , finalCost, finalDur, finalDis
}


func permutations(arr []int) [][]int {
   var helper func([]int, int)
   res := [][]int{}

   helper = func(arr []int, n int) {
       if n == 1 {
           tmp := make([]int, len(arr))
           copy(tmp, arr)
           res = append(res, tmp)
       } else {
           for i := 0; i < n; i++ {
               helper(arr, n-1)
               if n%2 == 1 {
                   tmp := arr[i]
                   arr[i] = arr[n-1]
                   arr[n-1] = tmp
               } else {
                   tmp := arr[0]
                   arr[0] = arr[n-1]
                   arr[n-1] = tmp
               }
           }
       }
   }
   helper(arr, len(arr))
   return res
}
   
  
func getUberCostTimeDis( startLoc int, locIds []int ) (float64,  int , float64) {
//append the startLoc behind the locIds slice
fmt.Println("Control moves to getUberCostTimeDis startLoc is", startLoc)
n:= len(locIds)
allLocs := make([]int, n)
copy(allLocs, locIds[:n])
allLocs= append(allLocs, startLoc)
fmt.Println("All elements of allLocs",allLocs)
fmt.Println("Value of startLoc being inserted",allLocs[n])
curLoc:= startLoc
var cost float64
var dist float64
var time int
for _,nextLoc := range allLocs{
  result:= UberPriceResponse{}
  fmt.Println("nextLoc", nextLoc, "CurrLoc", curLoc)
  url:= getUrl(curLoc, nextLoc)
  response, err:= http.Get(url) 
  if err != nil {
    fmt.Println("Error while computing the route", err.Error())
    os.Exit(1)
  
  }

json.NewDecoder(response.Body).Decode(&result)
fmt.Println(result)

cost=cost+result.Prices[0].HighEstimate
fmt.Println("Cost being set", cost)
time = time+result.Prices[0].Duration
dist = dist + result.Prices[0].Distance



fmt.Println("Cost and Stuff Stored")

curLoc=nextLoc
}
return cost, time, dist
}
func getUrl(aLoc int, bLoc int ) string{
  fmt.Println("Control moves to getUrl")

latA, lngA := getLatLng( aLoc)
latB, lngB := getLatLng(bLoc)
latAString := strconv.FormatFloat(latA, 'f', -1, 64)
lngAString:=strconv.FormatFloat(lngA, 'f', -1, 64)
latBString:=strconv.FormatFloat(latB, 'f', -1, 64)
lngBString:=strconv.FormatFloat(lngB, 'f', -1, 64)
url:= "https://api.uber.com/v1/estimates/price?start_latitude="+latAString+"&start_longitude="+lngAString+"&end_latitude="+latBString+"&end_longitude="+lngBString+"&server_token=6aSSJPLoTBCmMImqNzyl_6DmPIz6KziFSQYgbBkU"
  
return url
}
func getLatLng(id int) (lat float64, lng float64){
  fmt.Println("Control moves to getLatLng")
  
  mgo_res:= MongoResponse{}

  session, err1:= mgo.Dial(getDatabaseURL())
  if err1 != nil {
    fmt.Println("Error while connecting to database")
    os.Exit(1)
  } else {
    fmt.Println("Session Created")
  }
err2 := session.DB("locationdb").C("locationCollection").FindId(id).One(&mgo_res)

  if err2 != nil {
    panic(err2)
  } else {
    fmt.Println("Location retrieved from Location DB")
  }
  lat = mgo_res.Coordinate.Lat
  lng = mgo_res.Coordinate.Lng
  session.Close()

return lat, lng
}
func getId() int{
  count++
  return count
 }


func getDatabaseURL() string {
  fmt.Println("Control moves to getDburl")
  var dburl string = "mongodb://Pulkit:12345@ds045464.mongolab.com:45464/locationdb"
  return dburl
}

func getTripDatabaseURL() string {
  fmt.Println("Control moves to gettripdburl")
  var dburl string = "mongodb://Pulkit:12345@ds055574.mongolab.com:55574/trips"
  return dburl
}
func main() {
  mux := httprouter.New()
  mux.GET("/trips/:tripId", getTrip)
  mux.POST("/trips", createTrip)
  mux.PUT("/trips/:tripId/request", putTrip)
 
  server := http.Server{
    Addr:    "0.0.0.0:8080",
    Handler: mux,
  }
  server.ListenAndServe()
}

