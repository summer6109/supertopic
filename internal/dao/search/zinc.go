package search

import (
	"github.com/Masterminds/semver/v3"
	"github.com/rocboss/paopao-ce/internal/core"
	"github.com/rocboss/paopao-ce/internal/model"
	"github.com/rocboss/paopao-ce/pkg/json"
	"github.com/rocboss/paopao-ce/pkg/types"
	"github.com/rocboss/paopao-ce/pkg/zinc"
	"github.com/sirupsen/logrus"
)

var (
	_ core.TweetSearchService = (*zincTweetSearchServant)(nil)
	_ core.VersionInfo        = (*zincTweetSearchServant)(nil)
)

type zincTweetSearchServant struct {
	tweetSearchFilter

	indexName     string
	client        *zinc.ZincClient
	publicFilter  string
	privateFilter string
	friendFilter  string
}

func (s *zincTweetSearchServant) Name() string {
	return "Zinc"
}

func (s *zincTweetSearchServant) Version() *semver.Version {
	return semver.MustParse("v0.2.0")
}

func (s *zincTweetSearchServant) IndexName() string {
	return s.indexName
}

func (s *zincTweetSearchServant) AddDocuments(data core.DocItems, primaryKey ...string) (bool, error) {
	buf := make(core.DocItems, 0, len(data)+1)
	if len(primaryKey) > 0 {
		buf = append(buf, map[string]types.Any{
			"index": map[string]types.Any{
				"_index": s.indexName,
				"_id":    primaryKey[0],
			},
		})
	} else {
		buf = append(buf, map[string]types.Any{
			"index": map[string]types.Any{
				"_index": s.indexName,
			},
		})
	}
	buf = append(buf, data...)
	return s.client.BulkPushDoc(buf)
}

func (s *zincTweetSearchServant) DeleteDocuments(identifiers []string) error {
	for _, id := range identifiers {
		if err := s.client.DelDoc(s.indexName, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *zincTweetSearchServant) Search(user *model.User, q *core.QueryReq, offset, limit int) (resp *core.QueryResp, err error) {
	if q.Type == core.SearchTypeDefault && q.Query != "" {
		resp, err = s.queryByContent(user, q, offset, limit)
	} else if q.Type == core.SearchTypeTag && q.Query != "" {
		resp, err = s.queryByTag(user, q, offset, limit)
	} else {
		resp, err = s.queryAny(user, offset, limit)
	}
	if err != nil {
		logrus.Errorf("zincTweetSearchServant.search searchType:%s query:%s error:%v", q.Type, q.Query, err)
		return
	}

	logrus.Debugf("zincTweetSearchServant.Search type:%s query:%s resp Hits:%d NbHits:%d offset: %d limit:%d ", q.Type, q.Query, len(resp.Items), resp.Total, offset, limit)
	s.filterResp(user, resp)
	return
}

func (s *zincTweetSearchServant) queryByContent(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	resp, err := s.client.EsQuery(s.indexName, map[string]types.Any{
		"query": map[string]types.Any{
			"match_phrase": map[string]types.Any{
				"content": q.Query,
			},
		},
		"sort": []string{"-is_top", "-latest_replied_on"},
		"from": offset,
		"size": limit,
	})
	if err != nil {
		return nil, err
	}
	return s.postsFrom(resp)
}

func (s *zincTweetSearchServant) queryByTag(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	resp, err := s.client.ApiQuery(s.indexName, map[string]types.Any{
		"search_type": "querystring",
		"query": map[string]types.Any{
			"term": "tags." + q.Query + ":1",
		},
		"sort_fields": []string{"-is_top", "-latest_replied_on"},
		"from":        offset,
		"max_results": limit,
	})
	if err != nil {
		return nil, err
	}
	return s.postsFrom(resp)
}

func (s *zincTweetSearchServant) queryAny(user *model.User, offset, limit int) (*core.QueryResp, error) {
	queryMap := map[string]types.Any{
		"query": map[string]types.Any{
			"match_all": map[string]string{},
		},
		"sort": []string{"-is_top", "-latest_replied_on"},
		"from": offset,
		"size": limit,
	}
	resp, err := s.client.EsQuery(s.indexName, queryMap)
	if err != nil {
		return nil, err
	}
	return s.postsFrom(resp)
}

func (s *zincTweetSearchServant) postsFrom(resp *zinc.QueryResultT) (*core.QueryResp, error) {
	posts := make([]*model.PostFormated, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		item := &model.PostFormated{}
		raw, err := json.Marshal(hit.Source)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, item); err != nil {
			return nil, err
		}
		posts = append(posts, item)
	}

	return &core.QueryResp{
		Items: posts,
		Total: resp.Hits.Total.Value,
	}, nil
}

func (s *zincTweetSearchServant) createIndex() {
	// 不存在则创建索引
	s.client.CreateIndex(s.indexName, &zinc.ZincIndexProperty{
		"id": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Store:    true,
			Sortable: true,
		},
		"user_id": &zinc.ZincIndexPropertyT{
			Type:  "numeric",
			Index: true,
			Store: true,
		},
		"comment_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"collection_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"upvote_count": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"visibility": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"is_top": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"is_essence": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"content": &zinc.ZincIndexPropertyT{
			Type:           "text",
			Index:          true,
			Store:          true,
			Aggregatable:   true,
			Highlightable:  true,
			Analyzer:       "gse_search",
			SearchAnalyzer: "gse_standard",
		},
		"tags": &zinc.ZincIndexPropertyT{
			Type:  "keyword",
			Index: true,
			Store: true,
		},
		"ip_loc": &zinc.ZincIndexPropertyT{
			Type:  "keyword",
			Index: true,
			Store: true,
		},
		"latest_replied_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"attachment_price": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Sortable: true,
			Store:    true,
		},
		"created_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
		"modified_on": &zinc.ZincIndexPropertyT{
			Type:     "numeric",
			Index:    true,
			Sortable: true,
			Store:    true,
		},
	})
}
