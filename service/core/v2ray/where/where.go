package where

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/v2rayA/v2rayA/conf"
)

// CoreVersionMismatchError is returned when the core version does not match the expected version.
var CoreVersionMismatchError = fmt.Errorf("core version mismatch")

// CheckCoreVersion checks whether the core binary at corePath reports a version
// that exactly matches expectedVersion. If not, it returns CoreVersionMismatchError
// with details about the actual vs expected version.
func CheckCoreVersion(corePath string, expectedVersion string) error {
	cmd := exec.Command(corePath, "version")
	output := bytes.NewBuffer(nil)
	cmd.Stdout = output
	cmd.Stderr = output
	go func() {
		time.Sleep(5 * time.Second)
		p := cmd.Process
		if p != nil {
			_ = p.Kill()
		}
	}()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to execute %s --version: %w", corePath, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to wait for %s --version: %w", corePath, err)
	}

	fields := strings.Fields(strings.TrimSpace(output.String()))
	if len(fields) < 2 {
		return fmt.Errorf("cannot parse version output from %s: %q", corePath, output.String())
	}

	// fields[0] is the binary name (e.g. "v2raya_core"), fields[1] is the version string.
	actualVersion := fields[1]
	// Strip leading 'v' if present for comparison
	actualVer := strings.TrimPrefix(actualVersion, "v")
	expectedVer := strings.TrimPrefix(expectedVersion, "v")

	if actualVer != expectedVer {
		return fmt.Errorf("%w: core version %q does not match v2raya version %q", CoreVersionMismatchError, actualVersion, expectedVersion)
	}
	return nil
}

type Variant string

const (
	Unknown Variant = "Unknown"
	// V2rayaCore is the merged v2raya-core binary (xray-core + MultiObservatory).
	// Binary name: v2raya_core
	V2rayaCore Variant = "V2rayaCore"

	// Deprecated aliases kept for smooth migration; treated as V2rayaCore internally.
	V2ray  = V2rayaCore
	Xray   = V2rayaCore
	Merged = V2rayaCore
)

var NotFoundErr = fmt.Errorf("not found")
var ServiceNameList = []string{"v2raya_core", "xray", "v2ray"}
var v2rayVersion struct {
	variant    Variant
	version    string
	binPath    string
	lastUpdate time.Time
	mu         sync.Mutex
}

/* DetectCoreTypeByBinaryName detects the variant from the binary file name. */
func DetectCoreTypeByBinaryName(binPath string) Variant {
	baseName := strings.ToLower(filepath.Base(binPath))
	baseName = strings.TrimSuffix(baseName, ".exe")
	switch baseName {
	case "v2raya_core", "v2ray", "xray":
		return V2rayaCore
	}
	return Unknown
}

/* get the version of v2ray-core without 'v' like 4.23.1 */
func GetV2rayServiceVersion() (variant Variant, ver string, err error) {
	// cache for 10 seconds
	v2rayVersion.mu.Lock()
	defer v2rayVersion.mu.Unlock()
	if time.Since(v2rayVersion.lastUpdate) < 10*time.Second {
		return v2rayVersion.variant, v2rayVersion.version, nil
	}

	envConfig := conf.GetEnvironmentConfig()
	v2rayPath, err := GetV2rayBinPath()
	if err != nil || len(v2rayPath) <= 0 {
		return Unknown, "", fmt.Errorf("cannot find v2ray executable binary")
	}

	if envConfig.CoreType != "" {
		coreType := strings.ToLower(envConfig.CoreType)
		switch coreType {
		case "v2raya_core", "v2raya-core", "v2ray", "xray":
			variant = V2rayaCore
		default:
			variant = V2rayaCore
		}
	} else {
		variant = DetectCoreTypeByBinaryName(v2rayPath)
		if variant == Unknown {
			variant = V2rayaCore
		}
	}

	// Get version from binary
	cmd := exec.Command(v2rayPath, "version")
	output := bytes.NewBuffer(nil)
	cmd.Stdout = output
	cmd.Stderr = output
	go func() {
		time.Sleep(5 * time.Second)
		p := cmd.Process
		if p != nil {
			_ = p.Kill()
		}
	}()
	if err := cmd.Start(); err != nil {
		return Unknown, "", err
	}
	cmd.Wait()

	var fields []string
	if fields = strings.Fields(strings.TrimSpace(output.String())); len(fields) < 2 {
		return Unknown, "", fmt.Errorf("cannot parse version of v2ray")
	}
	ver = fields[1]


	v2rayVersion.variant = variant
	v2rayVersion.version = ver
	v2rayVersion.binPath = v2rayPath
	v2rayVersion.lastUpdate = time.Now()
	return
}

func GetV2rayBinPath() (string, error) {
	v2rayBinPath := conf.GetEnvironmentConfig().V2rayBin
	if v2rayBinPath == "" {
		return getV2rayBinPathAnyway()
	}
	return v2rayBinPath, nil
}

func getV2rayBinPathAnyway() (path string, err error) {
	for _, target := range ServiceNameList {
		if path, err = getV2rayBinPath(target); err == nil {
			return
		}
	}
	return
}

func getV2rayBinPath(target string) (string, error) {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(target), ".exe") {
		target += ".exe"
	}
	var pa string
	//从环境变量里找
	pa, err := exec.LookPath(target)
	if err == nil {
		return pa, nil
	}
	//从 pwd 里找
	pwd, err := os.Getwd()
	if err != nil {
		return "", NotFoundErr
	}
	pa = filepath.Join(pwd, target)
	if _, err := os.Stat(pa); err == nil {
		return pa, nil
	}
	return "", NotFoundErr
}
