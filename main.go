package main

import (
  "encoding/json"
  "fmt"
  "github.com/nuttapp/pinghist/ping"
  "io/ioutil"
  "net/http"
  "sort"
  "sync"
  "time"
)

const URL = "https://api.mullvad.net/www/relays/all/"

type Server struct {
  List         []float64
  Last         float64
  URL          string
  Hostname     string
  Country_code string
  Country_name string
  City_name    string
  Active       bool
  Type         string
}

var servers []*Server
var activeOpenVpnServers []*Server

type ByLast []*Server

func (a ByLast) Len() int           { return len(a) }
func (a ByLast) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLast) Less(i, j int) bool { return a[i].Last < a[j].Last }

func (s *Server) Ping() (float64, error) {
  //for i := 0; i < 3; i++ {
  fmt.Println("Pinging server", s.URL)
  r, err := ping.Ping(s.URL)
  if err != nil {
    return 0, err
  }

  if r.Time < 1 {
    r.Time = 999999999999
  }

  s.List = append(s.List, r.Time)
  s.Last = r.Time
  //}

  return s.Last, nil
}

func main() {
  fmt.Println("PING EM ALL AND LET THE SORT SORT EM OUT")

  response, err := http.Get(URL)
  if err != nil {
    fmt.Println("Request Error", err.Error())
  }
  defer response.Body.Close()
  body, err := ioutil.ReadAll(response.Body)

  jsonErr := json.Unmarshal(body, &servers)
  fmt.Printf("There are %d servers\n\n", len(servers))
  if jsonErr != nil {
    fmt.Println(jsonErr)
  } else {
    for i := range servers {
      servers[i].URL = servers[i].Hostname + ".relays.mullvad.net"
      if servers[i].Type == "openvpn" && servers[i].Active {
        activeOpenVpnServers = append(activeOpenVpnServers, servers[i])
      }
    }
  }
  fmt.Printf("There are %d active openVPN servers\n\n", len(activeOpenVpnServers))

  // Now we ping them!

  var wg sync.WaitGroup
  wg.Add(len(activeOpenVpnServers) - 1)

  for _, server := range activeOpenVpnServers {
    time.Sleep(100 * time.Millisecond)

    go func(wg *sync.WaitGroup, server *Server) {
      for i := 0; i < 3; i++ {
        _, err := server.Ping()
        if err != nil {
          fmt.Println("Error on ping", server.URL, err.Error())
          continue
        }

        break
      }

      wg.Done()
    }(&wg, server)
  }

  wg.Wait()

  fmt.Println("------------------------------------------")
  fmt.Println("------------------------------------------")

  sort.Sort(ByLast(activeOpenVpnServers))
  printed := 0
  for _, server := range activeOpenVpnServers {
    if server.Last < 1 {
      continue
    }

    fmt.Println(server.URL, "\t", server.Last, "ms")
    fmt.Println("------------------------------------------")

    if printed >= 9 {
      break
    }

    printed++
  }
}
