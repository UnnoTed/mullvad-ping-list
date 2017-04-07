package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/franela/goreq"
	"github.com/nuttapp/pinghist/ping"
	"github.com/yhat/scrape"
)

const URL = "https://www.mullvad.net/guides/our-vpn-servers/"

var servers []*Server

type Server struct {
	List []float64
	Last float64
	URL  string
}

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
	req := goreq.Request{
		Method: "GET",
		Uri:    URL,
	}

	var (
		res *goreq.Response
		err error
	)

	for {
		res, err = req.Do()
		if err != nil {
			fmt.Println("Request Error", err.Error())
			continue
		}

		break
	}

	root, err := html.Parse(res.Body)
	if err != nil {
		panic(err)
	}

	pres := scrape.FindAll(root, func(n *html.Node) bool {
		return n.DataAtom == atom.Pre
	})

	for _, p := range pres {
		txt := scrape.Text(p)
		lines := strings.Split(txt, "\n")

		for _, line := range lines {
			svr := strings.Split(line, "|")
			url := svr[len(svr)-1]

			if strings.Contains(url, ".mullvad.net") {
				servers = append(servers, &Server{
					URL: strings.TrimSpace(url),
				})
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(servers) - 1)

	for _, server := range servers {
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

	sort.Sort(ByLast(servers))
	printed := 0
	for _, server := range servers {
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
