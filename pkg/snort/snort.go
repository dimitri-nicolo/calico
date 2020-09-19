package snort

import (
        _ "os/exec"
	"strings"
	_ "context"
        //api "github.com/tigera/lma/pkg/api"
	_ "github.com/tigera/lma/pkg/elastic"
	_ "fmt"
	_ "time"
	_ "net/url"

	//hp "github.com/tigera/honeypod-controller/pkg/processor"

	_ "os"
	_ "os/exec"
	_ "bytes"
	_ "path/filepath"
	_ "io/ioutil"
)

/*
type SnortProcessor struct {
    instance
}
type AlertLogProcessor struct {
    Ctx context.Context
    logHandler api.AlertLogReportHandler
}*/
var GlobalSnort []Snort

type Snort struct {
    SigName string
    Category string
    Date_Src_Dst string
    Flags string
    Other string
}

func ParseSnort(reader_str string) ([]Snort, error) {
    list := strings.Split(reader_str,"\n\n")
    var result []Snort
    for _, item := range list {
	list2 := strings.Split(item,"\n")
	if len(list2) >= 5 {
	    var tmp Snort
	    tmp.SigName = list2[0]
	    tmp.Category = list2[1]
	    tmp.Date_Src_Dst = list2[2]
	    tmp.Flags = list2[3]
	    tmp.Other = list2[4]
	    result = append(result, tmp)
	}
    }
    return result, nil
}

func FilterSnort(SnortList []Snort) ([]Snort, error) {
     var tmpList []Snort
     if len(GlobalSnort) == 0 {
         GlobalSnort = append(GlobalSnort, SnortList...)
         return SnortList, nil
     }
     for _, items := range SnortList {
	 found := 0
         for _, items2 := range GlobalSnort {
	     if items.Date_Src_Dst == items2.Date_Src_Dst {
	         found = 1
	     }
	 }
	 if found == 0 {
             tmpList = append(tmpList, items)
         }
     }
     //fmt.Println(tmpList)
     GlobalSnort = append(GlobalSnort, tmpList...)
     return tmpList, nil
}

/*
func SendEvents(SnortList []Snort, ctx context.Context, p hp.HoneypodLogProcessor, e api.AlertResult) error {
    /*	
    cfg := elastic.MustLoadConfig()
    cfg.ElasticURI = "https://tigera-secure-es-http.tigera-elasticsearch.svc:9200"
    cfg.ParsedElasticURL, _ = url.Parse(cfg.ElasticURI)
    c, err := elastic.NewFromConfig(cfg)
    index := "tigera_secure_ee_events.cluster"
    exists, err := c.Backend().IndexExists(index).Do(context.Background())
    if err != nil {
        fmt.Println("err")
        fmt.Println(exists)
    }
    fmt.Println("probly exist")
    fmt.Println(exists)
    //ctx = context.Background()
    //json := '{"interface" : "caliddd0428b7a8","alerts" : "hacker","time": "04/22/20-23:59:00"}'
    for _,alert := range SnortList {
        snort_description := fmt.Sprintf("[Snort] Signature Triggered on %s/%s", *e.Record.DestNamespace, *e.Record.DestNameAggr)
        json_res := map[string]interface{}{
            "severity": 100,
            "description": snort_description,
            "alert": "honeypod.snort",
            "type" : "alert",
            "record": map[string]interface{}{
	        "snort": map[string]interface{}{
			"Descripton": alert.SigName,
			"Category": alert.Category,
			"Occurance": alert.Date_Src_Dst,
			"Flags": alert.Flags,
			"Other": alert.Other,
		},
            },
            "time" : time.Now(),
        }
        fmt.Println(json_res)
        res, err := c.Backend().Index().Index(index).Id("").BodyJson(json_res).Do(ctx)
        if err != nil {
            fmt.Println("Send Failed", err)
        }
        fmt.Println(res)
    }
    return nil
}
func Loop(ctx context.Context, p hp.HoneypodLogProcessor, node string) error {
    endTime := time.Now()
    startTime := endTime.Add(-10 * time.Minute)
    for e := range p.LogHandler.SearchAlertLogs(ctx, nil, &startTime, &endTime) {
        if e.Err != nil {
            fmt.Println("search fial")
	}
	//fmt.Println(e.SourceNamespace)
	//fmt.Println(e.DestNamespace)

	//fmt.Println("Type: ", e.Type)
	//fmt.Println("Description: ", e.Description)
	//fmt.Println("Alert: ", e.Alert)
	//fmt.Println("Record SourceNameAggr: ", *e.Record.SourceNameAggr)
	//fmt.Println("Record SourceNamespace: ", *e.Record.SourceNamespace)
	//fmt.Println("Record DestNamespace:", *e.Record.DestNamespace)
	//fmt.Println("Record DestNameAggr: ", *e.Record.DestNameAggr)
	//fmt.Println("Record HostKeyword: ", *e.Record.HostKeyword)

	if *e.Record.HostKeyword != node {
            continue
	}

	s := fmt.Sprintf("/pcap/%s/%s/%s", *e.Record.DestNamespace,"capture-honey", *e.Record.DestNameAggr)
	fmt.Println(s)

        //pcap/tigera-internal/capture-honey/tigera-internal-1-x2qwd_calicf3a8b433d5.pcap
	if _, err := os.Stat("/pcap/"); os.IsNotExist(err) {
	    fmt.Println("/pcap directory missing.")
	}
        matches, err := filepath.Glob(s)
	if err != nil {
	    fmt.Println("/pcap file.")
	}
	fmt.Println(matches)
	for _ , match := range matches {
	    //output := fmt.Sprintf("/snort/%s", match)
            //fmt.Println("Base: ", filepath.Base(match))
	    output := fmt.Sprintf("/snort/%s", *e.Record.DestNameAggr)
            if _, err := os.Stat(output); os.IsNotExist(err) {
                err = os.Mkdir(output, 755)
	        if err != nil {
	           fmt.Println("can't create snort folder")
                }
	    }
	    cmd := exec.Command("snort", "-q", "-k", "none", "-c", "/etc/snort/snort.conf", "-r", match, "-l", output)
	    fmt.Println("Exec: ", cmd.String())
	    //cmd := exec.Command("ls", match)
	    var out bytes.Buffer
	    cmd.Stdout = &out
	    err := cmd.Run()
	    if err != nil {
	        fmt.Println("exec failed")
	    }
	    fmt.Println(out.String())
	}
        matches, err = filepath.Glob("/snort/*")
	if err != nil {
	    fmt.Println("/snort file.")
	}
	fmt.Println(matches)
	for _, match := range matches {
	    path := fmt.Sprintf("%s/alert", match)
	    if _, err := os.Stat(path); os.IsNotExist(err) {
	        fmt.Println(path, " missing.")
	    }
	    reader, err := ioutil.ReadFile(path)
	    if err != nil {
	        fmt.Println("Read Error")
	    }
	    //fmt.Println(string(reader))
	    reader_str := string(reader)
	    SnortList, err := ParseSnort(reader_str)
	    if err != nil {
                fmt.Println("Parse Error")
	    }
	    FilterList, err := FilterSnort(SnortList)
	    if err != nil {
                fmt.Println("Filter Error")
	    }
	    for _, list := range FilterList {
	        fmt.Println(list.Date_Src_Dst)
	    }
	    w := *e
	    err = SendEvents(FilterList, ctx, p, w)
	    if err != nil {
	        fmt.Println("SendEvent Failed.")
	    }
	}

    }
    fmt.Println("done2")

    return nil
}

*/
