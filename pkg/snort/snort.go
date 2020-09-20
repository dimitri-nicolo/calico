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
