package repueng

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type RepEng struct{}

var program = "python3"
var arg0 = "-c"

func (r RepEng) Calculate(penalties []int, cur_view, new_view, t_left, t_all uint64) (int, error) {

	allPastPenalties := strings.Trim(strings.Replace(fmt.Sprint(penalties), " ", ",", -1), " ")

	para := fmt.Sprintf("%s, %d, %d, %d, %d", allPastPenalties, cur_view, new_view, t_left, t_all)

	arg1 := fmt.Sprintf("import repu; print(repu.calculator(%s))", para)
	cmd := exec.Command(program, arg0, arg1)
	//fmt.Println("command args:", cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1, err
	}

	if p, err := strconv.Atoi(strings.TrimSuffix(string(out), "\n")); err == nil {
		return p, nil
	} else {
		return -1, err
	}
}
