package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/tigera/egress-gateway/controlplane"
	"github.com/tigera/egress-gateway/data"
	"github.com/tigera/egress-gateway/sync"

	docopt "github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	VERSION   string // value is the build's GIT_VERSION
	USAGE_FMT string = `Egress Daemon - L2 and L3 management daemon for Tigera egress gateways.

Usage:
  %[1]s start [options]
  %[1]s (-h | --help)
  %[1]s --version
	
Options:
  --socket-path=<path>    Path to nodeagent-UDS over-which routing information is pulled [default: /var/run/nodeagent/socket]
  --log-severity=<trace|debug|info|warn|error|fatal>    Minimum reported log severity [default: info]
  --vni=<vni>    The VNI of the VXLAN interface being programmed [default: 4097]`
)

func main() {
	args, err := docopt.ParseArgs(fmt.Sprintf(USAGE_FMT, os.Args[0]), nil, VERSION)
	if err != nil {
		return
	}

	if logSeverity, ok := args["--log-severity"].(string); ok {
		ls, err := log.ParseLevel(logSeverity)
		if err != nil {
			docopt.DefaultParser.HelpHandler(err, fmt.Sprintf(USAGE_FMT, os.Args[0]))
		}

		log.SetLevel(ls)
	}

	var vni int
	vnis, ok := args["--vni"].(string)
	if !ok {
		log.Fatalf("invalid VNI value '%v'", args["--vni"])
	}
	vni, err = strconv.Atoi(vnis)
	if err != nil {
		log.Fatalf("invalid VNI value '%v'", args["--vni"])
	}

	syncSocket, ok := args["--socket-path"].(string)
	if !ok {
		log.Fatalf("invalid socket path value '%v'", args["--socket-path"])
	}

	log.Debugf("Starting %s with config: %v", os.Args[0], args)

	// source felix updates (over gRPC)
	syncClient := sync.NewClient(syncSocket, getDialOptions())
	datastore := data.NewRouteStore(syncClient.GetUpdatesPipeline)

	// register datastore observers
	routeManager := controlplane.NewRouteManager(datastore, "vxlan0", vni)

	// begin syncing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go routeManager.Start(ctx)
	go datastore.SyncForever(ctx)
	go syncClient.SyncForever(ctx)

	// Block until a signal is received.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Infof("Received interrupt: %v \n Exiting...", <-c)
}

func getDialOptions() []grpc.DialOption {
	d := &net.Dialer{}
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(
			func(ctx context.Context, target string) (net.Conn, error) {
				return d.DialContext(ctx, "unix", target)
			},
		),
	}
}
