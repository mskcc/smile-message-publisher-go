# smile_message_publisher_go

## Basic Usage

```bash

go run . -h

-f, --cfg_file string             Path to configuration file containing Lims, Nats settings & creds
-c, --cmo_requests_only string    Filter Lims requests by CMO requests flag
-e, --end_date string             End date [MM/DD/YYYY]. Fetch requests from LimsRest between the given start and end dates [START/END DATE MODE]
-h, --help                        Describes available options
-j, --json_filename string        Publishes contents of provided JSON file
-p, --publisher_filename string   Publishes contents of provided JSON file
-r, --request_ids string          Comma-separated list of request ids to fetch from LimSRest
-m, --smile_service string        Comma-separated list of request ids to fetch from Smile Web Service
-s, --start_date string           Start date [MM/DD/YYYY].  Fetch requests from LimsRest between the given start and end dates [START/END DATE MODE]

go run . -s 05/24/2022 -e 06/13/2022 -c true -f ./example-conf.yaml
go run . -r 05274_C,06048_BC -c true -f ./example-conf.yaml
go run . -j ./05274_C.json -c true -f ./example-conf.yaml
go run . -p ./publisher-file.txt -c true -f ./example-conf.yaml
go run . -m 05274_C -c true -f ./example-conf.yaml

where example-conf.yaml contains the proper lims/smile/nats properties
```
