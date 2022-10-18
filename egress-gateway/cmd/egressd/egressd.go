package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/projectcalico/calico/egress-gateway/controlplane"
	"github.com/projectcalico/calico/egress-gateway/data"
	"github.com/projectcalico/calico/egress-gateway/sync"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"

	docopt "github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	VERSION   string // value is the build's GIT_VERSION
	USAGE_FMT string = `Egress Daemon - L2 and L3 management daemon for Tigera egress gateways.

Usage:
  %[1]s start <gateway-ip> [options]
  %[1]s (-h | --help)
  %[1]s --version
	
Options:
  --socket-path=<path>    Path to nodeagent-UDS over-which routing information is pulled [default: /var/run/nodeagent/socket]
  --log-severity=<trace|debug|info|warn|error|fatal>    Minimum reported log severity [default: info]
  --vni=<vni>    The VNI of the VXLAN interface being programmed [default: 4097]

`
)

func main() {
	args, err := docopt.ParseArgs(fmt.Sprintf(USAGE_FMT, os.Args[0]), nil, VERSION)
	if err != nil {
		return
	}

	// parse this gateway's IP from string
	var ip net.IP
	if argEgressPodIP, ok := args["<gateway-ip>"].(string); !ok {
		exitWithErrorAndUsage(fmt.Errorf("invalid egress-gateway IP '%v'", args["<gateway-ip>"]))
	} else {
		ip = net.ParseIP(argEgressPodIP)
		if ip == nil {
			exitWithErrorAndUsage(fmt.Errorf("invalid egress-gateway IP '%v'", argEgressPodIP))
		}
	}

	// parse log severity
	if argLogSeverity, ok := args["--log-severity"].(string); !ok {
		exitWithErrorAndUsage(fmt.Errorf("invalid log severity value %v", args["--log-severity"]))
	} else {
		ls, err := log.ParseLevel(argLogSeverity)
		if err != nil {
			exitWithErrorAndUsage(err)
		}

		log.SetLevel(ls)
	}
	// Replace logrus' formatter with a custom one using our time format,
	// shared with the Python code.
	log.SetFormatter(&logutils.Formatter{Component: "felix"})
	// Install a hook that adds file/line no information.
	log.AddHook(&logutils.ContextHook{})

	var vni int
	argVNI, ok := args["--vni"].(string)
	if !ok {
		exitWithErrorAndUsage(fmt.Errorf("invalid VNI value '%v'", args["--vni"]))
	}
	vni, err = strconv.Atoi(argVNI)
	if err != nil {
		exitWithErrorAndUsage(fmt.Errorf("invalid VNI value '%s'", argVNI))
	}

	syncSocket, ok := args["--socket-path"].(string)
	if !ok {
		exitWithErrorAndUsage(fmt.Errorf("invalid socket path value '%v'", args["--socket-path"]))
	}

	log.Debugf("starting %s with config: %v", os.Args[0], args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// source felix updates (over gRPC)
	syncClient := sync.NewClient(ctx, syncSocket, getDialOptions())
	datastore := data.NewRouteStore(syncClient.GetUpdatesPipeline, ip)
	// register datastore observers
	routeManager := controlplane.NewRouteManager(datastore, "vxlan0", vni)

	// begin syncing
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

func exitWithErrorAndUsage(err error) {
	fmt.Printf(USAGE_FMT, os.Args[0])
	log.Fatal(err)
}
