package update

/* base frame
* a frame begin with "0xDB0xF3".....
 */

const (
	FRAME_HEADER_LEN   = 4      //a frame header is 4 bytes
	SECDATA_HEADER_LEN = 5      //a secData header len
	FRAMEFLAG          = 0xf3db //a frame is started with "0xf3db"
	MAX_DATA_LEN       = 1024   // the max frame data length
	MAX_FRAME_LEN      = 1080 + FRAME_HEADER_LEN
	CMDFRAME           = 0 //command frame flag
	DATAFRAME          = 1 //data frame flag

	// used by cmd data.
	LOGIN = iota
	EXEC  //exec cmd,be care!!!!!
	GET
	GETOVER
	PUT
	PUTOVER
	VERSION
	MaxCmdLen
)

var CMD = [MaxCmdLen]string{
	LOGIN:   "login",
	EXEC:    "excute",
	GET:     "get",
	GETOVER: "getover",
	PUT:     "put",
	PUTOVER: "putover",
	VERSION: "version",
}

type SecData struct {
	length uint16
	typ    byte
	data   []byte
}

func (Sec *SecData) DataFrame() bool { return Sec.typ == DATAFRAME }
func (Sec *SecData) CmdFrame() bool  { return Sec.typ == CMDFRAME }

type params struct {
	param1 string
	param2 string
}

//命令格式组合
func JoinCmd(cmd string, params []params) []byte {
	var b []byte
	b = append(b, []byte(cmd)...)
	b = append(b, []byte("\n")...)

	for _, v := range params {
		b = append(b, []byte(v.param1)...)
		b = append(b, []byte(":")...)
		b = append(b, []byte(v.param2)...)
		b = append(b, []byte("\n")...)
	}
	length := len(b)
	return b[:length]
}

func MakeCmdStr(cmdType, command string) []byte {
	switch cmdType {
	case CMD[LOGIN]:
		return JoinCmd(CMD[LOGIN], []params{{"passwd", command}, {"flage", "HandleVersion"}})
	case CMD[EXEC]:
		return JoinCmd(CMD[EXEC], []params{{"cmd", command}})
	case CMD[GET]:
		return JoinCmd(CMD[GET], []params{{"file", command}})
	case CMD[PUT]:
		return JoinCmd(CMD[PUT], []params{{"file", command}})
	case CMD[PUTOVER]:
		return JoinCmd(CMD[PUTOVER], []params{})
	case CMD[VERSION]:
		return JoinCmd(CMD[VERSION], []params{{"value", "1280"}})
	default:
		return nil
	}

}

func MakeCmdPacket(cmdType string, params string) ([]byte, error) {
	cmdByte := MakeCmdStr(cmdType, params)
	return BuildFrame(CMDFRAME, cmdByte)
}

func MakeDataPacket(content []byte) ([]byte, error) {
	return BuildFrame(DATAFRAME, content)
}

/*        1 byte      1 byte   　2 byte    1byte       1 byte      2byte  1 byte  less than 1024 byte
 * 格式:[FRAMEFLAG0][FRAMEFLAG0][length]([FRAMEFLAG0][FRAMEFLAG0][length][flag][data........])
 *      前两2个是协议开头标志　　后面数据字节数　　　括号里是加密的数据　flag表示是数据还是命令　data为真实数据
 */
func BuildFrame(flag byte, content []byte) (buf []byte, err error) {
	contentLength := len(content)
	secBuff := make([]byte, contentLength+SECDATA_HEADER_LEN)
	sec := NewLEStream(secBuff)
	sec.WriteUint16(FRAMEFLAG)
	sec.WriteUint16(uint16(contentLength))
	sec.WriteByte(flag)
	err = sec.WriteBuff(content)
	if err != nil {
		return nil, err
	}

	secLength := EncLen(sec.pos)
	frameBuff := make([]byte, secLength+FRAME_HEADER_LEN)
	frame := NewLEStream(frameBuff)
	frame.WriteUint16(FRAMEFLAG)
	frame.WriteUint16(uint16(secLength))

	tempBuff := make([]byte, secLength+FRAME_HEADER_LEN)
	//function Encrypt will combine secData and FrameData

	buf, err = Encrypt(sec.buff[:sec.pos], tempBuff)
	if err != nil {
		return nil, err
	}
	err = frame.WriteBuff(buf)
	if err != nil {
		return nil, err
	}
	return frame.buff[:frame.pos], nil
}
