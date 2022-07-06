package lims

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/copier"
	igo "github.com/mskcc/smile-commons/types/igo/v1"
	"github.com/mskcc/smile-message-publisher-go/types"
	sm "github.com/mskcc/smile-messaging-go/mom/nats"
	"google.golang.org/protobuf/proto"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func getMessaging(args types.Arguments) (*sm.Messaging, error) {
	return sm.NewMessaging(args.NatsUrl, sm.WithTLS(args.NatsTrustPath, args.NatsKeyPath, args.NatsConName, args.NatsConPw))
}

func getLimsHttpReq(url string, user, pw string) (*http.Request, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, cancel, err
	}
	req.Header.Set("Accept", "application/json")
	auth := user + ":" + pw
	b64Auth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+b64Auth)
	return req, cancel, nil
}

func FetchRequestsByDate(args types.Arguments) error {
	reqIds, err := fetchDeliveriesByDate(args)
	if err != nil {
		return err
	}
	return FetchRequests(reqIds, args)
}

func fetchDeliveriesByDate(args types.Arguments) ([]string, error) {
	getDelURL := fmt.Sprintf("https://%s/LimsRest/api/getDeliveries?timestamp=%d", args.LimsHost, args.StartDate.UnixMilli())
	req, cancel, err := getLimsHttpReq(getDelURL, args.LimsUser, args.LimsPW)
	if err != nil {
		return nil, err
	}
	defer cancel()

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP return code != 200: %d\n", resp.StatusCode)
	}

	var dels []igo.Delivery
	if err := json.Unmarshal(body, &dels); err != nil {
		return nil, err
	}
	var reqIds []string
	for _, del := range dels {
		delDt := time.UnixMilli(del.DeliveryDate)
		if !delDt.After(args.EndDate) {
			reqIds = append(reqIds, del.Request)
		}
	}
	return reqIds, nil
}

func FetchRequests(reqIds []string, args types.Arguments) error {
	m, err := getMessaging(args)
	if err != nil {
		return err
	}
	lc := 0
	for _, id := range reqIds {
		lc++
		log.Printf("Attempting to fetch and publish %d of %d request(s): %s\n", lc, len(reqIds), id)
		req, err := fetchRequest(id, args)
		if err != nil {
			log.Printf("Failure to fetch request %s: %s\n", id, err)
			continue
		}
		// skip a non-cmo request if CMOReqs are desired
		if args.CMOReqs && !req.IsCmoRequest {
			log.Printf("Skipping non-cmo request %s as 'cmo_requests_only (-c)' flag is set\n", id)
			continue
		}
		sMans := fetchSampleManifests(req, args)
		rwm := combineRequestAndSamples(req, sMans)
		out, err := proto.Marshal(&rwm)
		if err != nil {
			log.Printf("Failure to serialize request w/manifests %s: %s\n", id, err)
			continue
		}
		if err = m.Publish(args.LimsPubTop, out); err != nil {
			log.Printf("Failure to publish request w/manifests %s: %s\n", id, err)
			continue
		}
		log.Printf("Successfully fetched and published request %s\n", id)
	}
	m.Shutdown()
	return nil
}

func protoMarshal(jsonContent []byte) ([]byte, error) {
	rwm := igo.RequestWithManifests{}
	if err := json.Unmarshal([]byte(jsonContent), &rwm); err != nil {
		return nil, err
	}
	return proto.Marshal(&rwm)
}

func FetchRequestFromJSONFile(args types.Arguments) error {
	log.Printf("Attempting to fetch & publish request from JSON file\n")
	file, err := ioutil.ReadFile(args.JSONFilePath)
	if err != nil {
		return err
	}
	out, err := protoMarshal(file)
	if err != nil {
		return err
	}
	m, err := getMessaging(args)
	if err != nil {
		return err
	}
	if err = m.Publish(args.LimsPubTop, out); err != nil {
		return err
	}
	log.Printf("Successfully fetched and published request from JSON file\n")
	return nil
}

func FetchRequestFromPublisherFile(args types.Arguments) error {
	inFile, err := os.Open(args.PublisherFilePath)
	if err != nil {
		return err
	}
	defer inFile.Close()
	rd := bufio.NewReader(inFile)
	m, err := getMessaging(args)
	if err != nil {
		return err
	}
	lc := 0
	for {
		lc++
		log.Printf("Attempting to process row %d from publisher file\n", lc)
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			log.Printf("Error parsing row %d from publisher file, expecting 3 components, found %d\n", lc, len(parts))
			continue
		}
		out, err := protoMarshal([]byte(parts[2]))
		if err != nil {
			log.Printf("Failure to serialize row %d from publisher file: %s\n", lc, err)
			continue
		}
		if err = m.Publish(parts[1], out); err != nil {
			log.Printf("Failure to publish row %d from publisher file: %s\n", lc, err)
			continue
		}
		log.Printf("Successfully processed row %d of publisher file\n", lc)
	}
	return err
}

func fetchRequest(reqId string, args types.Arguments) (igo.Request, error) {
	var iReq igo.Request
	getReqURL := fmt.Sprintf("https://%s/LimsRest/api/getRequestSamples?request=%s", args.LimsHost, reqId)
	req, cancel, err := getLimsHttpReq(getReqURL, args.LimsUser, args.LimsPW)
	if err != nil {
		return iReq, err
	}
	defer cancel()

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return iReq, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return iReq, err
	}
	if resp.StatusCode != http.StatusOK {
		return iReq, fmt.Errorf("HTTP return code != 200: %d\n", resp.StatusCode)
	}
	err = json.Unmarshal(body, &iReq)
	return iReq, err
}

func fetchSampleManifests(iReq igo.Request, args types.Arguments) []igo.SampleManifest {
	var manifests []igo.SampleManifest
	ns := len(iReq.GetSamples())
	lc := 0
	for _, s := range iReq.GetSamples() {
		lc++
		log.Printf("Attempting to fetch %d of %d sample manifest(s): %s\n", lc, ns, s.IgoSampleId)
		man, err := fetchSampleManifest(s.IgoSampleId, args)
		if err != nil {
			log.Printf("Failure to fetch sample manifest %s: %s\n", s.IgoSampleId, err)
			continue
		}
		man.IgoComplete = s.IgoComplete
		manifests = append(manifests, man)
		log.Printf("Successfully fetched sample manifest %s\n", s.IgoSampleId)
	}
	return manifests
}

func fetchSampleManifest(sId string, args types.Arguments) (igo.SampleManifest, error) {
	var manifest igo.SampleManifest
	getManURL := fmt.Sprintf("https://%s/LimsRest/api/getSampleManifest?igoSampleId=%s", args.LimsHost, sId)
	req, cancel, err := getLimsHttpReq(getManURL, args.LimsUser, args.LimsPW)
	if err != nil {
		return manifest, err
	}
	defer cancel()
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return manifest, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return manifest, err
	}
	if resp.StatusCode != http.StatusOK {
		return manifest, fmt.Errorf("HTTP return code != 200: %d\n", resp.StatusCode)
	}
	var mans []igo.SampleManifest
	err = json.Unmarshal(body, &mans)
	return mans[0], err
}

func combineRequestAndSamples(iReq igo.Request, sMans []igo.SampleManifest) igo.RequestWithManifests {
	rwm := igo.RequestWithManifests{}
	copier.Copy(&rwm, &iReq)
	var combMans []*igo.SampleManifest
	for i, _ := range sMans {
		combMans = append(combMans, &sMans[i])
	}
	rwm.Samples = combMans
	rwm.ProjectId = strings.Split(iReq.RequestId, "_")[0]
	return rwm
}
