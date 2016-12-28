// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package ipfix

/*
#cgo pkg-config: libfixbuf glib-2.0
#include <fixbuf/public.h>
#include <glib.h>
#include <stdio.h>

#define TIGERA_PEN 49111

#define TIGERA_IENUM_TIERID		10
#define TIGERA_IENUM_POLICYID		11
#define TIGERA_IENUM_RULE		12
#define TIGERA_IENUM_RULE_IDX		13
#define TIGERA_IENUM_RULE_ACTION	14

static fbInfoElement_t tigeraElements[] = {
	FB_IE_INIT("tierId", TIGERA_PEN, TIGERA_IENUM_TIERID, FB_IE_VARLEN, 0),
	FB_IE_INIT("policyId", TIGERA_PEN, TIGERA_IENUM_POLICYID, FB_IE_VARLEN, 0),
	FB_IE_INIT("rule", TIGERA_PEN, TIGERA_IENUM_RULE, FB_IE_VARLEN, 0),
	FB_IE_INIT("ruleAction", TIGERA_PEN, TIGERA_IENUM_RULE_ACTION, FB_IE_VARLEN, 0),
	FB_IE_INIT("ruleIdx", TIGERA_PEN, TIGERA_IENUM_RULE_IDX, 2, 0),
	FB_IESPEC_NULL
};

static fbInfoElementSpec_t exportTemplate[] = {
	{"flowStartSeconds",		0, 0 },
	{"flowEndSeconds",		0, 0 },
	{"octetTotalCount",		0, 0 },
	{"reverseOctetTotalCount",	0, 0 },
	{"packetTotalCount",		0, 0 },
	{"reversePacketTotalCount",	0, 0 },
	{"sourceIPv4Address",		0, 0 },
	{"destinationIPv4Address",	0, 0 },
	{"sourceTransportPort",		0, 0 },
	{"destinationTransportPort",	0, 0 },
	{"protocolIdentifier",		0, 0 },
	{"flowEndReason",		0, 0 },
	{"paddingOctets",		2, 0 },
	{"subTemplateList",		0, 0 },
	FB_IESPEC_NULL
};

typedef struct exportRecord_st {
	uint32_t    flowStartSeconds;
	uint32_t    flowEndSeconds;
	uint64_t    octetTotalCount;
	uint64_t    reverseOctetTotalCount;
	uint64_t    packetTotalCount;
	uint64_t    reversePacketTotalCount;

	uint32_t    sourceIPv4Address;
	uint32_t    destinationIPv4Address;

	uint16_t    sourceTransportPort;
	uint16_t    destinationTransportPort;
	uint8_t     protocolIdentifier;
	uint8_t     flowEndReason;
	uint8_t     paddingOctets[2];

	fbSubTemplateList_t	ruleTrace;

} exportRecord_t;

static fbInfoElementSpec_t  ruleTraceTemplate[] = {
	{"tierId",		0, 0 },
	{"policyId",		0, 0 },
	{"rule",		0, 0 },
	{"ruleAction",		0, 0 },
	{"ruleIdx",		0, 0 },
	{"paddingOctets",	6, 0 },
	FB_IESPEC_NULL
};

typedef struct ruleTrace_st {
	fbVarfield_t	tierId;
	fbVarfield_t	policyId;
	fbVarfield_t	rule;
	fbVarfield_t	ruleAction;
	uint16_t	ruleIdx;
	uint8_t		paddingOctets[6];
} ruleTrace_t;

// Struct that can be used by go parts to pass in strings without having to
// worry about fbVarfield_t structures. fbVarfield_t struct contain a buf
// field, which is a pointer and will need memory allocated and such, which
// the boundary between go and C will probably make things complicated.
// Only the values of the strings used here will be used (i.e, copied) and not
// referenced.
typedef struct ruleTraceShim_st {
	char		*tierId;
	char		*policyId;
	char		*rule;
	char		*ruleAction;
	uint16_t	 ruleIdx;
} ruleTraceShim_t;

typedef struct fixbufData_st {
	fbConnSpec_t  exSocketDef;
	fbExporter_t  *exporter;
	fbSession_t   *exsession;

	fbTemplate_t  *exportTmpl;
	uint16_t      exportId;
	uint16_t      exportIdExt;

	fbTemplate_t  *ruleTraceTmpl;
	uint16_t      ruleTraceId;
	uint16_t      ruleTraceIdExt;

	fBuf_t        *ebuf;
} fixbufData_t;

static fbInfoModel_t *infoModel;

char *gchar_to_char(gchar *text) {
	return (char *)text;
}

fixbufData_t fixbuf_init(char *host, char *port, fbTransport_t transport) {
	GError *err = NULL;
	infoModel = fbInfoModelAlloc();
	fbInfoModelAddElementArray(infoModel, tigeraElements);

	fbConnSpec_t exSocketDef;

	exSocketDef.transport = transport;
	exSocketDef.host = host;
	exSocketDef.svc = port;
	// TODO(doublek): SSL Support.
	exSocketDef.ssl_ca_file = NULL;
	exSocketDef.ssl_cert_file = NULL;
	exSocketDef.ssl_key_file = NULL;
	exSocketDef.ssl_key_pass = NULL;
	exSocketDef.vai = NULL;
	exSocketDef.vssl_ctx = NULL;

	fixbufData_t fbData;

	fbData.exSocketDef = exSocketDef;
	fbData.exporter = fbExporterAllocNet(&exSocketDef);
	fbData.exsession = fbSessionAlloc(infoModel);

	fbData.exportTmpl = fbTemplateAlloc(infoModel);
	fbData.ruleTraceTmpl = fbTemplateAlloc(infoModel);
	fbTemplateAppendSpecArray(fbData.exportTmpl, exportTemplate, 0xffffffff, &err);
	fbTemplateAppendSpecArray(fbData.ruleTraceTmpl, ruleTraceTemplate, 0xffffffff, &err);

	fbData.ebuf = fBufAllocForExport(fbData.exsession, fbData.exporter);

	fbData.ruleTraceId = fbSessionAddTemplate(fbData.exsession, TRUE, FB_TID_AUTO, fbData.ruleTraceTmpl, &err);
	if (fbData.ruleTraceId == 0) {
		printf("Couldn't fbSessionAddTemplate\n");
		return fbData;
	}

	fbData.exportId = fbSessionAddTemplate(fbData.exsession, TRUE, FB_TID_AUTO, fbData.exportTmpl, &err);
	if (fbData.exportId == 0) {
		printf("Couldn't fbSessionAddTemplate\n");
		return fbData;
	}

	fbData.ruleTraceIdExt = fbSessionAddTemplate(fbData.exsession, FALSE, fbData.ruleTraceId, fbData.ruleTraceTmpl, &err);
	if (fbData.ruleTraceIdExt == 0) {
		printf("Couldn't fbSessionAddTemplate ext\n");
		return fbData;
	}

	fbData.exportIdExt = fbSessionAddTemplate(fbData.exsession, FALSE, fbData.exportId, fbData.exportTmpl, &err);
	if (fbData.exportIdExt == 1) {
		printf("Couldn't fbSessionAddTemplate ext\n");
		return fbData;
	}

	return fbData;
}

GError * fixbuf_export_templates(fixbufData_t fbData) {
	GError *err = NULL;

	if(!fbSessionExportTemplates(fbData.exsession, &err)) {
		return err;
	}
	if(!fBufEmit(fbData.ebuf, &err)){
		return err;
	}
	return NULL;
}

void fixbuf_fill_varfield(fbVarfield_t *varfield, char *value) {
	char *data;
	data = malloc(strlen(value));

	strcpy(data, value);
	varfield->len = strlen(data);
	varfield->buf = (uint8_t *)data;
}

GError * fixbuf_export_data(fixbufData_t fbData, exportRecord_t rec, ruleTraceShim_t *ruleTraceShimPtr, int numTraces) {
	int i;
	GError *err = NULL;
	exportRecord_t myrec;
	ruleTrace_t *ruleTracePtr = NULL;

	myrec.flowStartSeconds = rec.flowStartSeconds;
	myrec.flowEndSeconds = rec.flowEndSeconds;
	myrec.octetTotalCount = rec.octetTotalCount;
	myrec.reverseOctetTotalCount = rec.reverseOctetTotalCount;
	myrec.packetTotalCount = rec.packetTotalCount;
	myrec.reversePacketTotalCount = rec.reversePacketTotalCount;
	myrec.sourceIPv4Address = rec.sourceIPv4Address;
	myrec.destinationIPv4Address = rec.destinationIPv4Address;
	myrec.sourceTransportPort = rec.sourceTransportPort;
	myrec.destinationTransportPort = rec.destinationTransportPort;
	myrec.protocolIdentifier = rec.protocolIdentifier;
	myrec.flowEndReason = rec.flowEndReason;

	ruleTracePtr = (ruleTrace_t*)fbSubTemplateListInit(&(myrec.ruleTrace),
							0, fbData.ruleTraceId, fbData.ruleTraceTmpl, numTraces);
	for (i=0; i < numTraces; i++) {
		fixbuf_fill_varfield(&(ruleTracePtr->tierId), ruleTraceShimPtr->tierId);
		fixbuf_fill_varfield(&(ruleTracePtr->policyId), ruleTraceShimPtr->policyId);
		fixbuf_fill_varfield(&(ruleTracePtr->rule), ruleTraceShimPtr->rule);
		fixbuf_fill_varfield(&(ruleTracePtr->ruleAction), ruleTraceShimPtr->ruleAction);
		ruleTracePtr->ruleIdx = ruleTraceShimPtr->ruleIdx;

		ruleTracePtr++;
		ruleTraceShimPtr++;
	}

	if(!fBufSetInternalTemplate(fbData.ebuf, fbData.exportId, &err)) {
		return err;
	}
	if(!fBufSetExportTemplate(fbData.ebuf, fbData.exportIdExt, &err)) {
		return err;
	}

	if(!fBufAppend(fbData.ebuf, (uint8_t*)&myrec, sizeof(myrec), &err)) {
		return err;
	}

	if(err != NULL) {
		g_clear_error (&err);
	}
	return NULL;
}

*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/felix/go/felix/jitter"
)

const ExportingInterval = time.Duration(1) * time.Second

type FlowEndReasonType int

// Valid values of ExportRecord.FlowEndReason. Refer to
// http://www.iana.org/assignments/ipfix/ipfix.xhtml
// for an explanation of the different values below.
const (
	IdleTimeout     FlowEndReasonType = 0x01
	ActiveTimeout   FlowEndReasonType = 0x02
	EndOfFlow       FlowEndReasonType = 0x03
	ForcedEnd       FlowEndReasonType = 0x04
	LackOfResources FlowEndReasonType = 0x05
)

// An IPFIX record that is exported to IPFIX collectors. Refer to
// http://www.iana.org/assignments/ipfix/ipfix.xhtml
// for descriptions of the different fields that are exported.
type ExportRecord struct {
	FlowStart               time.Time
	FlowEnd                 time.Time
	OctetTotalCount         int
	ReverseOctetTotalCount  int
	PacketTotalCount        int
	ReversePacketTotalCount int

	SourceIPv4Address      net.IP
	DestinationIPv4Address net.IP

	SourceTransportPort      int
	DestinationTransportPort int
	ProtocolIdentifier       int
	FlowEndReason            FlowEndReasonType

	RuleTrace []RuleTraceRecord
}

type RuleTraceRecord struct {
	TierID     string
	PolicyID   string
	Rule       string
	RuleAction string
	RuleIndex  int
}

var fbTransport = map[string]C.fbTransport_t{
	"tcp": C.FB_TCP,
	"udp": C.FB_UDP,
}

type IPFIXExporter struct {
	host           net.IP
	port           int
	fixbufData     C.fixbufData_t
	templateTicker *jitter.Ticker
	source         <-chan *ExportRecord
}

// IPFIXExporter connects (and/or sends) IPFIX messages (ExportRecord objects),
// that are sent over the source channel, to a IPFIX collector listening on
// `host:port` over `transport`. transport can be either "tcp" or "udp" depending
// on the IPFIX collectors configuration.
func NewIPFIXExporter(host net.IP, port int, transport string, source <-chan *ExportRecord) *IPFIXExporter {
	log.Info("Creating IPFIX exporter to host ", host, " port ", port)
	fbData := C.fixbuf_init(C.CString(host.String()), C.CString(strconv.Itoa(port)), fbTransport[transport])
	return &IPFIXExporter{
		host:           host,
		port:           port,
		fixbufData:     fbData,
		source:         source,
		templateTicker: jitter.NewTicker(ExportingInterval, ExportingInterval/10),
	}
}

func (ie *IPFIXExporter) Start() {
	go ie.startExporting()
}

func (ie *IPFIXExporter) startExporting() {
	for {
		select {
		case erec := <-ie.source:
			log.Debugf("IPFIXExporter: Exporting %v", erec)
			err := ie.exportData(erec)
			if err != nil {
				log.Error(err)
			}
		case <-ie.templateTicker.C:
			log.Debug("Template export timer ticked")
			ie.exportTemplate()
		}
	}
}

func (ie *IPFIXExporter) exportTemplate() error {
	gerror := C.fixbuf_export_templates(ie.fixbufData)
	if gerror != nil {
		return fmt.Errorf("Couldn't export Templates Reason: %v", C.GoString(C.gchar_to_char(gerror.message)))
	}
	return nil
}

func (ie *IPFIXExporter) exportData(data *ExportRecord) error {
	// TODO(doublek): Maybe we can reflect this information?
	// TODO(doublek): Move this as a method to the ExportRecord struct.
	rec := C.struct_exportRecord_st{
		flowStartSeconds:         C.uint32_t(data.FlowStart.Unix()),
		flowEndSeconds:           C.uint32_t(data.FlowEnd.Unix()),
		octetTotalCount:          C.uint64_t(data.OctetTotalCount),
		reverseOctetTotalCount:   C.uint64_t(data.ReverseOctetTotalCount),
		packetTotalCount:         C.uint64_t(data.PacketTotalCount),
		reversePacketTotalCount:  C.uint64_t(data.ReversePacketTotalCount),
		sourceIPv4Address:        C.uint32_t(binary.BigEndian.Uint32(data.SourceIPv4Address.To4())),
		destinationIPv4Address:   C.uint32_t(binary.BigEndian.Uint32(data.DestinationIPv4Address.To4())),
		sourceTransportPort:      C.uint16_t(data.SourceTransportPort),
		destinationTransportPort: C.uint16_t(data.DestinationTransportPort),
		protocolIdentifier:       C.uint8_t(data.ProtocolIdentifier),
		flowEndReason:            C.uint8_t(data.FlowEndReason),
	}
	rtRec := []C.struct_ruleTraceShim_st{}
	for _, rt := range data.RuleTrace {
		// TODO(doublek): Move this as a method to the RuleTrace struct.
		rtRec = append(rtRec, C.struct_ruleTraceShim_st{
			tierId:     C.CString(rt.TierID),
			policyId:   C.CString(rt.PolicyID),
			rule:       C.CString(rt.Rule),
			ruleAction: C.CString(rt.RuleAction),
			ruleIdx:    C.uint16_t(rt.RuleIndex),
		})
	}
	log.Debug("Produced record for export: ", rec, " with rule trace: ", rtRec)
	var err error = nil
	if len(rtRec) != 0 {
		gerror := C.fixbuf_export_data(ie.fixbufData, rec, (*C.struct_ruleTraceShim_st)(&rtRec[0]), C.int(len(rtRec)))
		if gerror != nil {
			err = fmt.Errorf("Couldn't export %v Reason: %v", rec, C.GoString(C.gchar_to_char(gerror.message)))
		}
	} else {
		gerror := C.fixbuf_export_data(ie.fixbufData, rec, nil, C.int(len(rtRec)))
		if gerror != nil {
			err = fmt.Errorf("Couldn't export %v Reason: %v", rec, C.GoString(C.gchar_to_char(gerror.message)))
		}
	}
	return err
}
