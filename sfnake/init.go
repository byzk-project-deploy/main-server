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

func (s *SnowFlake) GetID() (uint64, error) {
	return s.sFlake.NextID()
}

func (s SnowFlake) GetIdStrUnwrap() string {
	id, err := s.GetID()
	if err != nil {
		panic(err.Error())
	}
	return strconv.FormatUint(id, 10)
}
