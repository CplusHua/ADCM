package update

import (
	"bufio"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/go-ini/ini"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var log *logs.BeeLogger

func init() {
	log = logs.NewLogger(10)
	log.EnableFuncCallDepth(true)
	log.SetLogFuncCallDepth(5)
}

//if S.data contains string "result:1",it means command executed fail by AD
func IsResultOK(data string) bool {
	return !strings.Contains(data, "result:1")
}

func IsGetOver(data string) bool {
	return strings.Contains(data, CMD[GETOVER])
}

func QueryVersion(data string) bool {
	return strings.Contains(data, "result:7629414")
}

// Get the Server Version(updateme program version)
func VersionResult(S *Session) {
	reg := regexp.MustCompile(`version:[\d]+`)
	str := reg.FindAllString(string(S.data), -1)[0]
	S.SerVersion = strings.Split(str, ":")[1]
}

//Get AD Version
func GetAppVersion(S *Session, appVersion string) {
	log.Debug("[GetAppVersion]appversion:\n%s", appVersion)
	reg := regexp.MustCompile(`[\w]+-[\w]+\.[\w]+`)
	str := reg.FindAllString(appVersion, -1)[0]
	S.AppVersion = strings.Split(str, "-")[1]
	log.Info("[GetAppVersion]The first line of appversion of the current device is:", S.AppVersion)
}

func IsArmChip(appVersion string) bool {
	str := strings.ToLower(appVersion)
	if strings.Contains(str, "-ac-") || strings.Contains(str, "sinfor-m") || strings.Contains(str, "-ad-") {
		return false
	}
	if strings.Contains(str, "-bm-") || strings.Contains(str, "-bc-") || strings.Contains(str, "-iam") {
		return false
	}

	if strings.Contains(str, "-nag") || strings.Contains(str, "sinfor--") || strings.Contains(str, "sangfor--") {
		return false
	}
	if strings.Contains(str, "ar") || strings.Contains(str, "xp") || strings.Contains(str, "plus") {
		return true
	}
	return false //default is not arm chip
}

//Get file from Server, and download,write it to the LocalFile
func Get(S *Session, RemoteFile, LocalFile string) (string, error) {
	if err := DoCmd(S, CMD[GET], RemoteFile); err != nil {
		return "", fmt.Errorf("[Get]the server can't send the file:%s.check the file exists,err msg:%s", RemoteFile, err)
	}

	if LocalFile != "" {
		file, err := os.OpenFile(LocalFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return "", fmt.Errorf("[Get]OpenFile %s fail:%s", LocalFile, err)
		}
		defer file.Close()
		bufW := bufio.NewWriter(file)
		if err := S.ReadPacket(); err != nil {
			log.Error("[Get]Get data error:%s", err)
			log.Error("[Get]Get data is :%s", string(S.data))
			return "", fmt.Errorf("[Get]Get data error:%s", err)
		}
		for S.typ == DATAFRAME {
			if byteWN, err := bufW.Write(S.data); err != nil || byteWN != len(S.data) {
				return "", fmt.Errorf("[Get]write error: %s", err)
			}
			if err := S.ReadPacket(); err != nil {
				log.Error("[Get]ReadPacket error:%s", err)
				return "", fmt.Errorf("[Get]ReadPacket error:%s", err)
			}
		}
		if err := bufW.Flush(); err != nil {
			return "", fmt.Errorf("[Get]Flush data error:%s", err)
		}
		//when readpacket type is not DATAFRAME,it must be CMDFRAME
		//So,just it IsGetOver use S.data
		if !IsGetOver(string(S.data)) {
			log.Debug("[Get]Get all data:\n%s", string(S.data))
			return "", fmt.Errorf("[Get]Not found getover flag while get the file:%s\n", RemoteFile)
		}

	} else {
		if err := S.ReadPacket(); err != nil {
			log.Error("[Get]Get data error:%s", err)
			log.Error("[Get]Get data is :%s", string(S.data))
			return "", fmt.Errorf("[Get]Get data error:%s", err)
		}
		var allData []byte
		for S.typ == DATAFRAME {
			allData = append(allData, S.data...)
			if err := S.ReadPacket(); err != nil {
				log.Error("[Get]ReadPacket error:%s", err)
				return "", fmt.Errorf("[Get]ReadPacket error:%s", err)
			}
		}
		//when readpacket type is not DATAFRAME,it must be CMDFRAME
		//So,just it IsGetOver use S.data
		if !IsGetOver(string(S.data)) {
			log.Debug("[Get]Get all data:\n%s", string(S.data))
			return "", fmt.Errorf("[Get]Not found getover flag while get the file:%s\n", RemoteFile)
		}

		return string(allData), nil
	}

	return "", nil
}

//return true,it mean command execute success by peer
//return false, it mean command execute fail by peer
func DoCmd(S *Session, cmdType, params string) error {
	cmdStr, err := MakeCmdPacket(cmdType, params)
	if err != nil {
		return fmt.Errorf("[DoCmd]MakeCmdPacket error:%v", err)

	}
	err = S.WritePacket(cmdStr)
	if err != nil {
		log.Error("[DoCmd]WritePacket error:%s", err)
		return fmt.Errorf("[DoCmd]WritePacket error:%s", err)
	}
	err = S.ReadPacket()
	if err != nil {
		log.Error("[DoCmd]ReadPacket error:%s", err)
		return fmt.Errorf("[DoCmd]ReadPacket error:%s", err)
	}
	if IsResultOK(string(S.data)) {
		log.Info("[DoCmd]Do Command %s return Result Ok", cmdType)
		return nil
	} else {
		log.Error("[Docmd]Do command %s result is not ok:%s", cmdType, string(S.data))
		return fmt.Errorf("[Docmd]Do command %s result is not ok:%s", cmdType, string(S.data))
	}

}

func Exec(S *Session, U *Update, Command string) (string, error) {
	doRet := DoCmd(S, CMD[EXEC], Command)
	getReturn, err := Get(S, U.TempRetFile, "")
	if err != nil {
		log.Error("[Exec]Get %s fail:%s", U.TempRetFile, err)
		return "", fmt.Errorf("[Exec]Get %s fail:%s", U.TempRetFile, err)
	}
	getResult, err1 := Get(S, U.TempRstFile, "")
	if err1 != nil {
		log.Error("[Exec]Get %s fail:%s", U.TempRstFile, err1)
		return getResult, fmt.Errorf("[Exec]Get %s fail:%s", U.TempRstFile, err1)
	}
	//TODO I should write a delete  white space by myself
	if strings.TrimSpace(getReturn) != "0" || doRet != nil {
		log.Error("[Exec]Exec %s fail:%s", Command, doRet)
		log.Debug("[Exec]return msg:%s", getReturn)
		return getResult, fmt.Errorf("[Exec]Exec %s fail:%s", Command, doRet)
	}
	return getResult, nil
}

func Put(S *Session, LocalFile, RemoteFile string) error {
	log.Info("[Put]put %s to %s is starting", LocalFile, RemoteFile)
	if !IsPathExist(LocalFile) {
		log.Error("[Put] %s don't exist", LocalFile)
		return fmt.Errorf("%s don't exist", LocalFile)
	}
	if err := DoCmd(S, CMD[PUT], RemoteFile); err != nil {
		log.Error("[Put]put %s fail,err msg is %s", RemoteFile, err)
		return fmt.Errorf("[Put]DoCmd fail, put %s fail,err msg is: %s\n", RemoteFile, err)
	}
	file, err := os.Open(LocalFile)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, MAX_DATA_LEN)
	bufRead := bufio.NewReader(file)

	for {
		n, err1 := bufRead.Read(buf)
		if err1 != nil && err1 != io.EOF {
			log.Error("[Put]read file %s error:%s", LocalFile, err1)
			return fmt.Errorf("[Put]read file %s error:%s", LocalFile, err1)
		}
		if 0 == n {
			break
		}
		data, errData := MakeDataPacket(buf[:n])
		if errData != nil {
			log.Error("[Put]MakeDataPacket error:%s", errData)
			return fmt.Errorf("[Put]MakeDataPacket error:%s", errData)
		}
		if err := S.WritePacket(data); err != nil {
			log.Error("[Put]WritePacket error:%s", err)
			return fmt.Errorf("[Put]WritePacket error:%s", err)
		}

	}
	if DoCmd(S, CMD[PUTOVER], "") != nil {
		log.Error("[Put]DoCmd fail, PUTOVER fail\n")
		return fmt.Errorf("DoCmd fail, PUTOVER fail\n")
	}
	log.Info("[Put]put %s to %s success", LocalFile, RemoteFile)
	return nil
}

func PutFile(ip, port, passwd, LocalFile, RemoteFile string) error {
	if !IsPathExist(LocalFile) {
		return fmt.Errorf("[PutFile]%s don't exist", LocalFile)
	}
	S, loginErr := Login(ip, port, passwd)
	if loginErr != nil {
		return loginErr
	}
	defer Logout(S)
	return Put(S, LocalFile, RemoteFile)
}

func GetFile(ip, passwd, port, LocalFile, RemoteFile string) error {
	S, loginErr := Login(ip, port, passwd)
	if loginErr != nil {
		return loginErr
	}
	defer Logout(S)
	_, err := Get(S, RemoteFile, LocalFile)
	return err
}

func NewSession(conn net.Conn) *Session {
	return &Session{Conn: conn, PeerInfo: &PeerInfo{}, SecData: &SecData{}}
}

func Login(ip, port, passwd string) (*Session, error) {
	conn, err := net.Dial("tcp4", ip+":"+port)
	if err != nil {
		return nil, err
	}
	S := NewSession(conn)
	if DoCmd(S, CMD[LOGIN], passwd) != nil {
		return nil, fmt.Errorf("[Login]Login fail,please check the passwd\n")
	}
	if QueryVersion(string(S.data)) {
		if DoCmd(S, CMD[VERSION], "") != nil {
			return nil, fmt.Errorf("[Login]DoCmd %s fail\n", CMD[VERSION])
		}
		VersionResult(S)
	} else {
		S.SerVersion = "300"
		log.Warn("[Login]server version lower than v300. nothing to do\n")
	}
	log.Info("[Login]loging %s:%s success", ip, port)
	return S, nil
}

func Logout(S *Session) error {
	return S.Conn.Close()
}

func UpgradeCheck(S *Session, U *Update) error {
	msg, err := Exec(S, U, "ls "+UPDATE_CHECK_SCRIPT)
	if err != nil {
		log.Warn("[UpgradeCheck]exec ls %s fail,msg:%s\n error msg:%s", UPDATE_CHECK_SCRIPT, msg, err)
		log.Info("[UpgradeCheck]begin to put %s to server %s", U.LocalUpdCheck, UPDATE_CHECK_SCRIPT)
		if err := Put(S, U.LocalUpdCheck, UPDATE_CHECK_SCRIPT); err != nil {
			log.Error("[UpgradeCheck]Put file %s to server %s fail,the error msg is:%s", U.LocalUpdCheck, UPDATE_CHECK_SCRIPT, err)
			return fmt.Errorf("Put file %s to server %s fail,the error msg is:%s", U.LocalUpdCheck, UPDATE_CHECK_SCRIPT, err)
		}
	}
	//execute /usr/sbin/updatercheck.sh, check it pass or fail
	msgVersion, resultVersion := Exec(S, U, UPDATE_CHECK_SCRIPT)
	if resultVersion != nil {
		log.Error("[Upgradecheck] exec %s fail,return msg:%s,error msg:%s", UPDATE_CHECK_SCRIPT, msgVersion, resultVersion)
		return fmt.Errorf("[Upgradecheck] exec %s fail,return msg:%s,error msg:%s", UPDATE_CHECK_SCRIPT, msgVersion, resultVersion)
	}

	//check upgrade sn valid or invalid
	msgSn, resultSn := Exec(S, U, CHECK_UPGRADE_SN)
	if resultSn != nil {
		log.Error("[Upgradecheck] exec %s fail,return msg:%s,error msg:%s", CHECK_UPGRADE_SN, msgSn, resultSn)
		return fmt.Errorf("[Upgradecheck] exec %s fail,return msg:%s,error msg:%s", CHECK_UPGRADE_SN, msgSn, resultSn)
	}
	return nil
}

//TODO only support to update single package right now
func ThreadUpdateAllPackages(S *Session, U *Update) error {
	switch U.SSUType {
	case PACKAGE_TYPE:
		if err := UpdateSinglePacket(S, U); err != nil {
			return err
		}
	case RESTORE_TYPE:
		if err := RestoreDefaultPriv(); err != nil {
			return err
		}
	case EXECUTE_TYPE:
		if err := Put(S, U.SSUPackage, U.Compose); err != nil {
			return err
		}
		if _, err := Exec(S, U, U.Compose); err != nil {
			return err
		}
	default:
		log.Error("[ThreadUpdateAllPackages]unknown type packet:%d", U.SSUType)
		return fmt.Errorf("[ThreadUpdateAllPackages]unknown type packet:%d", U.SSUType)
	}
	return nil
}

func UpdateUpgradeHistory(S *Session, U *Update) error {
	log.Info("[UpdateUpgradeHistory]begin to update Upgrade History")
	msg, err := Exec(S, U, "ls "+UPDHISTORY_SCRIPT)
	if err != nil {
		log.Warn("[UpdateUpgradeHistory] exec ls %s fail,msg:%s,err:%s", UPDHISTORY_SCRIPT, msg, err)
		log.Warn("[UpdateUpgradeHistory] begin to put  %s to server", UPDHISTORY_SCRIPT)
		if err := Put(S, U.LocalUpdHistory, UPDHISTORY_SCRIPT); err != nil {
			return err
		}
		if msg, err := Exec(S, U, "sync"); err != nil {
			log.Error("[UpdateUpgradeHistroy]exec sync error:%s,msg:%s", err, msg)
			return fmt.Errorf("[UpdateUpgradeHistroy]exec sync error:%s,msg:%s", err, msg)
		}
	}
	if msg, err := Exec(S, U, UPDHISTORY_SCRIPT+" "+U.SSUPackage); err != nil {
		log.Error("[UpdateUpgradeHistroy]exec %s error:%s,msg:%s", UPDHISTORY_SCRIPT, err, msg)
		return fmt.Errorf("[UpdateUpgradeHistroy]exec %s error:%s,msg:%s", UPDHISTORY_SCRIPT, err, msg)
	}
	log.Info("[UpdateUpgradeHistory]update Upgrade History success")
	return nil
}

//TODO: ini format file
//TODO: now
func ConfirmRebootDevice(S *Session, U *Update) error {
	log.Info("[ConfirmRebootDevice]begin to Confirm Reboot Device")
	cfg, err := ini.Load(filepath.Join(U.SingleUnpkg, "package.conf"))
	if err != nil {
		return fmt.Errorf("[ConfirmRebootDevice]Load package.conf fail:%s", err)
	}
	value := cfg.Section("restart").Key("needrestart").String()

	if strings.ToLower(value) == "yes" {
		log.Debug("[ConfirmRebootDevice] need to reboot")
		if msg, err := Exec(S, U, "reboot"); err != nil {
			log.Error("[ConfirmRebootDevice]exec reboot error:%s,msg:%s", err, msg)
			return fmt.Errorf("[ConfirmRebootDevice]exec reboot error:%s,msg:%s", err, msg)
		}
		log.Info("[ConfirmRebootDevice]Confirm Reboot Device success")
	}

	log.Debug("[ConfirmRebootDevice]don't need to reboot")
	return nil
}

func Upgrade(ip, port, password, ssu string) error {

	S, err := Login(ip, port, password)
	if err != nil {
		return err
	}
	var appVersion string
	appVersion, err = Get(S, APPVERSION_FILE, "")
	if err != nil {
		return err
	}
	GetAppVersion(S, appVersion)

	U := InitClient(appVersion)
	U.SSUPackage = ssu
	if err := UpgradeCheck(S, U); err != nil {
		return err
	}

	if err := PrepareUpgrade(S, U); err != nil {
		return err
	}

	if err := UnpackPackage(U); err != nil {
		return err
	}
	apps := GetApps(U.SingleUnpkg)
	for _, v := range apps {
		if err := EncFile(v, v+"_des"); err != nil {
			return err
		}
	}

	if err := ThreadUpdateAllPackages(S, U); err != nil {
		return err
	}
	if err := UpdateUpgradeHistory(S, U); err != nil {
		return err
	}
	if err := ConfirmRebootDevice(S, U); err != nil {
		return err
	}

	defer FreeUpdateDir()
	defer FreeCfgDir()
	defer Logout(S)

	log.Info("[Upgrade]Upgrade %s:%s sucess", ip, port)
	return nil
}

func ThreadUpgrade(ip []string, port []string, passwd []string, ssu []string) {

}
