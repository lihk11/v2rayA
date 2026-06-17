package service

import (
	"bytes"
	"fmt"
	"github.com/v2rayA/v2rayA/common"
	"github.com/v2rayA/v2rayA/core/iptables"
	"github.com/v2rayA/v2rayA/core/v2ray/asset"
	"github.com/v2rayA/v2rayA/core/v2ray/where"
	"os/exec"
	"strings"
)

var LowVersionError = fmt.Errorf("core version too low")

func IsV2rayServiceValid() bool {
	if !asset.DoesV2rayAssetExist("geoip.dat") || !asset.DoesV2rayAssetExist("geosite.dat") {
		return false
	}
	_, ver, err := where.GetV2rayServiceVersion()
	return err == nil && ver != ""
}

func IfTProxyModLoaded() bool {
	out, err := exec.Command("sh", "-c", "lsmod|grep xt_TPROXY").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func CheckAndProbeTProxy() (err error) {
	if !IfTProxyModLoaded() && !common.IsDocker() && !iptables.IsNft() { //docker下无法判断，nft不需要
		var out []byte
		out, err = exec.Command("sh", "-c", "modprobe xt_TPROXY").CombinedOutput()
		if err != nil {
			if !strings.Contains(string(out), "not found") {
				return fmt.Errorf("failed to modprobe xt_TPROXY: %v", string(out))
			}
			// modprobe失败，不支持xt_TPROXY方案
			return fmt.Errorf("not support xt_TPROXY: %v", string(out))
		}
	}
	return
}

func isVersionSatisfied(version string) (where.Variant, error) {
	variant, ver, err := where.GetV2rayServiceVersion()
	if err != nil {
		return where.Unknown, fmt.Errorf("failed to get the version of v2ray-core: %v", err)
	}
	if greaterEqual, err := common.VersionGreaterEqual(ver, version); err != nil || !greaterEqual {
		return variant, fmt.Errorf("%w: the version %v is lower than %v", LowVersionError, ver, version)
	}
	return variant, nil
}

func CheckV5() (variant where.Variant, err error) {
	variant, ver, err := where.GetV2rayServiceVersion()
	if err != nil {
		return variant, err
	}
	// v2raya_core is xray-based and does not follow v2ray v5 versioning
	if variant == where.V2rayaCore {
		return variant, nil
	}
	if greaterEqual, err := common.VersionGreaterEqual(ver, "5.0.0"); err != nil || !greaterEqual {
		return variant, fmt.Errorf("%w: the version %v is lower than 5.0.0", LowVersionError, ver)
	}
	return variant, nil
}
