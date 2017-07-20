package libv2ray

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	vlencoding "github.com/xiaokangwang/V2RayConfigureFileUtil/encoding"
)

var CurrentScan *QRScanContext

type QRScanContext struct {
	ec             *vlencoding.Encoder
	qd             *vlencoding.QRDecoder
	ScanReporter   QRScanReport
	vctx           *V2RayContext
	surpressFinish bool
}

func (qs *QRScanContext) OnNewScanResult(data string, allowdiscard bool) string {
	if qs.qd == nil {
		qs.qd = qs.ec.StartQRDecode()
		qs.surpressFinish = false
	}
	res := qs.ec.V2RayURLToByte(data)
	if res == nil {
		return "Unknown schema"
	}
	err := qs.qd.OnNewDataScanned(res)
	if err != nil {
		if strings.Contains(err.Error(), "inconsistent") && allowdiscard || true {
			qs.qd = nil
			qs.OnNewScanResult(data, allowdiscard)
		} else {
			return err.Error() + "\n[Tap to discard previous segment]"
		}
	}
	if qs.qd.IsDecodeReady() && !qs.surpressFinish {
		qs.surpressFinish = true
		qs.ScanReporter.ReadyToFinish()
	} else {
		qs.ScanReporter.StatUpdate(qs.qd.PieceNeeded, qs.qd.PieceReceived)
	}
	return "Scanned"
}

func (qs *QRScanContext) Init() {
	qs.ec = &vlencoding.Encoder{}
	CurrentScan = qs
}

type QRScanReport interface {
	ReadyToFinish()
	StatUpdate(need, acquired int)
}

func (qs *QRScanContext) Finish(name string) bool {
	if !qs.qd.IsDecodeReady() {
		fmt.Println("Called decoder when decode not ready")
		return false
	}
	dec, err := qs.qd.Finish()
	if err != nil {
		fmt.Println(err)
		return false
	}
	qs.qd = nil
	suf, towrite, err := qs.ec.UnpackV2RayConfB(dec)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fdr := filepath.Dir(qs.vctx.GetConfigureFile())
	err = ioutil.WriteFile(fmt.Sprintf("%v/%v%v", fdr, name, suf), towrite, 0600)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (qs *QRScanContext) Discard() {
	qs.qd = nil
}