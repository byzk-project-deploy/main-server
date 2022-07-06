package sfnake

import (
	"fmt"
	"testing"
	"time"
)

func TestSnowFlake_GetID(t *testing.T) {
	//id, err := SFlake.GetID()
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(id)
	fmt.Println(time.Now().UnixNano())
}