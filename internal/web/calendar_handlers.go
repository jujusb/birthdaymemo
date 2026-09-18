package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mcbill1/birthdaymemo/internal/i18n"
	"github.com/mcbill1/birthdaymemo/internal/models"
)

// monthCell 月视图单元格
type monthCell struct {
	Day       int              `json:"day"`
	Birthdays []birthdayInCell `json:"birthdays"`
}

type birthdayInCell struct {
	ID            uint         `json:"id"`
	Name          string       `json:"name"`
	Gender        string       `json:"gender"`
	Color         string       `json:"color"`
	Day           int          `json:"day"`          // 生日日期（几号）
	Age           int          `json:"age"`          // 当前年龄，-1=未知
	UpcomingAge   int          `json:"upcoming_age"` // 即将到来的生日后的年龄
	Tags          []models.Tag `json:"tags"`
	Shared        bool         `json:"shared"`
	CanEdit       bool         `json:"can_edit"`
	OwnerUsername string       `json:"owner_username,omitempty"`
}

// monthViewData 月视图数据
type monthViewData struct {
	Year            int              `json:"year"`
	Month           int              `json:"month"`
	FirstWeekday    int              `json:"first_weekday"` // 0=周日
	DaysInMonth     int              `json:"days_in_month"`
	Birthdays       []birthdayInCell `json:"birthdays"`
	SelfBirthdayDay int              `json:"self_birthday_day"` // 任务8：本月本人生日的日期，0=本月不是本人生日或未设置
}

// yearMonthData 年视图单月数据
type yearMonthData struct {
	Month   int            `json:"month"`
	Count   int            `json:"count"`
	Genders map[string]int `json:"genders"`
}

// yearViewData 年视图数据
type yearViewData struct {
	Year   int             `json:"year"`
	Months []yearMonthData `json:"months"`
}

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	view := r.URL.Query().Get("view")
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	now := time.Now()
	year, _ := strconv.Atoi(yearStr)
	if year == 0 {
		year = now.Year()
	}

	if view == "year" {
		s.yearView(w, r, uid, year)
		return
	}
	month, _ := strconv.Atoi(monthStr)
	if month < 1 || month > 12 {
		month = int(now.Month())
	}
	s.monthView(w, r, uid, year, month)
}

func (s *Server) monthView(w http.ResponseWriter, r *http.Request, uid uint, year, month int) {
	bds, err := visibleBirthdays(s.db, uid)
	if err != nil {
		Fail(w, CodeInternal, i18n.T(s.cfg.Language, "error.internal"))
		return
	}
	// 仅保留本月
	filtered := bds[:0]
	for _, b := range bds {
		if b.BirthMonth == month {
			filtered = append(filtered, b)
		}
	}
	bds = filtered
	tagsByBid := s.birthdaysTagsMap(uid, bds)
	ownerNames := s.birthdayOwnerNames(bds)
	now := time.Now()

	out := make([]birthdayInCell, 0, len(bds))
	for _, b := range bds {
		tags := tagsByBid[b.ID]
		if tags == nil {
			tags = []models.Tag{}
		}
		shared := b.UserID != uid
		out = append(out, birthdayInCell{
			ID:            b.ID,
			Name:          b.Name,
			Gender:        b.Gender,
			Color:         models.GenderColor(b.Gender),
			Day:           b.BirthDay,
			Age:           b.Age(now),
			UpcomingAge:   b.UpcomingAge(now),
			Tags:          tags,
			Shared:        shared,
			CanEdit:       !shared || canEditBirthday(s.db, uid, b),
			OwnerUsername: ownerNames[b.UserID],
		})
	}

	// 任务8：查询本人生日是否在本月
	var u models.User
	selfDay := 0
	if err := s.db.First(&u, uid).Error; err == nil {
		if u.HasSelfBirthday() && u.SelfBirthMonth == month {
			// 防御：避免日期超出当月天数
			daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
			if u.SelfBirthDay <= daysInMonth {
				selfDay = u.SelfBirthDay
			}
		}
	}

	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.Local).Day()
	OK(w, monthViewData{
		Year:            year,
		Month:           month,
		FirstWeekday:    int(first.Weekday()),
		DaysInMonth:     daysInMonth,
		Birthdays:       out,
		SelfBirthdayDay: selfDay,
	})
}

func (s *Server) yearView(w http.ResponseWriter, r *http.Request, uid uint, year int) {
	bds, _ := visibleBirthdays(s.db, uid)
	months := make([]yearMonthData, 12)
	for i := range months {
		months[i] = yearMonthData{Month: i + 1, Genders: map[string]int{
			models.GenderMale: 0, models.GenderFemale: 0, models.GenderNone: 0,
		}}
	}
	for _, b := range bds {
		idx := b.BirthMonth - 1
		if idx < 0 || idx > 11 {
			continue
		}
		months[idx].Count++
		months[idx].Genders[b.Gender]++
	}
	OK(w, yearViewData{Year: year, Months: months})
}

// upcomingBirthday 即将到来的生日
type upcomingBirthday struct {
	ID            uint         `json:"id"`
	Name          string       `json:"name"`
	Gender        string       `json:"gender"`
	Color         string       `json:"color"`
	BirthYear     int          `json:"birth_year"`
	BirthMonth    int          `json:"birth_month"`
	BirthDay      int          `json:"birth_day"`
	DaysUntil     int          `json:"days_until"`
	UpcomingAge   int          `json:"upcoming_age"`
	Date          string       `json:"date"`
	Tags          []models.Tag `json:"tags"`
	Shared        bool         `json:"shared"`
	CanEdit       bool         `json:"can_edit"`
	OwnerUsername string       `json:"owner_username,omitempty"`
}

func (s *Server) handleUpcoming(w http.ResponseWriter, r *http.Request) {
	u := userFromContext(r)
	if u == nil {
		FailStatus(w, http.StatusUnauthorized, CodeUnauthorized, i18n.T(s.cfg.Language, "error.unauthorized"))
		return
	}
	rangeDays := u.TopbarRangeDays
	if rangeDays <= 0 {
		rangeDays = 30
	}
	var bds []models.Birthday
	bds, _ = visibleBirthdays(s.db, u.ID)
	tagsByBid := s.birthdaysTagsMap(u.ID, bds)
	ownerNames := s.birthdayOwnerNames(bds)
	now := time.Now()

	out := make([]upcomingBirthday, 0)
	for _, b := range bds {
		// 计算今年/明年的下一个生日
		next := nextBirthday(b.BirthMonth, b.BirthDay, now)
		days := int(next.Sub(now).Hours() / 24)
		if days < 0 {
			days = 0
		}
		if days > rangeDays {
			continue
		}
		tags := tagsByBid[b.ID]
		if tags == nil {
			tags = []models.Tag{}
		}
		shared := b.UserID != u.ID
		out = append(out, upcomingBirthday{
			ID:            b.ID,
			Name:          b.Name,
			Gender:        b.Gender,
			Color:         models.GenderColor(b.Gender),
			BirthYear:     b.BirthYear,
			BirthMonth:    b.BirthMonth,
			BirthDay:      b.BirthDay,
			DaysUntil:     days,
			UpcomingAge:   b.UpcomingAge(now),
			Date:          next.Format("2006-01-02"),
			Tags:          tags,
			Shared:        shared,
			CanEdit:       !shared || canEditBirthday(s.db, u.ID, b),
			OwnerUsername: ownerNames[b.UserID],
		})
	}
	OK(w, out)
}

// nextBirthday 计算从 now 起最近的一次生日日期（含今天）
func nextBirthday(month, day int, now time.Time) time.Time {
	y := now.Year()
	t := time.Date(y, time.Month(month), day, 0, 0, 0, 0, now.Location())
	// 处理2/29等不存在日期：若非法则回退到3/1
	if t.Month() != time.Month(month) {
		// 当天不存在（如2/29在平年），取当月最后一天
		last := time.Date(y, time.Month(month)+1, 0, 0, 0, 0, 0, now.Location())
		t = last
	}
	if t.Before(now) {
		t = time.Date(y+1, time.Month(month), day, 0, 0, 0, 0, now.Location())
		if t.Month() != time.Month(month) {
			last := time.Date(y+1, time.Month(month)+1, 0, 0, 0, 0, 0, now.Location())
			t = last
		}
	}
	return t
}
