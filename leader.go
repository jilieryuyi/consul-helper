package consul
import (
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/consul/api"
	"time"
	"fmt"
)

type Leader struct {
	service IService
	consulLock ILock
	lockKey string
	leader bool
	session ISession
	sessionId string
	health *api.Health
	ServiceName string
	ServiceID string
	ServiceHost string
	ServicePort int
}
type ILeader interface {
	Deregister() error
	Register() error
	UpdateTtl() error
	GetServices(passingOnly bool) ([]*ServiceMember, error)
	Select(onLeader func(*ServiceMember))
	Get() (*ServiceMember, error)
}

func NewLeader(
	address string, //127.0.0.1:8500
	lockKey string,
	name string,
	host string,
	port int,
	opts ...ServiceOption,
) ILeader {
	consulConfig        := api.DefaultConfig()
	consulConfig.Address = address
	c, err              := api.NewClient(consulConfig)
	if err != nil {
		log.Panicf("%v", err)
	}
	session        := c.Session()
	kv             := c.KV()
	mySession      := NewSession(session)
	sessionId, err := mySession.Create(10)

	sev := NewService(c.Agent(), name, host, port, opts...)
	l   := &Leader{
		service     : sev,
		consulLock  : NewLock(sessionId, kv),
		lockKey     : lockKey,
		leader      : false,
		session     : mySession,
		sessionId   : sessionId,
		health      : c.Health(),
		ServiceName : name,
		ServiceID   : fmt.Sprintf("%s-%s-%d", name, host, port),
		ServiceHost : host,
		ServicePort : port,
	}
	return l
}

func (sev *Leader) Deregister() error {
	return sev.Deregister()
}

func (sev *Leader) Register() error {
	return sev.service.Register()
}

func (sev *Leader) UpdateTtl() error {
	return sev.service.UpdateTtl()
}

func (sev *Leader) GetServices(passingOnly bool) ([]*ServiceMember, error) {
	members, _, err := sev.health.Service(sev.ServiceName, "", passingOnly, nil)
	if err != nil {
		return nil, err
	}
	//return members, err
	data := make([]*ServiceMember, 0)
	for _, v := range members {
		m := &ServiceMember{}
		if v.Checks.AggregatedStatus() == "passing" {
			m.Status = statusOnline
			m.IsLeader  = v.Service.Tags[0] == "isleader:true"
		} else {
			m.Status = statusOffline
			m.IsLeader  = false
		}
		m.ServiceID = v.Service.ID//Tags[1]
		m.ServiceIp = v.Service.Address
		m.Port      = v.Service.Port
		data        = append(data, m)
	}
	return data, nil
}

func (sev *Leader) Select(onLeader func(*ServiceMember)) {
	leader := &ServiceMember{
		IsLeader: false,
		ServiceID: sev.ServiceID,
		Status: statusOnline,
		ServiceIp: sev.ServiceHost,
		Port: sev.ServicePort,
	}
	go func() {
		success, err := sev.consulLock.Lock(sev.lockKey, 10)
		if err == nil {
			sev.leader = success
			leader.IsLeader = success
			go onLeader(leader)
			sev.Register()
		}
		for {
			success, err := sev.consulLock.Lock(sev.lockKey, 10)
			if err == nil {
				if success != sev.leader {
					sev.leader = success
					leader.IsLeader = success
					go onLeader(leader)
					sev.Register()
				}
			}
			sev.session.Renew(sev.sessionId)
			sev.UpdateTtl()
			time.Sleep(time.Second * 3)
		}
	}()
}

func (sev *Leader) Get() (*ServiceMember, error) {
	members, _ := sev.GetServices(true)
	if members == nil {
		return nil, membersEmpty
	}
	for _, v := range members {
		if v.IsLeader {
			return v, nil
		}
	}
	return nil, leaderNotFound
}
