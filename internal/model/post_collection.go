package model

import (
	"time"

	"gorm.io/gorm"
)

type PostCollection struct {
	*Model
	Post   *Post `json:"-"`
	PostID int64 `json:"post_id"`
	UserID int64 `json:"user_id"`
}

func (p *PostCollection) Get(db *gorm.DB) (*PostCollection, error) {
	var star PostCollection
	tn := db.NamingStrategy.TableName("PostCollection") + "."

	if p.Model != nil && p.ID > 0 {
		db = db.Where(tn+"id = ? AND "+tn+"is_del = ?", p.ID, 0)
	}
	if p.PostID > 0 {
		db = db.Where(tn+"post_id = ?", p.PostID)
	}
	if p.UserID > 0 {
		db = db.Where(tn+"user_id = ?", p.UserID)
	}

	db = db.Joins("Post").Where("Post.visibility <> ?", PostVisitPrivate).Order("Post.id DESC")
	err := db.First(&star).Error
	if err != nil {
		return &star, err
	}

	return &star, nil
}

func (p *PostCollection) Create(db *gorm.DB) (*PostCollection, error) {
	err := db.Omit("Post").Create(&p).Error

	return p, err
}

func (p *PostCollection) Delete(db *gorm.DB) error {
	return db.Model(&PostCollection{}).Omit("Post").Where("id = ? AND is_del = ?", p.Model.ID, 0).Updates(map[string]interface{}{
		"deleted_on": time.Now().Unix(),
		"is_del":     1,
	}).Error
}

func (p *PostCollection) List(db *gorm.DB, conditions *ConditionsT, offset, limit int) ([]*PostCollection, error) {
	var collections []*PostCollection
	var err error
	tn := db.NamingStrategy.TableName("PostCollection") + "."

	if offset >= 0 && limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	if p.UserID > 0 {
		db = db.Where(tn+"user_id = ?", p.UserID)
	}

	for k, v := range *conditions {
		if k == "ORDER" {
			db = db.Order(v)
		} else {
			db = db.Where(tn+k, v)
		}
	}

	db = db.Joins("Post").Where("Post.visibility <> ?", PostVisitPrivate).Order("Post.id DESC")
	if err = db.Where(tn+"is_del = ?", 0).Find(&collections).Error; err != nil {
		return nil, err
	}

	return collections, nil
}

func (p *PostCollection) Count(db *gorm.DB, conditions *ConditionsT) (int64, error) {
	var count int64
	tn := db.NamingStrategy.TableName("PostCollection") + "."

	if p.PostID > 0 {
		db = db.Where(tn+"post_id = ?", p.PostID)
	}
	if p.UserID > 0 {
		db = db.Where(tn+"user_id = ?", p.UserID)
	}
	for k, v := range *conditions {
		if k != "ORDER" {
			db = db.Where(tn+k, v)
		}
	}

	db = db.Joins("Post").Where("Post.visibility <> ?", PostVisitPrivate)
	if err := db.Model(p).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
