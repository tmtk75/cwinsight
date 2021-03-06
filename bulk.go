package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var BulkCmd = cobra.Command{
	Use:   "bulk [file]",
	Short: "bulky query",
	Args:  cobra.ExactArgs(1),
	Run: func(c *cobra.Command, args []string) {
		f, err := os.Open(args[0])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		Bulk(viper.GetString(keyQueryString), either(f)(os.Stdin))
	},
}

type Result struct {
	QueryResult *QueryResult
	GroupName   string
}

func Bulk(qs string, r io.Reader) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	l := strings.Split(strings.Trim(string(b), " \t\n"), "\n")
	start, end := startEndTime()

	checkDurationQuota(end.Sub(start) * time.Duration(len(l)))

	res := make(chan *Result)
	var wg sync.WaitGroup
	f := func(lg string) {
		res <- &Result{QueryResult: Query(qs, lg, start, end), GroupName: lg}
		wg.Done()
	}

	for _, e := range l {
		wg.Add(1)
		go f(e)
	}

	type RT struct {
		Response  interface{}
		GroupName string
	}
	a := make([]*RT, 0)
	go func() {
		for e := range res {
			//fmt.Printf("%v\n", e)
			a = append(a, &RT{
				GroupName: e.GroupName,
				Response:  format(e.QueryResult),
			})
		}
	}()
	wg.Wait()
	close(res)

	type Output struct {
		QueryString string
		StartTime   time.Time
		EndTime     time.Time
		ResultSet   []*RT
	}
	bb, err := json.Marshal(Output{ResultSet: a, StartTime: start, EndTime: end, QueryString: qs})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v", string(bb))
}
