package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	rest "k8s.io/client-go/rest"
        //. "github.com/onsi/gomega"

	"github.com/Microsoft/hcsshim"
	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/current"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mcuadros/go-version"
	//"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
	"github.com/projectcalico/cni-plugin/internal/pkg/utils"
	plugintypes "github.com/projectcalico/cni-plugin/pkg/types"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	log "github.com/sirupsen/logrus"
)

const K8S_NONE_NS = "none"

func SetCertFilePath(config *rest.Config) *rest.Config {
	log.WithField("config:", config).Info("AKHILESH")
	config.TLSClientConfig.CertFile = os.Getenv("CERT_DIR") + "\\client.crt"
	config.TLSClientConfig.KeyFile = os.Getenv("CERT_DIR") + "\\client.key"
	config.TLSClientConfig.CAFile = os.Getenv("CERT_DIR") + "\\ca.crt"
	log.WithField("config:", config).Info("AKHILESH")
	return config
}

func CreateContainerUsingDocker() (string, error) {
	cmd := exec.Command("powershell.exe", "docker run --net none -d -i microsoft/powershell:nanoserver pwsh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	temp := out[:len(out)-1]
	fmt.Printf("container ID:\n%s\n", string(temp))
	return string(temp), nil
}

func CreateWindowsContainer() (string, error) {
	fmt.Printf("\nEntered func")
	ctx := context.Background()
	cli, err := dockerclient.NewEnvClient()
	if err != nil {
		fmt.Printf("\nError creating client")
		return "", err
	}
	fmt.Printf("\nClient created")

	_, err = cli.ImagePull(ctx, "microsoft/powershell:nanoserver", dockertypes.ImagePullOptions{})
	if err != nil {
		fmt.Printf("\nError pulling image")
		return "", err
	}
	fmt.Printf("\nImage pulled")

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:           "microsoft/powershell:nanoserver",
		Cmd:             []string{"pwsh.exe", "while(1){}"},
		NetworkDisabled: true,
		//StopTimeout: &timeout,
	}, nil, nil, "")
	if err != nil {
		fmt.Printf("\nError creating container")
		return "", err
	}
	fmt.Printf("\nContainer created")

	if err := cli.ContainerStart(ctx, resp.ID, dockertypes.ContainerStartOptions{}); err != nil {
		fmt.Printf("\nError starting container")
		return "", err
	}
	fmt.Printf("\nContainer started")

	out, err := cli.ContainerLogs(ctx, resp.ID, dockertypes.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		fmt.Printf("\nError getting logs")
		return "", err
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	fmt.Printf("\nExiting func")
	return resp.ID, nil
}

func CreateContainer(netconf, podName, podNamespace, ip, k8sNs string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateWindowsContainer()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}
	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "", k8sNs)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", nil, "", []string{}, []string{}, err
	}
	return
}

// Create container with the giving containerId when containerId is not empty
//
// Deprecated: Please call CreateContainerNamespace and then RunCNIPluginWithID directly.
func CreateContainerWithId(netconf, podName, podNamespace, ip, overrideContainerID, k8sNs string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateWindowsContainer()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}

	log.WithField("containerID", containerID).Info("calling  RunCNIPluginWithId with:")
	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "", k8sNs)
	if err != nil {
		fmt.Println("Error: ", err)
		return "", nil, "", []string{}, []string{}, err
	}
	return
}

// RunCNIPluginWithId calls CNI plugin with a containerID and targetNs passed to it.
// This is for when you want to call CNI for an existing container.
func RunCNIPluginWithId(
	netconf,
	podName,
	podNamespace,
	ip,
	containerId,
	ifName,
	k8sNs string,
	//targetNs ns.NetNS,
) (
	result *current.Result,
	contVeth string,
	contAddr []string,
	contRoutes []string,
	err error,
) {
	log.Infof("Inside RunCNIPluginWithId")
	// Set up the env for running the CNI plugin
	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)

		// Append IP=<ip> to CNI_ARGS only if it's not an empty string.
		if ip != "" {
			k8sEnv = fmt.Sprintf("%s;IP=%s", k8sEnv, ip)
		}
	}

	if ifName == "" {
		ifName = "eth0"
	}

	env := os.Environ()
	env = append(env, []string{
		"CNI_COMMAND=ADD",
		fmt.Sprintf("CNI_IFNAME=%s", ifName),
		fmt.Sprintf("CNI_PATH=%s", os.Getenv("BIN")),
		fmt.Sprintf("CNI_CONTAINERID=%s", containerId),
		fmt.Sprintf("CNI_NETNS=%s", podNamespace),
		k8sEnv,
	}...)
	log.WithField("env", env).Info("AKHILESH")
	args := &cniArgs{env}
	log.WithField("args", args).Info("AKHILESH")

	// Invoke the CNI plugin, returning any errors to the calling code to handle.
	log.Debugf("Calling CNI plugin with the following env vars: %v", env)
	var r types.Result
	pluginPath := fmt.Sprintf("%s\\%s", os.Getenv("BIN"), os.Getenv("PLUGIN"))
	//pluginPath := fmt.Sprintf("%s\\%s", "C:\\k", "calico")
	log.Debugf("pluginPath: %v", pluginPath)
	r, err = invoke.ExecPluginWithResult(pluginPath, []byte(netconf), args, nil)
	if err != nil {
		log.Errorf("invoke.ExecPluginWithResult %v", err)
		return
	}

	// Extract the target CNI version from the provided network config.
	var nc types.NetConf
	if err = json.Unmarshal([]byte(netconf), &nc); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}
	log.Infof("compare CNI VERSION")
	// Parse the result as the target CNI version.
	if version.Compare(nc.CNIVersion, "0.3.0", "<") {
		// Special case for older CNI verisons.
		var out []byte
		out, err = json.Marshal(r)
		log.Infof("CNI output: %s", out)
		r020 := types020.Result{}
		if err = json.Unmarshal(out, &r020); err != nil {
			log.Errorf("Error unmarshaling output to Result: %v\n", err)
			return
		}

		result, err = current.NewResultFromResult(&r020)
		if err != nil {
			return
		}

	} else {
		result, err = current.GetResult(r)
		if err != nil {
			return
		}
	}

	return
}

// Executes the Calico CNI plugin and return the error code of the command.
func DeleteContainer(netconf, podName, podNamespace, k8sNs string) (exitCode int, err error) {
	return DeleteContainerWithId(netconf, podName, podNamespace, "", k8sNs)
}

//func DeleteContainerWithId(netconf, netnspath, podName, podNamespace, containerId string) (exitCode int, err error) {
func DeleteContainerWithId(netconf, podName, podNamespace, containerId, k8sNs string) (exitCode int, err error) {
	//return DeleteContainerWithIdAndIfaceName(netconf, netnspath, podName, podNamespace, containerId, "eth0")
	return DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, "eth0", k8sNs)
}

//func DeleteContainerWithIdAndIfaceName(netconf, netnspath, podName, podNamespace, containerId, ifaceName string) (exitCode int, err error) {
func DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, ifaceName, k8sNs string) (exitCode int, err error) {
	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)
	}

	// Set up the env for running the CNI plugin
	env := os.Environ()
	env = append(env, []string{
		"CNI_COMMAND=DEL",
		fmt.Sprintf("CNI_CONTAINERID=%s", containerId),
		fmt.Sprintf("CNI_NETNS=%s", podNamespace),
		"CNI_IFNAME=" + ifaceName,
		fmt.Sprintf("CNI_PATH=%s", os.Getenv("BIN")),
		k8sEnv,
	}...)

	log.Infof("Deleting container with ID %v CNI plugin with the following env vars: %v", containerId, env)

	// Run the CNI plugin passing in the supplied netconf
	subProcess := exec.Command(fmt.Sprintf("%s\\%s", os.Getenv("BIN"), os.Getenv("PLUGIN")), netconf)
	subProcess.Env = env
	stdin, err := subProcess.StdinPipe()
	if err != nil {
		return
	}

	_, err = io.WriteString(stdin, netconf)
	if err != nil {
		return 1, err
	}
	_, err = io.WriteString(stdin, "\n")
	if err != nil {
		return 1, err
	}

	err = stdin.Close()
	if err != nil {
		return 1, err
	}

	//session, err := gexec.Start(subProcess, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	session, err := gexec.Start(subProcess, os.Stdout, os.Stderr)
	if err != nil {
		return
	}

	// Call the plugin. Will force a test failure if it hangs longer than 10s.
	session.Wait(30)
	//Expect(session).Should(gexec.Exit())
	exitCode = session.ExitCode()
	//now delete the container
	if containerId != "" {
		log.Infof("\ncalling DeleteWindowsContainer ")
		err = DeleteWindowsContainer(containerId)
		if err != nil {
			log.Errorf("Error deleting container %s", containerId)
		}
	}
	return
}

func DeleteWindowsContainer(containerId string) error {
	ctx := context.Background()
	cli, err := dockerclient.NewEnvClient()
	if err != nil {
		log.Infof("\nError creating client")
		return err
	}
	log.Infof("\nClient created")

	log.Infof("\ncontainer : %s", containerId)
	err = cli.ContainerStop(ctx, containerId, nil)
	if err != nil {
		log.Errorf("Error stopping container %s: %v", containerId, err)
		return err
	}

	err = cli.ContainerRemove(ctx, containerId, dockertypes.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		log.Errorf("Error removing container %s: %v", containerId, err)
		return err
	}
	log.Infof("delete successful")
	return nil
}

func NetworkPod(
	netconf string,
	podName string,
	ip string,
	ctx context.Context,
	calicoClient client.Interface,
	result *current.Result,
	containerID string,
	netns string,
	k8sNs string,
) (err error) {

	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, k8sNs)
		// Append IP=<ip> to CNI_ARGS only if it's not an empty string.
		if ip != "" {
			k8sEnv = fmt.Sprintf("%s;IP=%s", k8sEnv, ip)
		}
	}

	var args *skel.CmdArgs
	args = &skel.CmdArgs{
		ContainerID: containerID,
		Netns:       netns,
		IfName:      "eth0",
		Args:        k8sEnv,
		Path:        os.Getenv("BIN"),
		StdinData:   []byte(netconf),
	}
	conf := plugintypes.NetConf{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	var logger *log.Entry

	logger = log.WithFields(log.Fields{
		"ContainerID": containerID,
		"Pod":         podName,
		"Namespace":   netns,
	})
	_, _, err = utils.DoNetworking(ctx, calicoClient, args, conf, result, logger, "", nil)
	return err
}

func CheckNetwork(netconf string) (*hcsshim.HNSNetwork, error) {
	var conf plugintypes.NetConf
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}

	result := &current.Result{}

	_, subNet, _ := net.ParseCIDR(conf.IPAM.Subnet)

	var logger *log.Entry
	logger = log.WithFields(log.Fields{
		"Name": conf.Name,
	})

	var networkName string
	if conf.WindowsUseSingleNetwork {
		logger.WithField("name", conf.Name).Info("Overriding network name, only a single IPAM block will be supported on this host")
		networkName = conf.Name
	} else {
		networkName = utils.CreateNetworkName(conf.Name, subNet)
	}

	hnsNetwork, err := utils.EnsureNetworkExists(networkName, subNet, result, logger)
	if err != nil {
		logger.Errorf("Unable to create hns network %s", networkName)
		return nil, err
	}

	return hnsNetwork, nil
}

func CheckEndpoint(hnsNetwork *hcsshim.HNSNetwork, netconf string) (*hcsshim.HNSEndpoint, error) {
	var conf plugintypes.NetConf
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		log.Errorf("unmarshal err: ", err)
		panic(err)
	}

	var result *current.Result

	_, subNet, _ := net.ParseCIDR(conf.IPAM.Subnet)

	var logger *log.Entry
	logger = log.WithFields(log.Fields{
		"Name": conf.Name,
	})

	epName := hnsNetwork.Name + "_ep"
	hnsEndpoint, err := utils.CreateAndAttachHostEP(epName, hnsNetwork, subNet, result, logger)
	if err != nil {
		logger.Errorf("Unable to create host hns endpoint %s", epName)
		return nil, err
	}

	return hnsEndpoint, nil
}
