package consul

import (
	"github.com/hashicorp/consul/api"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type Session struct {
	session *api.Session
	timeout int64 //seconds
	ID string
}

func NewSession(session *api.Session, timeout int64) *Session {
	s := &Session{
		session:session,
		timeout:timeout,
		ID:"",
	}
	s.create()
	return s
}
// create a session
func (session *Session) create() {
	se := &api.SessionEntry{
		Behavior : api.SessionBehaviorDelete,
		//TTL: fmt.Sprintf("%ds", session.timeout),
	}
	if session.timeout > 0 {
		se.TTL = fmt.Sprintf("%ds", session.timeout)
	}
	ID, _, err := session.session.Create(se, nil)
	if err != nil {
		log.Errorf("create session error: %+v", err)
		return
	}
	session.ID = ID
}

// destory a session
func (session *Session) Destroy() error {
	_, err := session.session.Destroy(session.ID, nil)
	return err
}

// refresh a session
func (session *Session) Renew() error {
	_, _, err := session.session.Renew(session.ID, nil)
	return err
}
