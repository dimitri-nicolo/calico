// Copyright (c) 2016 Tigera, Inc. All rights reserved.

package ipfix

/*
#cgo pkg-config: libfixbuf glib-2.0
#include <fixbuf/public.h>
#include <glib.h>
#include <stdio.h>

static fbInfoElementSpec_t exportTemplate[] = {
    {"flowStartSeconds",               0, 0 },
    {"flowEndSeconds",                 0, 0 },
    {"octetTotalCount",                     0, 0 },
    {"reverseOctetTotalCount",              0, 0 },
    {"packetTotalCount",                    0, 0 },
    {"reversePacketTotalCount",             0, 0 },
    {"sourceIPv4Address",                   0, 0 },
    {"destinationIPv4Address",              0, 0 },
    {"sourceTransportPort",                 0, 0 },
    {"destinationTransportPort",            0, 0 },
    {"protocolIdentifier",                  0, 0 },
    {"flowEndReason",                       0, 0 },
    FB_IESPEC_NULL
};

// TODO(doublek): Support rule lists.
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

} exportRecord_t;

typedef struct fixbufData_st {
	fbConnSpec_t  exSocketDef;
	fbExporter_t  *exporter;
	fbSession_t   *exsession;
	fbTemplate_t  *etmpl;
	fBuf_t        *ebuf;
	uint16_t      etid;
	uint16_t      etid_ext;
} fixbufData_t;

static fbInfoModel_t *infoModel;

fixbufData_t fixbuf_init(char *host, char *port) {
	GError *err = NULL;
	infoModel = fbInfoModelAlloc();

	fbConnSpec_t exSocketDef;

	exSocketDef.transport = FB_TCP;
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
	fbData.etmpl = fbTemplateAlloc(infoModel);

	fbTemplateAppendSpecArray(fbData.etmpl, exportTemplate, 0xffffffff, &err);

	fbData.ebuf = fBufAllocForExport(fbData.exsession, fbData.exporter);

	fbData.etid = fbSessionAddTemplate(fbData.exsession, TRUE, FB_TID_AUTO, fbData.etmpl, &err);

	if (fbData.etid == 0) {
		printf("Couldn't fbSessionAddTemplate\n");
		return fbData;
	}

	fbData.etid_ext = fbSessionAddTemplate(fbData.exsession, FALSE, FB_TID_AUTO, fbData.etmpl, &err);

	if (fbData.etid_ext == 0) {
		printf("Couldn't fbSessionAddTemplate ext\n");
		return fbData;
	}

	return fbData;
}

GError * fixbuf_export(fixbufData_t fbData, exportRecord_t rec) {
	GError *err = NULL;
	exportRecord_t myrec;

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

	if(!fbSessionExportTemplates(fbData.exsession, &err)) {
		return err;
	}

	if(!fBufSetInternalTemplate(fbData.ebuf, fbData.etid, &err)) {
		printf("Couldn't fBufSetInternalTemplate %s\n", err->message);
		return err;
	}
	if(!fBufSetExportTemplate(fbData.ebuf, fbData.etid_ext, &err)) {
		printf("error fBufSetExportTemplate %s\n", err->message);
		return err;
	}
	if(!fBufAppend(fbData.ebuf, (uint8_t*)&myrec, sizeof(myrec), &err)) {
		printf("error fBufAppend %s\n", err->message);
		//if(!fBufEmit(fbData.ebuf, &err)){
		//	printf("error fBufEmit %s\n", err->message);
		//	return err;
		//}
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
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

// TODO(doublek): Convert Gerror to errors type
var (
	IPFIXExporterError = errors.New("Conflict in RuleTracePoint")
)

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
	FlowEndReason            int
}

type IPFIXExporter struct {
	host       net.IP
	port       int
	fixbufData C.fixbufData_t
	source     <-chan *ExportRecord
}

func NewIPFIXExporter(host net.IP, port int, source <-chan *ExportRecord) *IPFIXExporter {
	fbData := C.fixbuf_init(C.CString(string(host)), C.CString(strconv.Itoa(port)))
	return &IPFIXExporter{
		host:       host,
		port:       port,
		fixbufData: fbData,
		source:     source,
	}
}

func (ie *IPFIXExporter) Start() {
	go ie.startExporting()
}

func (ie *IPFIXExporter) startExporting() {
	for er := range ie.source {
		fmt.Println("exporting --- ", er)
		ie.export(er)
	}
}

func (ie *IPFIXExporter) export(data *ExportRecord) error {
	// TODO(doublek): Maybe we can reflect this information?
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
	fmt.Println("--- ", rec)
	gerror := C.fixbuf_export(ie.fixbufData, rec)
	if gerror != nil {
		// TODO(doublek): Convert Gerror to error
		fmt.Println("Error:", gerror.message)
		return IPFIXExporterError
	}
	return nil
}
