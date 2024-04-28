package utils

import (
	"os"
	"runtime"

	ipfsLog "github.com/ipfs/go-log/v2"
)

func GetDownloadPath(log *ipfsLog.ZapEventLogger) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorln("Error getting user home directory : ", err)
		return ""
	}

	var downloadPath string
	switch runtime.GOOS {
	case "windows":
		downloadPath = homeDir + "\\Downloads\\"
		log.Debugln("The os is windows")
	case "darwin":
		downloadPath = homeDir + "/Downloads/"
		log.Debugln("The os is darwin")
	case "linux":
		downloadPath = homeDir + "/Downloads/"
		log.Debugln("The os is linux")
	default:
		downloadPath = homeDir
		log.Debugln("The os is ", runtime.GOOS)
	}

	log.Debugln("Download path : ", downloadPath)
	return downloadPath
}
