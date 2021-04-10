package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var youLessAddress = ""
var serverAddress = ""
var client = http.Client{
	Timeout: 10 * time.Second,
}

type YouLessMetricsDto struct {
	Power int    `json:"pwr"`
	Meter string `json:"cnt"`
}

type YouLessMetrics struct {
	Power int
	Meter int
}

func getMetrics() (result YouLessMetrics, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				err = r.(error)
			case string:
				err = errors.New(r.(string))
			default:
				err = errors.New(fmt.Sprintf("%+v", r))
			}
		}
	}()
	response, err := client.Get(youLessAddress)
	if err != nil {
		return
	}
	defer response.Body.Close()
	dto := YouLessMetricsDto{}
	err = json.NewDecoder(response.Body).Decode(&dto)
	if err != nil {
		return
	}
	result.Power = dto.Power
	meter, err := strconv.Atoi(strings.Replace(strings.TrimSpace(dto.Meter), ",", "", 1))
	if err != nil {
		return
	}
	result.Meter = meter
	return
}

var servePrometheus = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	metrics, err := getMetrics()
	if err != nil {
		log.Printf("error getting metrics: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintln(writer, "# HELP meter_reading The current reading of the power meter in kWh")
	fmt.Fprintln(writer, "# TYPE meter_reading gauge")
	fmt.Fprintf(writer, "meter_reading %d\n\n", metrics.Meter)

	fmt.Fprintln(writer, "# HELP current_power The current power consumption in W")
	fmt.Fprintln(writer, "# TYPE current_power gauge")
	fmt.Fprintf(writer, "current_power %d\n\n", metrics.Power)
})

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s http://<youLessAddress>\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "\nOptions:")
		flag.CommandLine.PrintDefaults()
	}
	flag.StringVar(&serverAddress, "listen", ":80", "The address on which the metrics are served.")
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	youlessUrl, err := url.Parse(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(flag.CommandLine.Output(), "Invalid YouLess url: %s", err)
		flag.Usage()
		os.Exit(2)
	}
	youlessUrl.Path = "/a"
	youlessUrl.RawQuery = "f=j"
	youLessAddress = youlessUrl.String()

	// test it once
	metrics, err := getMetrics()
	if err != nil {
		log.Printf("failed to connect to %s: %s", youLessAddress, err)
		os.Exit(3)
	} else {
		log.Printf("connected to youless\n\tcurrent count: %d kWh\n\tcurrent power: %d W", metrics.Meter, metrics.Power)
	}

	log.Printf("starting http server on %s", serverAddress)

	if err := http.ListenAndServe(serverAddress, servePrometheus); err != http.ErrServerClosed {
		log.Fatalf("http server crashed: %s", err)
	}

}
