package update

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var Flag uint16
var ssuMutex *sync.RWMutex = new(sync.RWMutex)
var appMutex *sync.RWMutex = new(sync.RWMutex)

/*
//return false,the caller have to unpack the SSU,and inc Flag
func GetFlag() bool {
	m.RLock()
	defer m.RUnlock()
	if Flag == 0 {
		return false
	} else {
		return true
	}
}

//when unpack SSU done, it should call this function
func IncFlag() {
	m.Lock()
	defer m.Unlock()
	Flag++
}

//when upgrade success, it should call this function
func DecFlag() {
	m.Lock()
	defer m.Unlock()
	if Flag > 0 {
		Flag--
	}
}

//相同的版本的SSU只能解压一次,在没有解压完成之前其它goroute只能等待解压完成，需要channel来通信
var once sync.Once

func (S *Session) unpackSSU(ssu string) {

}
*/

/*
func UnpackSSU() {
	if !GetFlag() {
		IncFlag()
		//don't have to unpack SSU,because it has been unpacked
		return
	}
	//var name string
	var S Session
	once.Do(S.unpackSSU)

	IncFlag()
}
*/

func unpack(packPath, destPath, unpackTool, logFile string) error {
	if runtime.GOOS == "windows" {
		unpackTool = filepath.Join(CurrentDirectory(), "tool", "7z.exe")
	}
	newArgs := []string{
		0: "x",
		1: "-y",
		2: "-p" + SSU_DEC_PASSWD,
		3: packPath,
		4: "-o" + destPath,
	}

	new := exec.Command(unpackTool, newArgs...)
	stdout, _ := new.StdoutPipe()
	if err := new.Start(); err != nil {
		return err
	}
	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Error("[unpack]unpack log has been lost,error msg:%s", err)
	}
	if err := ioutil.WriteFile(logFile, data, 0664); err != nil {
		log.Error("[unpack]unpack log can't write it to logfile:", err)
	}
	if err := new.Wait(); err == nil {
		log.Info("[unpack]Unpack %s success", packPath)
		return nil
	} else {
		log.Debug("[unpack]use new password to unpack fail:", err)
	}

	oldArgs := []string{
		0: "x",
		1: "-y",
		2: "-p" + SSU_DEC_PASSWD_OLD,
		3: packPath,
		4: "-o" + destPath,
	}
	old := exec.Command(unpackTool, oldArgs...)
	stdout, _ = old.StdoutPipe()
	if err := old.Start(); err != nil {
		return err
	}
	data, err = ioutil.ReadAll(stdout)
	if err != nil {
		log.Error("[unpack]unpack log has been lost,error msg:%s", err)
	}
	if err := ioutil.WriteFile(logFile, data, 0664); err != nil {
		log.Error("[unpack]unpack log can't write it to logfile:", err)
	}
	if err := old.Wait(); err != nil {
		log.Info("[unpack]Unpack %s success", packPath)
		return fmt.Errorf("[unpack]use old or new password to unpack fail,err msg:%s", err)
	}
	log.Info("[unpack]Unpack %s success", packPath)
	return nil

}

func unpackPackage(md5 string, U *Update) error {
	// function InitEnvironment has been init the path U.SingleUnpkg

	ssuPath, err := JudgeUnpack(md5, U)
	if err == nil {
		log.Info("[unpackPackage]find %s in ssu.conf", md5)
		log.Info("[unpackPackage]don't need to unpack ssu package:%s", U.SSUPackage)
		U.SSUFolder = SSUPath(ssuPath)
		log.Info("[unpackPackage]SSUFolder is %s", U.SSUFolder)
		return InitEnvironment(U, true)
	}
	//因为要解压,每个包解压存放的目录就以包的名字来命令
	U.SSUFolder = SSUPath(U.SSUPackage)
	log.Info("[unpackPackage]SSUFolder is %s", U.SSUFolder)
	if err := InitEnvironment(U, false); err != nil {
		return err
	}

	log.Info("[UnpackPackage]begin to unpack the package")
	logFile := filepath.Join(CurrentDirectory(), U.FolderPrefix, U.SSUFolder, "7z.log")
	if err := unpack(U.SSUPackage, U.SingleUnpkg, "7za", logFile); err != nil {
		return err
	}

	apps := GetApps(U.SingleUnpkg)
	for _, v := range apps {
		if err := EncFile(v, v+"_des"); err != nil {
			return err
		}
	}

	if err := WriteMd5ToConf(md5, U.SSUPackage, U); err != nil {
		return err
	}
	return nil
}

func UnpackPackage(md5 string, U *Update) error {
	if U.SSUType == PACKAGE_TYPE || U.SSUType == RESTORE_TYPE {
		return unpackPackage(md5, U)
	}
	return fmt.Errorf("[UnpackPackage]Package type %d is not support", U.SSUType)
}

//cfg is a config file,it should be a config file absolute path
func UnpackCfg(U *Update, cfg string) error {
	log.Info("[UnpackCfg]begin to unpack the config package")
	logFile := filepath.Join(CurrentDirectory(), "unpakccfg.log")
	return unpack(cfg, U.CfgPath, "7z", logFile)
}

//TODO pack the config file, not done yet
func PackCfg(U *Update, cfg string) error {
	log.Info("[PackCfg]begin to unpack the config package")
	logFile := filepath.Join(CurrentDirectory(), "pakccfg.log")
	return unpack(cfg, U.CfgPathTmp, "7z", logFile)
}

//TODO pack the config file, not done yet
func pack(packPath, destPath, unpackTool, logFile string) error {
	if runtime.GOOS == "windows" {
		unpackTool = filepath.Join(CurrentDirectory(), "tool", "7z.exe")
	}
	Args := []string{
		0: "a",
		1: "-p" + SSU_DEC_PASSWD_OLD,
		2: packPath,
		3: "-o" + destPath,
	}

	new := exec.Command(unpackTool, Args...)
	stdout, _ := new.StdoutPipe()
	if err := new.Start(); err != nil {
		return err
	}
	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Warn("[pack]unpack log has been lost")
	}
	if err := ioutil.WriteFile(logFile, data, 0664); err != nil {
		log.Warn("[pack]unpack log can't write it to logfile:", err)
	}
	if err := new.Wait(); err != nil {
		return err
	}
	log.Info("[pack]unpack success")
	return nil
}

func FreeUpdateDir() {

}

func FreeCfgDir() {

}

func GetApps(appPath string) (apps []string) {

	reg := regexp.MustCompile(`app[\d]`)
	files := FileList(appPath)
	for _, v := range files {
		//return nil means find the str
		if reg.FindAllString(v.Name(), -1) != nil {
			apps = append(apps, filepath.Join(appPath, v.Name()))
		}

	}
	log.Debug("[GetApps]Apps is %v", apps)
	return apps
}

func GetDesApps(DesAppPath string) (desApps []string) {
	reg := regexp.MustCompile(`app[\d]_des`)
	files := FileList(DesAppPath)
	for _, v := range files {
		//return nil means find the str
		if reg.FindAllString(v.Name(), -1) != nil {
			desApps = append(desApps, filepath.Join(DesAppPath, v.Name()))
		}

	}
	return desApps

}

func LoadAppData(AppPath string) {
	return
}

func PutDesApp(S *session, LocalFile, RemoteFile string) error {
	if !IsPathExist(LocalFile) {
		log.Error("[PutDesApp]%s don't exist", LocalFile)
		return fmt.Errorf("%s don't exist", LocalFile)
	}
	if err := DoCmd(S, CMD[PUT], RemoteFile); err != nil {
		log.Error("[PutDesApp]DoCmd fail, put %s fail,err msg:%s", RemoteFile, err)
		return fmt.Errorf("[PutDesApp]DoCmd fail, put %s fail,err msg:%s", RemoteFile, err)
	}
	file, err := os.Open(LocalFile)
	if err != nil {
		return err
	}

	defer file.Close()

	buf := make([]byte, 1038)
	bufRead := bufio.NewReader(file)
	var n int = 0
	for {
		n, err = bufRead.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if 0 == n {
			break
		}
		if err := S.WritePacket(buf[:n]); err != nil {
			log.Error("[PutDesApp]WritePacket error:%s", err)
			return fmt.Errorf("[PutDesApp]WritePacket error:%s", err)
		}
	}
	if err := DoCmd(S, CMD[PUTOVER], ""); err != nil {
		log.Error("[PutDesApp]DoCmd fail, PUTOVER fail,err msg:%s", err)
		return fmt.Errorf("[PutDesApp]DoCmd fail, PUTOVER fail,err msg:%s", err)
	}
	return nil
}

//如果desApps的路径包含有app就糟糕了　TODO: i will make it right later
func UpdateApps(S *session, U *Update, desApps []string) error {
	for _, desApp := range desApps {
		app := strings.TrimSuffix(desApp, "_des")
		appsh := strings.Replace(app, "app", "appsh", 1)
		log.Info("[UpdateApps]uploading %s to /stmp/app", desApp)
		if err := PutDesApp(S, desApp, "/stmp/app"); err != nil {
			return err
		}
		log.Info("[UpdateApps]upload %s to /stmp/app: success", desApp)

		log.Info("[UpdateApps]start to put %s to %s", appsh, U.ServerAppSh)
		if err := Put(S, appsh, U.ServerAppSh); err != nil {
			return err
		}
		log.Info("[UpdateApps]start to put %s to %s success", appsh, U.ServerAppSh)

		log.Info("[UpdateApps]start to exec %s", U.ServerAppSh)
		msg, err := Exec(S, U, U.ServerAppSh)
		if err != nil {
			log.Error("[UpdateApps] exec %s fail,get msg:%s,err msg:%s", U.ServerAppSh, msg, err)
			return fmt.Errorf("[UpdateApps] exec %s fail,get msg:%s,err msg:%s", U.ServerAppSh, msg, err)
		}
		log.Debug("[UpdateApps]exec %s retrun message:%s", U.ServerAppSh, msg)
	}
	return nil
}

func RestoreDefaultPriv() error {

	return nil
}

func UpdateSinglePacket(S *session, U *Update) error {
	if err := CheckUpdateCondition(S, U); err != nil {
		return err
	}
	log.Info("[UpdateSinglePacket]appre exec success")
	desApps := GetDesApps(U.SingleUnpkg)
	if err := UpdateApps(S, U, desApps); err != nil {
		return err
	}
	return nil
}

func CheckUpdateCondition(S *session, U *Update) error {
	log.Info("[CheckUpdateCondition]begin to check the update confition by appre.")
	if err := Put(S, filepath.Join(U.SingleUnpkg, "apppre"), U.ServerAppRe); err != nil {
		return err
	}
	if _, err := Exec(S, U, U.ServerAppRe); err != nil {
		return err
	}
	return nil
}

func NewUpdate() *Update {
	ssuInfo := make([]SSUSlice, 1)
	return &Update{SSU: &SSU{SSUInfo: ssuInfo}, Package: &Package{}, Unpack: &Unpack{}, Cfg: &Cfg{}}
}

func InitClient(appVersion string) *Update {
	U := NewUpdate()
	U.CurrentWorkFolder = CurrentDirectory()
	//U.FolderPrefix = RandomString(32)
	U.FolderPrefix = "update/ssu/"
	U.ssuConf = filepath.Join(U.CurrentWorkFolder, "update", "conf", "ssu.conf")
	U.appConf = filepath.Join(U.CurrentWorkFolder, "update", "conf", "app.conf")
	if IsArmChip(appVersion) {
		U.TempExecFile, U.TempRstFile = ARM_LINUX_BASIC[0], ARM_LINUX_BASIC[1]
		U.CustomErrFile, U.TempRetFile = ARM_LINUX_BASIC[2], ARM_LINUX_BASIC[3]
		U.LoginPwdFile, U.Compose = ARM_LINUX_BASIC[4], ARM_LINUX_BASIC[5]

		U.ServerAppRe, U.ServerAppSh = ARM_LINUX_UPDATE[0], ARM_LINUX_UPDATE[1]
		U.ServerCfgPre, U.ServerCfgSh = ARM_LINUX_UPDATE[2], ARM_LINUX_UPDATE[3]

		U.LocalBackSh = filepath.Join(U.CurrentWorkFolder, "update", "/arm_bin/bakcfgsh")
		U.LocalPreCfgSh = filepath.Join(U.CurrentWorkFolder, "update", "/arm_bin/prercovcfgsh")
		U.LocalCfgSh = filepath.Join(U.CurrentWorkFolder, "update", "/arm_bin/rcovcfgsh")
		U.LocalUpdHistory = filepath.Join(U.CurrentWorkFolder, "update", "/arm_bin/updhistory.sh")
		U.LocalUpdCheck = filepath.Join(U.CurrentWorkFolder, "update", "/arm_bin/updatercheck.sh")

		log.Info("[InitClient]The device is a arm platform,init arm info.")
		return U
	}

	U.TempExecFile, U.TempRstFile = X86_LINUX_BASIC[0], X86_LINUX_BASIC[1]
	U.CustomErrFile, U.TempRetFile = X86_LINUX_BASIC[2], X86_LINUX_BASIC[3]
	U.LoginPwdFile, U.Compose = X86_LINUX_BASIC[4], X86_LINUX_BASIC[5]

	U.ServerAppRe, U.ServerAppSh = X86_LINUX_UPDATE[0], X86_LINUX_UPDATE[1]
	U.ServerCfgPre, U.ServerCfgSh = X86_LINUX_UPDATE[2], X86_LINUX_UPDATE[3]

	U.LocalBackSh = filepath.Join(U.CurrentWorkFolder, "update", "/bin/bakcfgsh")
	U.LocalPreCfgSh = filepath.Join(U.CurrentWorkFolder, "update", "/bin/prercovcfgsh")
	U.LocalCfgSh = filepath.Join(U.CurrentWorkFolder, "update", "/bin/rcovcfgsh")
	U.LocalUpdHistory = filepath.Join(U.CurrentWorkFolder, "update", "/bin/updhistory.sh")
	U.LocalUpdCheck = filepath.Join(U.CurrentWorkFolder, "update", "/bin/updatercheck.sh")

	log.Info("[InitClient]The device is a x86 platform,init x86 info.")

	return U
}

func InitEnvironment(U *Update, flag bool) error {
	log.Info("[InitEnvironment]now init enviroment for update or restore")
	U.SingleUnpkg = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, U.SSUFolder, "/unpkg/")
	U.ComposeUnpkg = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, U.SSUFolder, "/compose_unpkg/")
	U.PkgTemp = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, U.SSUFolder, "/pkg_tmp/")
	U.Download = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, U.SSUFolder, "/download/")
	U.AutoBak = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, U.SSUFolder, "/autobak/")

	//如果没有从配置文件里找到已经解压了ssu包则要init
	if !flag {
		if err := InitDirectory(U.SingleUnpkg); err != nil {
			return err
		}
		if err := InitDirectory(U.ComposeUnpkg); err != nil {
			return err
		}
		if err := InitDirectory(U.PkgTemp); err != nil {
			return err
		}
		if err := InitDirectory(U.Download); err != nil {
			return err
		}
		if err := InitDirectory(U.AutoBak); err != nil {
			return err
		}
	}
	log.Warn("[InitEnvironment]U.singleUnpkg is %s", U.SingleUnpkg)

	return nil
}

func InitCfgEnvironment(U *Update) error {
	if U.RestoringFlag {
		return fmt.Errorf("it is restoring,now can't restore")
	}
	U.UpdatePath = filepath.Join(U.CurrentWorkFolder, U.FolderPrefix, "updater")
	U.CfgPath = filepath.Join(U.UpdatePath, "cfg")
	U.CfgPathTmp = filepath.Join(U.UpdatePath, "cfg_tmp")
	if err := InitDirectory(U.CfgPath); err != nil {
		return err
	}
	if err := InitDirectory(U.CfgPathTmp); err != nil {
		return err
	}
	return nil

}

//read file  from start to end
func ReadMd5FromPackage(ssuPath string, start, end int64) (string, error) {
	if start < 0 || end < 0 || start > end {
		log.Error("[ReadMd5FromPackage]params start or end is wrong,start:%d,end:%d", start, end)
		return "", fmt.Errorf("[ReadMd5FromPackage]params start or end is wrong,start:%d,end:%d", start, end)
	}
	file, err := os.Open(ssuPath)
	if err != nil {
		return "", fmt.Errorf("[ReadMd5FromPackage]%s", err)
	}
	length := end - start
	buf := make([]byte, length)
	_, err = file.Seek(start, 1)
	n, err := io.ReadFull(file, buf)
	if err != nil && int64(n) != length {
		return "", fmt.Errorf("[ReadMd5FromPackage]%s", err)
	}
	return string(buf), nil
}

//用于检查升级包是否为组合升级包，目前AD不是组合的
//TODO:when encounter error,I think it should print error md5 and correct md5
func ComposePackageMd5(ssuPath string) (string, error) {
	ssuMd5, err := ReadMd5FromPackage(ssuPath, 8, 40)
	if err != nil {
		return "", err
	}
	correctMd5 := Md5Sum(ssuPath, 48)
	if ssuMd5 == correctMd5 {
		return ssuMd5, nil
	}
	log.Warn("[ComposePackageMd5]compose package md5 don't match\n\tcorrectMd5:%s\n\terrorMd5:%s", correctMd5, ssuMd5)
	return "", fmt.Errorf("[ComposePackageMd5]compose package md5 don't match\n\tcorrectMd5:%s\n\terrorMd5:%s", correctMd5, ssuMd5)

}

//用于检查升级包是否为组合升级包，目前AD不是组合的
func ComposePackage(ssuPath string) (string, bool) {
	if md5, err := ComposePackageMd5(ssuPath); err == nil {
		if filepath.Ext(ssuPath) == ".cssu" {
			return md5, true
		} else {
			log.Error("[ComposePackage]The package %s is a cssu file,but not have a .cssu extname.", ssuPath)
			return "", false
		}
	}
	return "", false

}

//如果在配置文件里找到此ssu包已经解压了，就不用再解压了
//如果没有找到就准备写入配置文件(当然要解压好再写入)
//如果超过了限制的就把最早解压的包删掉，也从配置文件里删掉
func WriteMd5ToConf(md5, ssu string, u *Update) error {
	value, err := ReadValueFromConf(u.appConf, "ssu", "ssunum", appMutex)
	if err != nil {
		return err
	}
	num, err1 := strconv.Atoi(value)
	log.Info("[WriteMd5ToConf]app.conf show ssunum is %d", num)
	if err1 != nil {
		return err1
	}
	keys, err2 := FindAllKeyValue(u.appConf, "ssu", appMutex)
	if err2 != nil {
		return err2
	}

	//如果大于配置文件里设置的就删除第一个key,
	//再在尾部插入
	if len(keys) > num {
		//TODO not done yet
	}

	if err := WriteMsgToConf(u.ssuConf, "ssu", md5, ssu, ssuMutex); err != nil {
		return fmt.Errorf("[WriteMd5ToConf]wirte md5 to conf:%s, fail:%s", u.ssuConf, err)
	}
	return nil
}

func JudgeUnpack(md5 string, u *Update) (string, error) {
	hash, err := FindAllKeyValue(u.ssuConf, "ssu", ssuMutex)
	if err != nil {
		return "", err
	}
	//如果在配置文件里找到了与md5相同的key那么就表示已经解压过此ssu包了
	value, err1 := CompareKeyFromMap(hash, md5)
	if err1 != nil {
		return "", err1
	}

	return value, nil
}

//TODO: not done yet
//用于检查升级包是否为组合升级包，目前AD不是组合的
func InitComposePackageArr(ssuPath string) []string {
	return nil
}

func SinglePackageMd5(ssuPath string) (string, error) {
	ssuMd5, err := ReadMd5FromPackage(ssuPath, 0, 32)
	if err != nil {
		return "", err
	}
	correctMd5 := Md5Sum(ssuPath, 33)
	if ssuMd5 == correctMd5 {
		return ssuMd5, nil
	}
	log.Error("[SinglePackageMd5]single package md5 don't match\n\t\tcorrectMd5:%s\n\t\terrorMd5:%s", correctMd5, ssuMd5)
	return "", fmt.Errorf("[SinglePackageMd5]single package md5 don't match\n\t\tcorrectMd5:%s\n\t\terrorMd5:%s", correctMd5, ssuMd5)

}

func PrepareUpgrade(S *session, U *Update) (string, error) {
	log.Info("[PrepareUpgrade]init to upgrade or restore  the package:%s", U.SSUPackage)
	if U.UpdatingFlag && (time.Now().Sub(U.UpdateTime) < UPD_TIMEOUT*time.Second) {
		return "", fmt.Errorf("[PrepareUpgrade]now update the package:%s,begin at %v\n ....", U.SSUPackage, U.UpdateTime)
	}
	if err := FtpDownloadSSUPackage(U.SSUPackage, "admin", "admin"); err != nil {
		return "", err
	}
	if !IsPathExist(U.SSUPackage) {
		return "", fmt.Errorf("can't find the SSU package,please check it\n")
	}

	if md5, err := ComposePackage(U.SSUPackage); err == true {
		InitComposePackageArr(U.SSUPackage) //TODO: not done yet
		return md5, nil
	} else if md5, err := SinglePackageMd5(U.SSUPackage); err == nil {
		var ssuInfo SSUSlice
		ssuInfo.SSUPacket = U.SSUPackage
		ssuInfo.SSUType = PACKAGE_TYPE
		U.SSUInfo = append(U.SSUInfo, ssuInfo)

		U.SSUType = PACKAGE_TYPE //TODO: it will be abandoned
		log.Info("[PrepareUpgrade]The package %s is a valid single package", U.SSUPackage)
		return md5, nil
	}
	return "", fmt.Errorf("[PrepareUpgrade]The package %s is not a valid package,please check first. if your use a ftp path,please download it to local and try again.\n", U.SSUPackage)

}
