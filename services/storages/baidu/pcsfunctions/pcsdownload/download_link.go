package pcsdownload

import (
	"Lefiles/services/storages/baidu/pcsconfig"
	"net/url"

	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
)

func GetLocateDownloadLinks(pcs *baidupcs.BaiduPCS, pcspath string) (dlinks []*url.URL, err error) {
	dInfo, pcsError := pcs.LocateDownload(pcspath)
	if pcsError != nil {
		return nil, pcsError
	}

	us := dInfo.URLStrings(pcsconfig.Config.EnableHTTPS)
	if len(us) == 0 {
		return nil, ErrDlinkNotFound
	}

	return us, nil
}
