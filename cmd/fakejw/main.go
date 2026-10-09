// fakejw 本地开发用的模拟教务系统：不连学校服务器也能完整运行前后端。
//
//	go run ./cmd/fakejw            # 监听 :8081
//	JW_URL=http://localhost:8081 go run .
//
// 测试账号：学号 2024001，密码 pw，验证码 1234。
package main

import (
	"flag"
	"log"
	"net/http"

	"jww/internal/jwclient/jwtest"
	"jww/internal/model"
)

func main() {
	addr := flag.String("addr", ":8081", "listen address")
	flag.Parse()

	s := jwtest.NewHandler()
	s.SetCourses(func(week int) []model.Course {
		list := []model.Course{
			{Name: "高等数学", Teacher: "王老师", Room: "A101", DayOfWeek: 1, PeriodStart: 1, Periods: 2},
			{Name: "大学英语", Teacher: "赵老师", Room: "C301", DayOfWeek: 3, PeriodStart: 3, Periods: 2},
		}
		if week%2 == 0 {
			list = append(list, model.Course{Name: "体育", Teacher: "孙老师", Room: "操场", DayOfWeek: 5, PeriodStart: 5, Periods: 2})
		}
		return list
	})
	s.SetScores([]map[string]any{
		jwtest.ScoreRow("高等数学", "92", "4.2", "4"),
		jwtest.ScoreRow("大学英语", "85", "3.5", "3"),
	})
	log.Printf("fake jw listening on %s (账号 %s / %s，验证码 %s)", *addr, jwtest.Username, jwtest.Password, jwtest.Captcha)
	log.Fatal(http.ListenAndServe(*addr, s))
}
