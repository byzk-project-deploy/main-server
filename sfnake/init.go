package sfnake

import (
	"github.com/sony/sonyflake"
	"strconv"
)

var (
	SFlake *SnowFlake
)

// SnowFlake SnowFlake算法结构体
type SnowFlake struct {
	sFlake *sonyflake.Sonyflake
}

func init() {
	SFlake = NewSnowFlake()
}

func NewSnowFlake() *SnowFlake {
	st := sonyflake.Settings{}
	// machineID是个回调函数
	//st.MachineID = getMachineID
	return &SnowFlake{
		sFlake: sonyflake.NewSonyflake(st),
	}
}

func GetID() (uint64, error) {
	return SFlake.sFlake.NextID()
}

func GetIdStr() (string, error) {
	id, err := GetID()
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(id, 10), nil
}

func GetIdStrUnwrap() string {
	str, err := GetIdStr()
	if err != nil {
		panic(err)
	}
	return str
}
