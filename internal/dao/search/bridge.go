package search

import (
	"github.com/rocboss/paopao-ce/internal/core"
	"github.com/rocboss/paopao-ce/internal/model"
	"github.com/sirupsen/logrus"
)

var (
	_ core.TweetSearchService = (*bridgeTweetSearchServant)(nil)
)

type documents struct {
	primaryKey  []string
	docItems    core.DocItems
	identifiers []string
}

type bridgeTweetSearchServant struct {
	ts           core.TweetSearchService
	updateDocsCh chan *documents
}

func (s *bridgeTweetSearchServant) IndexName() string {
	return s.ts.IndexName()
}

func (s *bridgeTweetSearchServant) AddDocuments(data core.DocItems, primaryKey ...string) (bool, error) {
	s.updateDocs(&documents{
		primaryKey: primaryKey,
		docItems:   data,
	})
	return true, nil
}

func (s *bridgeTweetSearchServant) DeleteDocuments(identifiers []string) error {
	s.updateDocs(&documents{
		identifiers: identifiers,
	})
	return nil
}

func (s *bridgeTweetSearchServant) Search(user *model.User, q *core.QueryReq, offset, limit int) (*core.QueryResp, error) {
	return s.ts.Search(user, q, offset, limit)
}

func (s *bridgeTweetSearchServant) updateDocs(doc *documents) {
	select {
	case s.updateDocsCh <- doc:
		logrus.Debugln("addDocuments send documents by chan")
	default:
		go func(item *documents) {
			if len(item.docItems) > 0 {
				if _, err := s.ts.AddDocuments(item.docItems, item.primaryKey...); err != nil {
					logrus.Errorf("addDocuments in gorotine occurs error: %v", err)
				}
			} else if len(item.identifiers) > 0 {
				if err := s.ts.DeleteDocuments(item.identifiers); err != nil {
					logrus.Errorf("deleteDocuments in gorotine occurs error: %s", err)
				}
			}
		}(doc)
	}
}

func (s *bridgeTweetSearchServant) startUpdateDocs() {
	for doc := range s.updateDocsCh {
		if len(doc.docItems) > 0 {
			if _, err := s.ts.AddDocuments(doc.docItems, doc.primaryKey...); err != nil {
				logrus.Errorf("addDocuments occurs error: %v", err)
			}
		} else if len(doc.identifiers) > 0 {
			if err := s.ts.DeleteDocuments(doc.identifiers); err != nil {
				logrus.Errorf("deleteDocuments occurs error: %s", err)
			}
		}
	}
}
