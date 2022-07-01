package main

import (
	"fmt"
	"github.com/mskcc/smile_message_publisher_go/lims"
	"github.com/mskcc/smile_message_publisher_go/smile"
	"github.com/mskcc/smile_message_publisher_go/types"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func setupOptions() {
	pflag.BoolP("help", "h", false, "Describes available options")
	pflag.StringP("cfg_file", "f", "", "Path to configuration file containing Lims, Nats settings & creds")
	pflag.StringP("request_ids", "r", "", "Comma-separated list of request ids to fetch from LimSRest")
	pflag.StringP("start_date", "s", "", "Start date [MM/DD/YYYY].  Fetch requests from LimsRest between the given start and end dates [START/END DATE MODE]")
	pflag.StringP("end_date", "e", "", "End date [MM/DD/YYYY]. Fetch requests from LimsRest between the given start and end dates [START/END DATE MODE]")
	pflag.StringP("cmo_requests_only", "c", "", "Filter Lims requests by CMO requests flag")
	pflag.StringP("json_filename", "j", "", "Publishes contents of provided JSON file")
	pflag.StringP("publisher_filename", "p", "", "Publishes contents of provided JSON file")
	pflag.StringP("smile_service", "m", "", "Comma-separated list of request ids to fetch from Smile Web Service")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

func parseDates(args *types.Arguments) error {
	sds := viper.GetString("start_date")
	eds := viper.GetString("end_date")
	if sds == "" && eds == "" {
		return nil
	}
	if (sds != "" && eds == "") ||
		(sds == "" && eds != "") {
		return fmt.Errorf("Both start_date and end_date must be provided together")
	}

	sd, err := time.Parse("01/02/2006", sds)
	if err != nil {
		return err
	}
	ed, err := time.Parse("01/02/2006", eds)
	if err != nil {
		return err
	}
	if ed.Before(sd) {
		return fmt.Errorf("end_date comes before start_date")
	}
	args.StartDate = sd
	args.EndDate = ed
	args.DateMode = true

	return nil
}

func parseReqIds(args *types.Arguments) error {
	reqIds := viper.GetString("request_ids")
	if reqIds == "" {
		return nil
	}
	for _, id := range strings.Split(reqIds, ",") {
		args.ReqIds = append(args.ReqIds, id)
	}
	args.ReqIdMode = true
	return nil
}

func parseJSONFile(args *types.Arguments) error {
	jFile := viper.GetString("json_filename")
	if jFile == "" {
		return nil
	}
	args.JSONFileMode = true
	args.JSONFilePath = jFile
	return nil
}

func parsePublisherFile(args *types.Arguments) error {
	pFile := viper.GetString("publisher_filename")
	if pFile == "" {
		return nil
	}
	args.PublisherFileMode = true
	args.PublisherFilePath = pFile
	return nil
}

func parseSmileServiceMode(args *types.Arguments) error {
	sModeIds := viper.GetString("smile_service")
	if sModeIds == "" {
		return nil
	}
	for _, id := range strings.Split(sModeIds, ",") {
		args.ReqIds = append(args.ReqIds, id)
	}
	args.SmileServiceMode = true
	return nil
}

func readConfig(cfg types.Config, args *types.Arguments) error {
	viper.SetConfigName(cfg.Name)
	viper.SetConfigType(cfg.Type)
	viper.AddConfigPath(cfg.Path)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if args.LimsHost = viper.GetString("lims.host"); args.LimsHost == "" {
		return fmt.Errorf("Missing lims.host property in config file")
	}
	if args.LimsUser = viper.GetString("lims.username"); args.LimsUser == "" {
		return fmt.Errorf("Missing lims.username property in config file")
	}
	if args.LimsPW = viper.GetString("lims.password"); args.LimsPW == "" {
		return fmt.Errorf("Missing lims.password property in config file")
	}
	if args.LimsPubTop = viper.GetString("lims.publisher_topic"); args.LimsPubTop == "" {
		return fmt.Errorf("Missing lims.publisher_topic property in config file")
	}
	u, err := url.Parse(viper.GetString("nats.url"))
	if err != nil {
		return fmt.Errorf("Missing or malformed nats.url property in config file")
	} else {
		args.NatsUrl = u.String()
	}
	if args.NatsConName = viper.GetString("nats.consumer_name"); args.NatsConName == "" {
		return fmt.Errorf("Missing nats.consumer_name property in config file")
	}
	if args.NatsConPw = viper.GetString("nats.consumer_password"); args.NatsConPw == "" {
		return fmt.Errorf("Missing nats.consumer_password property in config file")
	}
	if args.NatsKeyPath = viper.GetString("nats.keystore_path"); args.NatsKeyPath == "" {
		return fmt.Errorf("Missing nats.keystore_path property in config file")
	}
	if args.NatsTrustPath = viper.GetString("nats.truststore_path"); args.NatsTrustPath == "" {
		return fmt.Errorf("Missing nats.truststore_path property in config file")
	}
	u, err = url.Parse(viper.GetString("smile.request_url"))
	if err != nil {
		return fmt.Errorf("Missing or malformed smile.request_url property in config file")
	} else {
		args.SmileRequestUrl = u.String()
	}
	if args.SmilePubTop = viper.GetString("smile.publisher_topic"); args.SmilePubTop == "" {
		return fmt.Errorf("Missing smile.publisher_topic property in config file")
	}
	return nil
}

func parseConfig() (types.Config, error) {
	toReturn := types.Config{}
	cf := viper.GetString("cfg_file")
	if cf == "" {
		return toReturn, fmt.Errorf("Missing cfg_file argument")
	}
	ext := filepath.Ext(cf)
	name := filepath.Base(cf)
	toReturn.Name = strings.TrimSuffix(name, ext)
	toReturn.Type = strings.TrimPrefix(ext, ".")
	toReturn.Path = filepath.Dir(cf)

	return toReturn, nil
}

func parseArgs() (types.Arguments, error) {
	toReturn := types.Arguments{}

	if viper.GetBool("help") {
		pflag.PrintDefaults()
		return toReturn, nil
	}

	config, err := parseConfig()
	if err != nil {
		return toReturn, err
	}
	if err = readConfig(config, &toReturn); err != nil {
		return toReturn, err
	}

	if err = parseDates(&toReturn); err != nil {
		return toReturn, err
	}
	parseReqIds(&toReturn)
	parseJSONFile(&toReturn)
	parsePublisherFile(&toReturn)
	parseSmileServiceMode(&toReturn)
	toReturn.CMOReqs = viper.GetBool("cmo_requests_only")
	return toReturn, nil
}

func run() error {
	setupOptions()
	args, err := parseArgs()
	if err != nil {
		log.Printf("Error parsing arguments: %s", err)
		return err
	}
	if args.ReqIdMode {
		if err = lims.FetchRequests(args.ReqIds, args); err != nil {
			log.Printf("Error fetching requests: %s\n", err)
		}
	} else if args.DateMode {
		if err = lims.FetchRequestsByDate(args); err != nil {
			log.Printf("Error fetching requests by date: %s\n", err)
		}
	} else if args.JSONFileMode {
		if err = lims.FetchRequestFromJSONFile(args); err != nil {
			log.Printf("Error fetching request from JSON file: %s\n", err)
		}
	} else if args.PublisherFileMode {
		if err = lims.FetchRequestFromPublisherFile(args); err != nil {
			log.Printf("Error fetching request from publisher file: %s\n", err)
		}
	} else if args.SmileServiceMode {
		if err = smile.FetchRequests(args); err != nil {
			log.Printf("Error fetching request from smile service: %s\n", err)
		}
	}
	return err
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
