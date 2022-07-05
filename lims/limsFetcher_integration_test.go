//go:build integration

// can run test from app root as following:
// go test -tags integration ./lims  -limshost=igolims.mskcc.org:8443 -limsuser=XXX -limspw=XXX
package lims

import (
	"encoding/json"
	"flag"
	"github.com/google/go-cmp/cmp"
	igo "github.com/mskcc/smile-commons/types/igo/v1"
	"github.com/mskcc/smile_message_publisher_go/types"
	"google.golang.org/protobuf/testing/protocmp"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var args types.Arguments

func initArgs() error {
	flag.StringVar(&args.LimsHost, "limshost", "", "")
	flag.StringVar(&args.LimsUser, "limsuser", "", "")
	flag.StringVar(&args.LimsPW, "limspw", "", "")
	flag.Parse()
	var err error
	args.StartDate, err = time.Parse("01/02/2006", "06/29/2022")
	if err != nil {
		return err
	}
	args.EndDate, err = time.Parse("01/02/2006", "06/30/2022")
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	err := initArgs()
	if err != nil {
		os.Exit(1)
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestLimsFetcher_fetchDeliveriesByData(t *testing.T) {
	reqIds, err := fetchDeliveriesByDate(args)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}
	if len(reqIds) != 2 {
		t.Error("incorrect result: expected 2, got", len(reqIds))
	}
}

func openExpectedRequest(t *testing.T) (igo.Request, error) {
	req := igo.Request{}
	jsonBytes, err := ioutil.ReadFile("testData/13370.json")
	if err != nil {
		return req, err
	}
	err = json.Unmarshal(jsonBytes, &req)
	return req, err
}

func TestLimsFetcher_fetchRequestIntegration(t *testing.T) {
	req, err := fetchRequest("13370", args)
	if err != nil {
		t.Error("Unexpected  error: ", err)
	}
	expected, err := openExpectedRequest(t)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, req, protocmp.Transform()); diff != "" {
		t.Error(diff)
	}
}

func TestLimsFetcher_fetchSampleManifestsIntegration(t *testing.T) {
	req, err := openExpectedRequest(t)
	if err != nil {
		t.Fatal(err)
	}
	sms := fetchSampleManifests(req, args)
	if len(sms) != 4 {
		t.Error("incorrect result: expected 4, got", len(sms))
	}
	sIds := map[string]string{
		"13370_1": "",
		"13370_2": "",
		"13370_3": "",
		"13370_4": "",
	}
	for _, s := range sms {
		_, ok := sIds[s.IgoId]
		if !ok {
			t.Error("Unexpected sample id: ", s.IgoId)
		}
	}
}

func openExpectedSampleManifest(t *testing.T) (igo.SampleManifest, error) {
	man := igo.SampleManifest{}
	jsonBytes, err := ioutil.ReadFile("testData/13370_1.json")
	if err != nil {
		return man, err
	}
	err = json.Unmarshal(jsonBytes, &man)
	return man, err
}

func TestLimsFetcher_fetchSampleManifestIntegration(t *testing.T) {
	man, err := fetchSampleManifest("13370_1", args)
	if err != nil {
		t.Error("Unexpected  error: ", err)
	}
	expected, err := openExpectedSampleManifest(t)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, man, protocmp.Transform()); diff != "" {
		t.Error(diff)
	}
}
