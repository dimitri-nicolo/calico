package fortimanager

const (
	FortiManagerUnknownError           = -01
	FortiManagerDuplicateObject        = -02
	FortiManagerObjectNotExist         = -03
	FortiManagerInvalidUrl             = -06
	FortiManagerNoPermission           = -11
	FortiManagerSessionRetry           = -11
	FortiManagerObjectInUse            = -10015
	FortiManagerEmptyMemberInAddrGroup = -9998
)

const (
	FortiGateReturnSuccess       = 200
	FortiGateBadRequest          = 400
	FortiGateNotAuthorized       = 401
	FortigateForbidden           = 403
	FortiGateResourceNotFound    = 404
	FortiGateMethodNotAllowed    = 405
	FortiGateInternalServerError = 500
)

type reqLoginData struct {
	User     string `json:"user"`
	Password string `json:"password"`
}
type respStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ReqIpaddrData struct {
	Name                string   `json:"name"`
	Comment             string   `json:"comment"`
	Subnet              []string `json:"subnet"`
	AssociatedInterface []string `json:"associated-interface"`
}
type reqIpaddrParams struct {
	URL  string        `json:"url"`
	DATA ReqIpaddrData `json:"data"`
}
type requestIpAddr struct {
	ID      int               `json:"id"`
	Method  string            `json:"method"`
	Params  []reqIpaddrParams `json:"params"`
	Session string            `json:"session"`
}

func (r *requestIpAddr) SetSession(session string) {
	r.Session = session
}

type reqLoginParams struct {
	URL  string       `json:"url"`
	DATA reqLoginData `json:"data"`
}

type requestLogin struct {
	ID      int              `json:"id"`
	Method  string           `json:"method"`
	Params  []reqLoginParams `json:"params"`
	Session string           `json:"session"`
}

func (r *requestLogin) SetSession(session string) {
	r.Session = session
}

type respLogin struct {
	ID     int `json:"id"`
	Result []struct {
		Status respStatus `json:"status"`
		URL    string     `json:"url"`
	} `json:"result"`
	Session string `json:"session"`
}
type respFWAddresses struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   []struct {
			Name                string   `json:"name"`
			Type                int      `json:"type"`
			Comment             string   `json:"comment"`
			Subnet              []string `json:"subnet"`
			AssociatedInterface []string `json:"associated-interface"`
		}
	} `json:"result"`
}

type respFWAddressByName struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   struct {
			Name                string   `json:"name"`
			Comment             string   `json:"comment"`
			Subnet              []string `json:"subnet"`
			AssociatedInterface []string `json:"associated-interface"`
		}
	} `json:"result"`
}

type respFWDeleteAddressByName struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
	} `json:"result"`
}
type ReqFWAddressGroupData struct {
	Name    string   `json:"name"`
	Comment string   `json:"comment"`
	Member  []string `json:"member"`
}
type respFWAddressGroupName struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   struct {
			Name    string   `json:"name"`
			Comment string   `json:"comment"`
			Member  []string `json:"member"`
		}
	} `json:"result"`
}

type respFWAddressGroups struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   []struct {
			Name    string   `json:"name"`
			Comment string   `json:"comment"`
			Member  []string `json:"member"`
		}
	} `json:"result"`
}

type reqFWAddressGroupParams struct {
	URL  string                `json:"url"`
	DATA ReqFWAddressGroupData `json:"data"`
}
type requestFWAddressGroup struct {
	ID      int                       `json:"id"`
	Method  string                    `json:"method"`
	Params  []reqFWAddressGroupParams `json:"params"`
	Session string                    `json:"session"`
}

func (r *requestFWAddressGroup) SetSession(session string) {
	r.Session = session
}

// FortiGate request-response types

type RespFortiGateFWAddressData struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	SubType string `json:"sub-type"`
	Comment string `json:"comment"`
	Subnet  string `json:"subnet"`
}

type RespFortiGateAddress struct {
	HttpMethod string                       `json:"http_method"`
	Vdom       string                       `json:"vdom"`
	Path       string                       `json:"path"`
	Name       string                       `json:"name"`
	Mkey       string                       `json:"mkey"`
	Status     string                       `json:"status"`
	HttpStatus int                          `json:"http_status"`
	Result     []RespFortiGateFWAddressData `json:"results"`
}

type RespFortiGateStatus struct {
	HttpMethod string `json:"http_method"`
	Vdom       string `json:"vdom"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Mkey       string `json:"mkey"`
	Status     string `json:"status"`
	HttpStatus int    `json:"http_status"`
}
type RespFortiGateFWAddressGrpData struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
	Member  []struct {
		Name string `json:"name"`
	} `json:"member"`
}
type RespFortiGateAddressGrp struct {
	HttpMethod string                          `json:"http_method"`
	Vdom       string                          `json:"vdom"`
	Path       string                          `json:"path"`
	Name       string                          `json:"name"`
	Mkey       string                          `json:"mkey"`
	Status     string                          `json:"status"`
	HttpStatus int                             `json:"http_status"`
	Result     []RespFortiGateFWAddressGrpData `json:"results"`
}

type FortiFWAddress struct {
	Name                string
	Type                string
	SubType             string
	Comment             string
	IpAddr              string
	Mask                string
	AssociatedInterface string
}

type FortiFWAddressGroup struct {
	Name    string
	Comment string
	Members []string
}

type FortiGateApiKey struct {
	FortiSecRefKey FortiSecretRefKey `yaml:"secretKeyRef"`
}

type FortiMgrPwdKey struct {
	FortiSecRefKey FortiSecretRefKey `yaml:"secretKeyRef"`
}

type FortiSecretRefKey struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

type FortiGateConfig struct {
	Name   string          `yaml:"name"`
	Ip     string          `yaml:"ip"`
	ApiKey FortiGateApiKey `yaml:"apikey"`
}

type FortiMgrConfig struct {
	Name        string         `yaml:"name"`
	Ip          string         `yaml:"ip"`
	Adom        string         `yaml:"adom"`
	Tier        string         `yaml:"tier"`
	Username    string         `yaml:"username"`
	PackageName string         `yaml:"packagename"`
	Password    FortiMgrPwdKey `yaml:"password"`
}

type FwFortiDevConfig struct {
	Name     string
	Ip       string
	Username string
	Password string
	Adom     string
	ApiKey   string
	PkgName  string
	Tier     string
}

type reqFWPolicyPkgParams struct {
	URL string `json:"url"`
}

type reqFWPolicyPkg struct {
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  []reqFWPolicyPkgParams `json:"params"`
	Session string                 `json:"session"`
}

func (r *reqFWPolicyPkg) SetSession(session string) {
	r.Session = session
}

type respFWPolicyPkgName struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   struct {
			Name    string `json:"name"`
			Package string `json:"pkg"`
		}
	} `json:"result"`
}

type respFWPolicy struct {
	ID     int `json:"id"`
	Result []struct {
		Url    string     `json:"url"`
		Status respStatus `json:"status"`
		Data   []struct {
			SrcAddr  []string `json:"srcaddr"`
			DstAddr  []string `json:"dstaddr"`
			Service  []string `json:"service"`
			Name     string   `json:"name"`
			Action   int      `json:"action"`
			Comments string   `json:"comments"`
		}
	} `json:"result"`
}

type FortiFWPolicy struct {
	SrcAddr  []string
	DstAddr  []string
	Service  []string
	Name     string
	Action   int
	Comments string
}

var FortiServicesTCP = map[string]string{
	"AFS3":       "7000:7009",
	"ALL_TCP":    "1:65535",
	"AOL":        "5190:5194",
	"BGP":        "179",
	"CVSPSERVER": "2401",
	"DCS-RPC":    "135",
	"DNS":        "53",
	"FINGER":     "79",
	"FTP":        "21",
	"FTP_GET":    "21",
	"FTP_PUT":    "21",
	"GOPHER":     "70",
	"H323":       "1720",
	"HTTP":       "80",
	"HTTPS":      "443",
	"IMAP":       "143",
	"IMAPS":      "993",
	"IRC":        "6660:6669",
	"KERBEROS":   "88,464",
	"L2TP":       "1701",
	"LDAP":       "389",
	"MMS":        "1755",
	"MS-SQL":     "1433:1434",
	"MYSQL":      "3306",
	"NFS":        "2049",
	"NNTP":       "119",
	"NTP":        "123",
	"NetMeeting": "1720",
	"ONC-RPC":    "111",
	"POP3":       "110",
	"POP3S":      "995",
	"PPTP":       "1723",
	"RDP":        "3389",
	"REXEC":      "512",
	"SAMBA":      "139",
	"SCCP":       "2000",
	"SIP":        "5060",
	"SMB":        "445",
	"SMTP":       "25",
	"SMTPS":      "465",
	"SNMP":       "161:162",
	"SOCKS":      "1080",
	"SQUID":      "3128",
	"SSH":        "22",
	"TELNET":     "23",
	"UUCP":       "540",
	"VDOLIVE":    "7000:7010",
	"VNC":        "5900",
	"WAIS":       "210",
	"WINS":       "1512",
	"X-WINDOWS":  "6000:6063",
}

var FortiServicesUDP = map[string]string{
	"DHCP":       "67:68",
	"DHCP6":      "546:547",
	"DNS":        "53",
	"GTP":        "3386",
	"IKE":        "500",
	"LDAP_UDP":   "389",
	"MGCP":       "2727",
	"NFS":        "2049",
	"NTP":        "123",
	"ONC-RPC":    "111",
	"RADIUS":     "1812:1813",
	"RAUDION":    "7070",
	"RIP":        "520",
	"RTSP":       "554",
	"SIP":        "5060",
	"SNMP":       "161:162",
	"SOCKS":      "1080",
	"SYSLOG":     "514",
	"TALK":       "517:518",
	"TFTP":       "69",
	"TRACEROUTE": "33434:33535",
}
var FortiServicesICMP = map[string]string{
	"ALL_ICMP":  "ICMP",
	"ALL_ICMP6": "ICMPv6",
	"PING":      "ICMP",
	"PING6":     "ICMPv6",
}
