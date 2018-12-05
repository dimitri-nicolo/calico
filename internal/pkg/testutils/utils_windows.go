package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/current"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mcuadros/go-version"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/backend"
	client "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"
)

const K8S_TEST_NS = "test"
const TEST_DEFAULT_NS = "default"

// Delete everything under /calico from etcd.
func WipeEtcd() {
	be, err := backend.NewClient(apiconfig.CalicoAPIConfig{
		Spec: apiconfig.CalicoAPIConfigSpec{
			DatastoreType: apiconfig.EtcdV3,
			EtcdConfig: apiconfig.EtcdConfig{
				EtcdEndpoints: os.Getenv("ETCD_ENDPOINTS"),
			},
		},
	})
	if err != nil {
		panic(err)
	}
	_ = be.Clean()

	// Set the ready flag so calls to the CNI plugin can proceed
	calicoClient, _ := client.NewFromEnv()
	newClusterInfo := api.NewClusterInformation()
	newClusterInfo.Name = "default"
	datastoreReady := true
	newClusterInfo.Spec.DatastoreReady = &datastoreReady
	ci, err := calicoClient.ClusterInformation().Create(context.Background(), newClusterInfo, options.SetOptions{})
	if err != nil {
		panic(err)
	}
	log.Printf("Set ClusterInformation: %v %v\n", ci, *ci.Spec.DatastoreReady)
}

// MustCreateNewIPPool creates a new Calico IPAM IP Pool.
func MustCreateNewIPPool(c client.Interface, cidr string, ipip, natOutgoing, ipam bool, blocksize int) string {
	log.SetLevel(log.DebugLevel)

	log.SetOutput(os.Stderr)

	name := strings.Replace(cidr, ".", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, "/", "-", -1)
	var mode api.IPIPMode
	if ipip {
		mode = api.IPIPModeAlways
	} else {
		mode = api.IPIPModeNever
	}

	pool := api.NewIPPool()
	pool.Name = name
	pool.Spec.CIDR = cidr
	pool.Spec.NATOutgoing = natOutgoing
	pool.Spec.Disabled = !ipam
	pool.Spec.IPIPMode = mode
	pool.Spec.BlockSize = blocksize

	_, err := c.IPPools().Create(context.Background(), pool, options.SetOptions{})
	if err != nil {
		panic(err)
	}
	return pool.Name
}

func MustDeleteIPPool(c client.Interface, cidr string) {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)

	name := strings.Replace(cidr, ".", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	name = strings.Replace(name, "/", "-", -1)

	_, err := c.IPPools().Delete(context.Background(), name, options.DeleteOptions{})
	if err != nil {
		panic(err)
	}
}
func SetCertFilePath(config *rest.Config) *rest.Config {
	log.WithField("config:", config).Info("AKHILESH")
	config.TLSClientConfig.CertFile = os.Getenv("CERT_DIR") + "\\client.crt"
	config.TLSClientConfig.KeyFile = os.Getenv("CERT_DIR") + "\\client.key"
	config.TLSClientConfig.CAFile = os.Getenv("CERT_DIR") + "\\ca.crt"
	log.WithField("config:", config).Info("AKHILESH")
	return config
}

// Delete all K8s pods from the "test" namespace
func WipeK8sPods() {
	config, err := clientcmd.DefaultClientConfig.ClientConfig()
	if err != nil {
		panic(err)
	}
	config = SetCertFilePath(config)
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err)
	}
	log.WithField("clientset:", clientset).Info("AKHILESH")
	pods, err := clientset.CoreV1().Pods(K8S_TEST_NS).List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		err = clientset.CoreV1().Pods(K8S_TEST_NS).Delete(pod.Name, &metav1.DeleteOptions{})

		if err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}
			panic(err)
		}
	}
	log.Info("WipeK8sPods Sucess")
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

func CreateContainer(netconf, podName, podNamespace, ip string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateWindowsContainer()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}
	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "")
	if err != nil {
		fmt.Println("Error: ", err)
		return "", nil, "", []string{}, []string{}, err
	}
	return
}

// Create container with the giving containerId when containerId is not empty
//
// Deprecated: Please call CreateContainerNamespace and then RunCNIPluginWithID directly.
func CreateContainerWithId(netconf, podName, podNamespace, ip, overrideContainerID string) (containerID string, result *current.Result, contVeth string, contAddr []string, contRoutes []string, err error) {
	containerID, err = CreateWindowsContainer()
	if err != nil {
		return "", nil, "", []string{}, []string{}, err
	}

	log.WithField("containerID", containerID).Info("calling  RunCNIPluginWithId with:")
	result, contVeth, contAddr, contRoutes, err = RunCNIPluginWithId(netconf, podName, podNamespace, ip, containerID, "")
	if err != nil {
		fmt.Println("Error: ", err)
		return "", nil, "", []string{}, []string{}, err
	}
	return
}

// Used for passing arguments to the CNI plugin.
type cniArgs struct {
	Env []string
}

func (c *cniArgs) AsEnv() []string {
	return c.Env
}

// RunCNIPluginWithId calls CNI plugin with a containerID and targetNs passed to it.
// This is for when you want to call CNI for an existing container.
func RunCNIPluginWithId(
	netconf,
	podName,
	podNamespace,
	ip,
	containerId,
	ifName string,
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
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, podNamespace)

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
		fmt.Sprintf("CNI_NETNS=%s", "none"),
		k8sEnv,
	}...)
	log.WithField("env", env).Info("AKHILESH")
	args := &cniArgs{env}
	log.WithField("args", args).Info("AKHILESH")

	// Invoke the CNI plugin, returning any errors to the calling code to handle.
	log.Debugf("Calling CNI plugin with the following env vars: %v", env)
	var r types.Result
	pluginPath := fmt.Sprintf("%s/%s", os.Getenv("BIN"), os.Getenv("PLUGIN"))
	//pluginPath := fmt.Sprintf("%s\\%s", "C:\\k", "calico")
	log.Debugf("pluginPath: %v", pluginPath)
	r, err = invoke.ExecPluginWithResult(pluginPath, []byte(netconf), args, nil)
	if err != nil {
		log.Errorf("iinvoke.ExecPluginWithResult %v", err)
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
func DeleteContainer(netconf, podName, podNamespace string) (exitCode int, err error) {
	return DeleteContainerWithId(netconf, podName, podNamespace, "")
}

//func DeleteContainerWithId(netconf, netnspath, podName, podNamespace, containerId string) (exitCode int, err error) {
func DeleteContainerWithId(netconf, podName, podNamespace, containerId string) (exitCode int, err error) {
	//return DeleteContainerWithIdAndIfaceName(netconf, netnspath, podName, podNamespace, containerId, "eth0")
	return DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, "eth0")
}

//func DeleteContainerWithIdAndIfaceName(netconf, netnspath, podName, podNamespace, containerId, ifaceName string) (exitCode int, err error) {
func DeleteContainerWithIdAndIfaceName(netconf, podName, podNamespace, containerId, ifaceName string) (exitCode int, err error) {
	k8sEnv := ""
	if podName != "" {
		k8sEnv = fmt.Sprintf("CNI_ARGS=K8S_POD_NAME=%s;K8S_POD_NAMESPACE=%s;K8S_POD_INFRA_CONTAINER_ID=whatever", podName, podNamespace)
	}

	// Set up the env for running the CNI plugin
	env := os.Environ()
	env = append(env, []string{
		"CNI_COMMAND=DEL",
		fmt.Sprintf("CNI_CONTAINERID=%s", containerId),
		fmt.Sprintf("CNI_NETNS=%s", "none"),
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

	session, err := gexec.Start(subProcess, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return
	}

	// Call the plugin. Will force a test failure if it hangs longer than 5s.
	session.Wait(10)

	exitCode = session.ExitCode()
	log.Infof("retuning from DeleteContainerWithIdAndIfaceName")
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
