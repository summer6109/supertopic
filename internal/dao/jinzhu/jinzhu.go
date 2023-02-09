// Core service implement base gorm+mysql/postgresql/sqlite3.
// Jinzhu is the primary developer of gorm so use his name as
// pakcage name as a saluter.

package jinzhu

import (
	"github.com/Masterminds/semver/v3"
	"github.com/rocboss/paopao-ce/internal/conf"
	"github.com/rocboss/paopao-ce/internal/core"
	"github.com/rocboss/paopao-ce/internal/dao/cache"
	"github.com/rocboss/paopao-ce/internal/dao/security"
	"github.com/sirupsen/logrus"
)

var (
	_ core.DataService = (*dataServant)(nil)
	_ core.VersionInfo = (*dataServant)(nil)
)

type dataServant struct {
	core.IndexPostsService
	core.WalletService
	core.MessageService
	core.TopicService
	core.TweetService
	core.TweetManageService
	core.TweetHelpService
	core.CommentService
	core.CommentManageService
	core.UserManageService
	core.SecurityService
	core.AttachmentCheckService
}

func NewDataService() (core.DataService, core.VersionInfo) {
	// initialize CacheIndex if needed
	var (
		c core.CacheIndexService
		v core.VersionInfo
	)
	db := conf.MustGormDB()

	i := newIndexPostsService(db)
	if conf.CfgIf("SimpleCacheIndex") {
		i = newSimpleIndexPostsService(db)
		c, v = cache.NewSimpleCacheIndexService(i)
	} else if conf.CfgIf("BigCacheIndex") {
		c, v = cache.NewBigCacheIndexService(i)
	} else {
		c, v = cache.NewNoneCacheIndexService(i)
	}
	logrus.Infof("use %s as cache index service by version: %s", v.Name(), v.Version())

	ds := &dataServant{
		IndexPostsService:      c,
		WalletService:          newWalletService(db),
		MessageService:         newMessageService(db),
		TopicService:           newTopicService(db),
		TweetService:           newTweetService(db),
		TweetManageService:     newTweetManageService(db, c),
		TweetHelpService:       newTweetHelpService(db),
		CommentService:         newCommentService(db),
		CommentManageService:   newCommentManageService(db),
		UserManageService:      newUserManageService(db),
		SecurityService:        newSecurityService(db),
		AttachmentCheckService: security.NewAttachmentCheckService(),
	}
	return ds, ds
}

func NewAuthorizationManageService() core.AuthorizationManageService {
	return &authorizationManageServant{
		db: conf.MustGormDB(),
	}
}

func (s *dataServant) Name() string {
	return "Gorm"
}

func (s *dataServant) Version() *semver.Version {
	return semver.MustParse("v0.1.0")
}
