package smile

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mskcc/smile_message_publisher_go/types"
	sm "github.com/mskcc/smile_messaging_go/mom/nats"
	igo "github.com/mskcc/smile_types/igo/v1"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func FetchRequests(args types.Arguments) error {
	m, err := sm.NewMessaging(args.NatsUrl, sm.WithTLS(args.NatsTrustPath, args.NatsKeyPath, args.NatsConName, args.NatsConPw))
	if err != nil {
		return nil
	}
	lc := 0
	for _, id := range args.ReqIds {
		lc++
		log.Printf("Attempting to fetch and publish %d of %d request(s): %s\n", lc, len(args.ReqIds), id)
		req, err := fetchRequest(id, args)
		if err != nil {
			log.Printf("Failure to fetch request %s: %s\n", id, err)
			continue
		}
		if args.CMOReqs && !req.IsCmoRequest {
			log.Printf("Skipping non-cmo request %s as 'cmo_requests_only (-c)' flag is set\n", id)
			continue
		}
		out, err := proto.Marshal(&req)
		if err != nil {
			log.Printf("Failure to serialize request %s: %s\n", id, err)
			continue
		}
		if err = m.Publish(args.SmilePubTop, out); err != nil {
			log.Printf("Failure to publish request %s: %s\n", id, err)
			continue
		}
		log.Printf("Successfully fetched and published request %s\n", id)
	}
	m.Shutdown()
	return nil
}

func fetchRequest(reqId string, args types.Arguments) (igo.RequestWithManifests, error) {
	rwm := igo.RequestWithManifests{}
	reqUrl := args.SmileRequestUrl + reqId
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return rwm, err
	}
	req.Header.Set("Accept", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return rwm, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rwm, err
	}
	if resp.StatusCode != http.StatusOK {
		return rwm, fmt.Errorf("HTTP return code != 200: %d\n", resp.StatusCode)
	}
	err = json.Unmarshal(body, &rwm)
	return rwm, err
}
