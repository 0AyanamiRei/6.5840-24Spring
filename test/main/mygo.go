package main

import "fmt"

type Follower struct {
	Me  int
	Log []int
}

type Leader struct {
	NextIndex []int
	Log       []int
}

// 从st的位置开始追加entries
func AppednLog(f *Follower, st int, entries []int) {
	f.Log = append(f.Log[:st], entries...)
}

func nextindex(l *Leader, f *Follower) {
	//fmt.Printf("l's log:%v\nf's log:%v\n", l.Log, f.Log)

	var prevIndex, prevTerm int
	var XTerm, XIndex, XLen int
	var Success bool

	for {
		prevIndex = l.NextIndex[f.Me] - 1
		prevTerm = l.Log[prevIndex]
		//fmt.Printf("l->s%d 一致性检查:(index,term)=(%d,%d)\n",
		//	f.Me, prevIndex, prevTerm)

		// 一致性检查
		// 1: 日志不存在
		if len(f.Log) < prevIndex {
			//fmt.Println("日志不存在")
			XTerm = -1
			XLen = len(f.Log)
			Success = false
		} else {
			// 2: 任期不匹配
			if f.Log[prevIndex] != prevTerm {
				//	fmt.Println("任期不匹配")
				XTerm = f.Log[prevIndex]
				// 找到该任期在日志第一次出现的下标
				for i := 1; i <= prevIndex; i++ {
					if f.Log[i] == XTerm {
						XIndex = i
					}
				}
				Success = false
			}

			// 3: 检查通过
			if f.Log[prevIndex] == prevTerm {
				Success = true
			}

		}

		// 一致性检查通过
		if Success {
			fmt.Printf("通过: l->s%d NextIndex=%d\n", f.Me, l.NextIndex[f.Me])
			break
		}
		// 一致性检查未通过
		if !Success {
			// 任期冲突导致失败
			if XTerm != -1 {
				find := false
				for j := range l.Log {
					// 找到该冲突任期在leader日志中的下标
					if l.Log[j] == XTerm {
						l.NextIndex[f.Me] = j + 1
						find = true
						break
					}
				}
				// leader日志中不存在该冲突任期
				if !find {
					l.NextIndex[f.Me] = XIndex
				}
			} else { // 日志不存在
				l.NextIndex[f.Me] = XLen
			}
		}
	}
}

func repetitionRPCs(l *Leader, f *Follower, time int) {
	addlog := l.Log[len(l.Log)-1] + 1
	l.Log = append(l.Log, addlog)

	entries := make([]int, len(l.Log[l.NextIndex[f.Me]:]))
	copy(entries, l.Log[l.NextIndex[f.Me]:])

	fmt.Printf("l->s%d 追加%d次 相同日志: %d\n", f.Me, time, addlog)
	for i := 0; i < time; i++ {
		AppednLog(f, l.NextIndex[f.Me], entries)
	}
}

func main() {
	s0 := Follower{
		Me:  0,
		Log: []int{0, 1, 2, 3},
	}

	s1 := Follower{
		Me:  1,
		Log: []int{0, 1, 2, 2, 2, 2, 2, 2, 2, 2},
	}

	leader := Leader{
		Log:       []int{0, 1, 2, 3, 4, 5, 5},
		NextIndex: []int{7, 7},
	}

	fmt.Printf("初始日志:\nl: %v\ns0:%v\ns1:%v\n", leader.Log, s0.Log, s1.Log)

	nextindex(&leader, &s0)
	nextindex(&leader, &s1)
	//repetitionRPCs(&leader, &s0, 2)
	repetitionRPCs(&leader, &s1, 2)
	fmt.Printf("l: %v\ns0:%v\ns1:%v\n", leader.Log, s0.Log, s1.Log)

}
