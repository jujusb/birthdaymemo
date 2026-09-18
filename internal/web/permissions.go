package web

import (
	"errors"

	"github.com/mcbill1/birthdaymemo/internal/models"
	"gorm.io/gorm"
)

// ---- 委托编辑权限层（单点收敛，供 birthday/calendar/share/grant 共用） ----

// grantMaps 返回某 grantee 相对某 owner 的授权：tagID -> permission
func grantMaps(db *gorm.DB, granteeID, ownerID uint) map[uint]string {
	out := map[uint]string{}
	if granteeID == 0 || ownerID == 0 || granteeID == ownerID {
		return out
	}
	var grants []models.TagGrant
	db.Where("grantee_user_id = ? AND owner_user_id = ?", granteeID, ownerID).Find(&grants)
	for _, g := range grants {
		out[g.TagID] = g.Permission
	}
	return out
}

// birthdayTagIDs 查询一个生日当前的所有 tagID
func birthdayTagIDs(db *gorm.DB, birthdayID uint) []uint {
	var bts []models.BirthdayTag
	db.Where("birthday_id = ?", birthdayID).Find(&bts)
	out := make([]uint, 0, len(bts))
	for _, bt := range bts {
		out = append(out, bt.TagID)
	}
	return out
}

// canViewBirthday 生日可见：本人/admin，或 ≥1 tag 被授予 view/edit
func canViewBirthday(db *gorm.DB, uid uint, bd models.Birthday) bool {
	if uid == 0 {
		return false
	}
	if uid == bd.UserID {
		return true
	}
	var u models.User
	if err := db.First(&u, uid).Error; err == nil && u.Role == models.RoleAdmin {
		return true
	}
	gm := grantMaps(db, uid, bd.UserID)
	if len(gm) == 0 {
		return false
	}
	for _, tid := range birthdayTagIDs(db, bd.ID) {
		if p, ok := gm[tid]; ok && (p == models.GrantView || p == models.GrantEdit) {
			return true
		}
	}
	return false
}

// canEditBirthday 生日可编辑：本人/admin，或 ≥1 当前 tag 被授予 edit
// 注意：删除不走此函数（grantee 永远不可删除，见 deleteBirthday）
func canEditBirthday(db *gorm.DB, uid uint, bd models.Birthday) bool {
	if uid == 0 {
		return false
	}
	if uid == bd.UserID {
		return true
	}
	var u models.User
	if err := db.First(&u, uid).Error; err == nil && u.Role == models.RoleAdmin {
		return true
	}
	gm := grantMaps(db, uid, bd.UserID)
	if len(gm) == 0 {
		return false
	}
	for _, tid := range birthdayTagIDs(db, bd.ID) {
		if gm[tid] == models.GrantEdit {
			return true
		}
	}
	return false
}

// errCreateResolve 创建目标解析失败
var (
	errNoGrant      = errors.New("no grant")
	errMixedOwners  = errors.New("mixed owners")
	errViewOnly     = errors.New("view only")
	errInvalidScope = errors.New("invalid scope")
)

// resolveCreateOwner 解析 POST /api/birthdays 的实际 owner：
//   - tag 全为自己所有 → owner = uid（原行为）
//   - 含被授予 edit 的他人 tag（且只能来自同一个 owner）→ owner = 该 grantor（创建进对方日历）
//   - 含 view-only 或无权 tag，或混合多 owner → 报错
func resolveCreateOwner(db *gorm.DB, uid uint, tagIDs []uint) (uint, error) {
	if len(tagIDs) == 0 {
		return uid, nil
	}
	// 去重
	seen := map[uint]struct{}{}
	uniq := make([]uint, 0, len(tagIDs))
	for _, t := range tagIDs {
		if t == 0 {
			continue
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			uniq = append(uniq, t)
		}
	}
	if len(uniq) == 0 {
		return uid, nil
	}
	var tags []models.Tag
	db.Where("id IN ?", uniq).Find(&tags)
	if len(tags) != len(uniq) {
		return 0, errInvalidScope
	}
	byID := make(map[uint]models.Tag, len(tags))
	for _, t := range tags {
		byID[t.ID] = t
	}
	var grantor uint
	grantorSet := false
	for _, tid := range uniq {
		t := byID[tid]
		if t.UserID == uid {
			continue
		}
		var g models.TagGrant
		if err := db.Where("grantee_user_id = ? AND tag_id = ?", uid, tid).First(&g).Error; err != nil {
			return 0, errNoGrant
		}
		if g.Permission != models.GrantEdit {
			return 0, errViewOnly
		}
		if g.OwnerUserID != t.UserID {
			return 0, errInvalidScope
		}
		if !grantorSet {
			grantor = g.OwnerUserID
			grantorSet = true
		} else if grantor != g.OwnerUserID {
			return 0, errMixedOwners
		}
	}
	if grantorSet {
		return grantor, nil
	}
	return uid, nil
}

// validateUpdateTagSet 校验 PUT 结果 tag 集：
//  1. 每个 tag 必须是调用者自己所有，或被授予 edit 的他人 tag（且属于同一生日 owner）
//  2. 结果集必须仍保留 ≥1 个授予-edit tag（防止剥离 granting tag 后劫持）——本人/admin 豁免
func validateUpdateTagSet(db *gorm.DB, uid uint, bd models.Birthday, newTagIDs []uint) error {
	var u models.User
	isOwnerOrAdmin := uid == bd.UserID
	if err := db.First(&u, uid).Error; err == nil && u.Role == models.RoleAdmin {
		isOwnerOrAdmin = true
	}
	if isOwnerOrAdmin {
		// 本人/admin：沿用原 validateTagIDs 语义（admin 跨 owner 由调用方处理）
		return nil
	}
	seen := map[uint]struct{}{}
	uniq := make([]uint, 0, len(newTagIDs))
	for _, t := range newTagIDs {
		if t == 0 {
			continue
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			uniq = append(uniq, t)
		}
	}
	if len(uniq) == 0 {
		return errNoGrant // grantee 不可把共享生日剥成无 tag（会使其不可见/劫持）
	}
	var tags []models.Tag
	db.Where("id IN ?", uniq).Find(&tags)
	if len(tags) != len(uniq) {
		return errInvalidScope
	}
	gm := grantMaps(db, uid, bd.UserID)
	kept := false
	for _, t := range tags {
		if t.UserID == uid {
			continue
		}
		if t.UserID != bd.UserID {
			return errInvalidScope
		}
		if gm[t.ID] != models.GrantEdit {
			return errNoGrant
		}
		kept = true
	}
	// 若调用者自己的 tag 全集 + 授予 tag 至少保留一个授予 tag 即 kept；
	// 若新集全是调用者自己的 tag（剥离了 granting tag）→ kept=false → 拒绝
	if !kept {
		// 例外：新集里含调用者自己 tag 的同时也可能含授予 tag；kept 已覆盖。
		// 全自有 = 试图转移所有权 → 拒绝
		return errNoGrant
	}
	return nil
}

// visibleBirthdays 返回 uid 可见的全部生日（本人 + 被授予 view/edit 的他人生日）
func visibleBirthdays(db *gorm.DB, uid uint) ([]models.Birthday, error) {
	var own []models.Birthday
	if err := db.Where("user_id = ?", uid).Find(&own).Error; err != nil {
		return nil, err
	}
	var grants []models.TagGrant
	db.Where("grantee_user_id = ?", uid).Find(&grants)
	if len(grants) == 0 {
		return own, nil
	}
	tagIDs := make([]uint, 0, len(grants))
	for _, g := range grants {
		tagIDs = append(tagIDs, g.TagID)
	}
	var bids []uint
	db.Model(&models.BirthdayTag{}).Where("tag_id IN ?", tagIDs).Distinct().Pluck("birthday_id", &bids)
	if len(bids) == 0 {
		return own, nil
	}
	var shared []models.Birthday
	db.Where("id IN ? AND user_id <> ?", bids, uid).Find(&shared)
	// 二次过滤：确保每个 shared 生日确实命中 ≥1 授予 tag（grant 可能指向已删除 tag）
	seen := map[uint]bool{}
	for _, b := range own {
		seen[b.ID] = true
	}
	out := append([]models.Birthday{}, own...)
	for _, b := range shared {
		if seen[b.ID] {
			continue
		}
		if canViewBirthday(db, uid, b) {
			out = append(out, b)
		}
	}
	return out, nil
}
