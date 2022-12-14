package template

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kelseyhightower/memkv"
	"github.com/projectcalico/calico/confd/pkg/backends/calico"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxBIRDSymLen = 64
)

func newFuncMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["base"] = path.Base
	m["split"] = strings.Split
	m["json"] = UnmarshalJsonObject
	m["jsonArray"] = UnmarshalJsonArray
	m["dir"] = path.Dir
	m["map"] = CreateMap
	m["getenv"] = Getenv
	m["join"] = strings.Join
	m["datetime"] = time.Now
	m["toUpper"] = strings.ToUpper
	m["toLower"] = strings.ToLower
	m["contains"] = strings.Contains
	m["replace"] = strings.Replace
	m["hasSuffix"] = strings.HasSuffix
	m["lookupIP"] = LookupIP
	m["lookupSRV"] = LookupSRV
	m["fileExists"] = isFileExist
	m["base64Encode"] = Base64Encode
	m["base64Decode"] = Base64Decode
	m["hashToIPv4"] = hashToIPv4
	m["emitBIRDExternalNetworkConfig"] = EmitBIRDExternalNetworkConfig
	m["emitExternalNetworkTableName"] = EmitExternalNetworkTableName
	return m
}

func addFuncs(out, in map[string]interface{}) {
	for name, fn := range in {
		out[name] = fn
	}
}

func EmitExternalNetworkTableName(name string) (string, error) {
	// call truncateAndHashName here to normalize
	pieces := []string{"T_", ""}
	// resizedName, err := truncateAndHashName(name, maxBIRDSymLen - len(strings.Join(pieces, "")))
	// if err != nil {
	//     return "", err
	// }
	// pieces[1] = resizedName
	pieces[1] = name
	fullName := strings.Join(pieces, "")
	return fmt.Sprintf("'%s'", fullName), nil
}

func EmitBIRDExternalNetworkConfig(externalNetworkKVPs memkv.KVPairs, globalPeersKVP memkv.KVPairs,
	nodeSpecificPeersKVP memkv.KVPairs) ([]string, error) {
	lines := []string{}
	var line string
	if len(externalNetworkKVPs) == 0 {
		line = fmt.Sprint("# No ExternalNetworks configured")
		lines = append(lines, line)
		return lines, nil
	}
	for _, kvp := range externalNetworkKVPs {
		var externalNetwork v3.ExternalNetwork
		err := json.Unmarshal([]byte(kvp.Value), &externalNetwork)
		if err != nil {
			return []string{}, fmt.Errorf("Error unmarshalling JSON into ExternalNetwork: %s", err)
		}
		var routeTableIndex uint32
		if externalNetwork.Spec.RouteTableIndex != nil {
			routeTableIndex = *externalNetwork.Spec.RouteTableIndex
		}
		externalNetworkName := path.Base(kvp.Key)
		// call truncateAndHashName here to normalize
		tableName, err := EmitExternalNetworkTableName(externalNetworkName)
		if err != nil {
			return []string{}, err
		}
		kernelName := fmt.Sprintf("'K_%s'", externalNetworkName)
		line = fmt.Sprintf("# ExternalNetwork %s", externalNetworkName)
		lines = append(lines, line)

		line = fmt.Sprintf("table %s;", tableName)
		lines = append(lines, line)

		//var bgpPeerProtocols []string

		for _, globalPeerKVP := range globalPeersKVP {
			var globalPeer calico.BackendBGPPeer
			err = json.Unmarshal([]byte(globalPeerKVP.Value), globalPeer)
			if err != nil {
				return []string{}, fmt.Errorf("Error unmarshalling JSON into BackendBGPPeer: %s", err)
			}
			if globalPeer.ExternalNetwork != "" {

			}
		}

		//for _, nodeSpecificPeerKVP := range nodeSpecificPeersKVP {

		//}

		kernel := []string{
			fmt.Sprintf("protocol kernel %s from kernel_template {", kernelName),
			"device routes yes;",
			fmt.Sprintf("  table %s;", tableName),
			fmt.Sprintf("  kernel table %d;", routeTableIndex),
			"  export filter {",
			"    print \"route: \", net, \", from, \", \", \", proto, \", \", bgp_next_hop;",
			" # Print \"if proto = <peer> then accept;\" for all global and node-specific peers",
			"    reject;",
			"  };",
			"}",
		}

		lines = append(lines, kernel...)
	}
	return lines, nil
}

// hashToIPv4 hashes the given string and
// formats the resulting 4 bytes as an IPv4 address.
func hashToIPv4(nodeName string) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(nodeName))
	if err != nil {
		return ""
	}
	hashBytes := hash.Sum(nil)
	ip := hashBytes[:4]
	routerId := strconv.Itoa(int(ip[0])) + "." +
		strconv.Itoa(int(ip[1])) + "." +
		strconv.Itoa(int(ip[2])) + "." +
		strconv.Itoa(int(ip[3]))
	return routerId
}

// Getenv retrieves the value of the environment variable named by the key.
// It returns the value, which will the default value if the variable is not present.
// If no default value was given - returns "".
func Getenv(key string, v ...string) string {
	defaultValue := ""
	if len(v) > 0 {
		defaultValue = v[0]
	}

	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// CreateMap creates a key-value map of string -> interface{}
// The i'th is the key and the i+1 is the value
func CreateMap(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("map keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func UnmarshalJsonObject(data string) (map[string]interface{}, error) {
	var ret map[string]interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

func UnmarshalJsonArray(data string) ([]interface{}, error) {
	var ret []interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

func LookupIP(data string) []string {
	ips, err := net.LookupIP(data)
	if err != nil {
		return nil
	}
	// "Cast" IPs into strings and sort the array
	ipStrings := make([]string, len(ips))

	for i, ip := range ips {
		ipStrings[i] = ip.String()
	}
	sort.Strings(ipStrings)
	return ipStrings
}

type sortSRV []*net.SRV

func (s sortSRV) Len() int {
	return len(s)
}

func (s sortSRV) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortSRV) Less(i, j int) bool {
	str1 := fmt.Sprintf("%s%d%d%d", s[i].Target, s[i].Port, s[i].Priority, s[i].Weight)
	str2 := fmt.Sprintf("%s%d%d%d", s[j].Target, s[j].Port, s[j].Priority, s[j].Weight)
	return str1 < str2
}

func LookupSRV(service, proto, name string) []*net.SRV {
	_, addrs, err := net.LookupSRV(service, proto, name)
	if err != nil {
		return []*net.SRV{}
	}
	sort.Sort(sortSRV(addrs))
	return addrs
}

func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func Base64Decode(data string) (string, error) {
	s, err := base64.StdEncoding.DecodeString(data)
	return string(s), err
}
