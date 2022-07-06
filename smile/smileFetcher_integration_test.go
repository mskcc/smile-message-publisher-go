//go:build integration

// can run test from app root as following:
// go test -tags integration ./smile -smilerequesturl=http://plvicmometadb1.mskcc.org:3000/request/
package smile

import (
	"encoding/json"
	"flag"
	"github.com/google/go-cmp/cmp"
	igo "github.com/mskcc/smile-commons/types/igo/v1"
	"github.com/mskcc/smile-message-publisher-go/types"
	"google.golang.org/protobuf/testing/protocmp"
	"io/ioutil"
	"os"
	"testing"
)

var args types.Arguments

func initArgs() {
	flag.StringVar(&args.SmileRequestUrl, "smilerequesturl", "", "")
	flag.Parse()
}

func TestMain(m *testing.M) {
	initArgs()
	exitVal := m.Run()
	os.Exit(exitVal)
}

func openExpected(t *testing.T) (igo.RequestWithManifests, error) {
	rwm := igo.RequestWithManifests{}
	jsonBytes, err := ioutil.ReadFile("testData/05274_C.json")
	if err != nil {
		return rwm, err
	}
	err = json.Unmarshal(jsonBytes, &rwm)
	return rwm, err
}

func TestSmileFetcher_fetchRequestIntegration(t *testing.T) {
	rwm, err := fetchRequest("05274_C", args)
	if err != nil {
		t.Error("Unexpected error: ", err)
	}

	expected, err := openExpected(t)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, rwm, protocmp.Transform()); diff != "" {
		t.Error(diff)
	}
}
