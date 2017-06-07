package update

import (
	"net"
	"time"
)

type PeerInfo struct {
	SerVersion string //Updateme version
	AppVersion string //AD version
}

type SSUSlice struct {
	SSUPacket string
	SSUType   int
}

type SSU struct {
	Md5sum     string //Same Version SSU packet has been unpack or not
	SSUPackage string //SSU packet name
	SSUType    int    /*PACKAGE_TYPE = 1 RESTORE_TYPE = 2 EXECUTE_TYPE  = 3 AUTOBAK_NUMS  = 10 */
	SSUInfo    []SSUSlice
}

type Unpack struct {
	FolderPrefix      string //random string
	CurrentWorkFolder string
	SSUFolder         string //解压之后的ssu目录

	LocalBackSh     string
	LocalPreCfgSh   string
	LocalCfgSh      string
	LocalUpdHistory string
	LocalUpdCheck   string
	ServerAppRe     string
	ServerAppSh     string
	ServerCfgPre    string
	ServerCfgSh     string
	TempExecFile    string
	TempRstFile     string
	TempRetFile     string
	CustomErrFile   string
	LoginPwdFile    string
	Compose         string

	SingleUnpkg  string
	ComposeUnpkg string
	PkgTemp      string
	Download     string
	AutoBak      string

	UpdatePath string
}

type Package struct {
	UpdatingFlag  bool      //updating or not
	UpdateTime    time.Time //when to update
	RestoringFlag bool
	ssuConf       string
	appConf       string
}

type Cfg struct {
	CfgPath    string
	CfgPathTmp string
}

type session struct {
	Conn net.Conn
	*PeerInfo
	*SecData
}

type Update struct {
	*SSU
	*Package
	*Unpack
	*Cfg
}

type params struct {
	param1 string
	param2 string
}

type SecData struct {
	length uint16
	typ    byte
	data   []byte
}

func (Sec *SecData) DataFrame() bool { return Sec.typ == DATAFRAME }
func (Sec *SecData) CmdFrame() bool  { return Sec.typ == CMDFRAME }
