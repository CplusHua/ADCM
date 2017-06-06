package update

import (
	"fmt"
	"io"
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

type Session struct {
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

//read data from peer and decrypt data, and return data
func (S *Session) ReadPacket() error {
	//step 1: 分配frame长度的大小的空间
	frameHeaderBuf := make([]byte, FRAME_HEADER_LEN)
	var n int
	var err error
	var realNeed int = 0
	//step 2:　读取frame长度大小的数据
	for {
		n, err = S.Conn.Read(frameHeaderBuf[realNeed:])
		if err != nil && err != io.EOF {
			log.Error("[ReadPacket]Read Frame error:%s", err)
			return fmt.Errorf("[ReadPacket]Read Frame error:%s", err)
		}
		realNeed = realNeed + n
		if realNeed == FRAME_HEADER_LEN || 0 == n {
			realNeed = 0
			break
		}
	}

	frameHeader := NewLEStream(frameHeaderBuf)
	frameFlag, errFlag := frameHeader.ReadUint16()
	if errFlag != nil {

		log.Error("[ReadPacket]read frame flag fail:%s", errFlag)
		return fmt.Errorf("[ReadPacket]frame flag is wrong:0x%x", frameFlag)
	}
	secDataLen, errDataLen := frameHeader.ReadUint16()
	if errDataLen != nil {
		log.Error("[ReadPacket]read frame secDataLen fail:%s", errDataLen)
		return errDataLen
	}
	if frameFlag != FRAMEFLAG {
		log.Error("[ReadPacket]frame flag is wrong:0x%x", frameFlag)
		return fmt.Errorf("[ReadPacket]frame flag is wrong:0x%x", frameFlag)
	}

	if secDataLen > MAX_FRAME_LEN {
		log.Error("[ReadPacket]SecDataLen wrong:0x%x", secDataLen)
		return fmt.Errorf("[ReadPacket]SecDataLen wrong:0x%x", secDataLen)
	}
	//step 3: 分配加了密的sec Data的长度的空间
	encSecData := make([]byte, secDataLen)

	for {
		n, err = S.Conn.Read(encSecData[realNeed:])
		if err != nil && err != io.EOF {
			return fmt.Errorf("[Readpacket] read Sec Data error:", err)
		}
		realNeed = realNeed + n
		if realNeed == int(secDataLen) || n == 0 {
			realNeed = 0
			break
		}

	}

	var decSecData []byte
	//step 4: 由于暂时没法知道解密之后的数据是多大，所以直接先分配最大的
	//TODO:   当然是可以通过EncLen这个函数反过来推知，暂时不做
	outSecData := make([]byte, MAX_DATA_LEN)
	decSecData, err = Decrypt(encSecData, outSecData)
	if err != nil {
		log.Error("[ReadPacket]dec sec data error:%s", err)
		return fmt.Errorf("[ReadPacket]dec sec data error:%s", err)
	}

	secDataHeader := NewLEStream(decSecData)
	secDataFlag, errSecDataFlag := secDataHeader.ReadUint16()
	if errSecDataFlag != nil {
		log.Error("[ReadPacket]Read Sec Data Flag error:%s", errSecDataFlag)
		return fmt.Errorf("[ReadPacket]Read Sec Data Flag error:%s", errSecDataFlag)
	}
	if secDataFlag != FRAMEFLAG {
		log.Error("[ReadPacket]Sec Data Flag wrong:0x%x", secDataFlag)
		return fmt.Errorf("[ReadPacket]Sec Data Flag wrong:0x%x", secDataFlag)
	}
	dataLen, errSecDataLen := secDataHeader.ReadUint16()
	if errSecDataLen != nil {
		log.Error("[ReadPacket]Read Sec Data Len error:%s", errSecDataLen)
		return fmt.Errorf("[ReadPacket]Read Sec Data Len error:%s", errSecDataLen)
	}
	secDataType, errSecDataType := secDataHeader.ReadByte()
	if errSecDataType != nil {
		log.Error("[ReadPacket]Read Sec Data Type error:%s", errSecDataType)
		return fmt.Errorf("[ReadPacket]Read Sec Data Type error:%s", errSecDataType)
	}

	if secDataType != CMDFRAME && secDataType != DATAFRAME {
		log.Error("[ReadPacket]Sec Data Type wrong:%d", secDataType)
		return fmt.Errorf("[ReadPacket]Sec Data Type wrong:%d", secDataType)
	}

	realDataLen := uint16(len(decSecData[secDataHeader.Pos():]))
	if dataLen != realDataLen {
		log.Error("[ReadPacket]Read Sec Data len %d is not equal need Read Sec Data len %d", realDataLen, dataLen)
		return fmt.Errorf("[ReadPacket]Read Sec Data len %d is not equal need Read Sec Data len %d", realDataLen, dataLen)
	}

	S.typ = secDataType
	S.length = secDataLen
	S.data = secDataHeader.DataSelect(secDataHeader.Pos(), secDataHeader.Size())
	return nil
}

//just send data to peer
func (S *Session) WritePacket(data []byte) error {
	_, err := S.Conn.Write(data)
	return err
}
