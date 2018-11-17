package server

import (
	"testing"
	"net"
	"time"
)

type conn struct {
	net.Conn
}

func testMemMonitor() *Monitor{
	m := &Monitor{Repository:&MonitorStore{
		clients : make(map[string]ClientInfo),
		sessions : make(map[string]SessionInfo),
		subscriptions : make(map[string]map[string]SubscriptionsInfo),
	}}
	m.Repository.Open()
	return m
}
func (c *conn) RemoteAddr() net.Addr {
	return dummyAddr("test-remote")
}

func testMonitorClient(opts *ClientOptions) *Client {
	srv := NewServer()
	c := srv.newClient(&conn{})
	c.opts = opts
	return c
}

func TestMonitor_Register(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	m.Register(testMonitorClient(opts),false)
	cinfo , ok := m.GetClient(opts.ClientId)
	if !ok  {
		t.Fatalf("GetClient error, want true, got false")
	}
	if cinfo.ConnectedAt.IsZero() {
		t.Fatalf("ConnectedAt error, got zero")
	}
	want := ClientInfo{
		ClientId:opts.ClientId,
		Username:opts.Username,
		RemoteAddr:"test-remote",
		CleanSession:opts.CleanSession,
		KeepAlive:60,
		ConnectedAt:cinfo.ConnectedAt,
	}
	if cinfo != want {
		t.Fatalf("Register() error, want %v, got %v", want,cinfo)
	}

	clientList := m.Clients()
	if len(clientList) != 1 {
		t.Fatalf("Clients() len error, want 1, got %d",len(clientList))
	}
	if clientList[0] != want {
		t.Fatalf("Clients() error, want %v, got %v", want,cinfo)
	}

	if len(m.ClientSubscriptions(opts.ClientId)) != 0 {
		t.Fatalf("ClientSubscriptions() len error, want 0, got %d",len(m.ClientSubscriptions(opts.ClientId)))
	}
}

func TestMonitor_UnRegister(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	m.Register(testMonitorClient(opts),false)
	m.UnRegister(opts.ClientId,true)
	_, ok1 := m.Repository.GetClient(opts.ClientId)
	if ok1 {
		t.Fatalf("GetClient() error, want false, got true")
	}
	_, ok2 := m.Repository.GetSession(opts.ClientId)
	if ok2 {
		t.Fatalf("GetSession() error, want false, got true")
	}
}

func TestMonitor_UnRegister_SessionStore(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	client := testMonitorClient(opts)
	m.Register(testMonitorClient(opts),false)
	sub1 := SubscriptionsInfo{
		opts.ClientId,
		2,
		"qos2",
		time.Now(),
	}
	sub2 := SubscriptionsInfo{
		opts.ClientId,
		1,
		"qos1",
		time.Now().Add(1 * time.Second),
	}
	m.Subscribe(sub1)
	m.Subscribe(sub2)
	m.UnRegister(opts.ClientId,false)

	_, ok1 := m.Repository.GetClient(opts.ClientId)
	if ok1 {
		t.Fatalf("GetClient() error, want false, got true")
	}

	sgot, ok2 := m.Repository.GetSession(opts.ClientId)

	if !ok2 {
		t.Fatalf("GetSession() error, want true, got false")
	}

	swant := SessionInfo{
		ClientId:opts.ClientId,
		Status:STATUS_OFFLINE,
		RemoteAddr:client.rwc.RemoteAddr().String(),
		CleanSession:opts.CleanSession,
		Subscriptions: 2,
		MaxInflight: client.session.maxInflightMessages,
		InflightLen: 0,
		MaxMsgQueue: client.session.maxQueueMessages,
		MsgQueueLen: 0,
		MsgQueueDropped: 0,
		ConnectedAt: sgot.ConnectedAt,
		OfflineAt: sgot.OfflineAt,

	}
	if sgot != swant {
		t.Fatalf("GetSession() error, want %v, got %v", swant, sgot)
	}
	if time.Now().Second() - sgot.OfflineAt.Second() >= 10 {
		t.Fatalf("OfflineAt error, time.Now(): %d, OfflineAt: %d",time.Now().Second(), sgot.OfflineAt.Second())
	}

	sublist := m.Subscriptions()
	if len(sublist) == 0 {
		t.Fatalf("Subscriptions() error, want 2, got %d", len(sublist))
	}
	if sublist[0] != sub1 {
		t.Fatalf("Subscriptions()[0] error, want %v, got %v", sub1, sublist[0])
	}
	if sublist[1] != sub2 {
		t.Fatalf("Subscriptions()[1] error, want %v, got %v", sub2, sublist[1])
	}
	clientSubList := m.ClientSubscriptions(opts.ClientId)
	for k, v := range sublist {
		if clientSubList[k] != v {
			t.Fatalf("clientSubList[%d] error, want %v, got %v",k ,v, clientSubList[k])
		}
	}

}

func TestMonitor_Register_SessionReuse(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	client := testMonitorClient(opts)
	m.Register(client,false)
	sub1 := SubscriptionsInfo{
		opts.ClientId,
		2,
		"qos2",
		time.Now(),
	}
	sub2 := SubscriptionsInfo{
		opts.ClientId,
		1,
		"qos1",
		time.Now().Add(1 * time.Second),
	}
	m.Subscribe(sub1)
	m.Subscribe(sub2)

	s, _ := m.GetSession(opts.ClientId)
	if s.Subscriptions != 2 {
		t.Fatalf("Subscriptions error, want 2, got %d", s.Subscriptions)
	}

	m.UnRegister(opts.ClientId, false)

	m.Register(client, true)
	_, ok1 := m.Repository.GetClient(opts.ClientId)
	if !ok1 {
		t.Fatalf("GetClient() error, want true, got false")
	}

	sgot, ok2 := m.Repository.GetSession(opts.ClientId)

	if !ok2 {
		t.Fatalf("GetSession() error, want true, got false")
	}

	swant := SessionInfo{
		ClientId:opts.ClientId,
		Status:STATUS_ONLINE,
		RemoteAddr:client.rwc.RemoteAddr().String(),
		CleanSession:opts.CleanSession,
		Subscriptions: 2,
		MaxInflight: client.session.maxInflightMessages,
		InflightLen: 0,
		MaxMsgQueue: client.session.maxQueueMessages,
		MsgQueueLen: 0,
		MsgQueueDropped: 0,
		ConnectedAt: sgot.ConnectedAt,
		OfflineAt: sgot.OfflineAt,

	}
	if sgot != swant {
		t.Fatalf("GetSession() error, want %v, got %v", swant, sgot)
	}
	if time.Now().Second() - sgot.OfflineAt.Second() >= 10 {
		t.Fatalf("OfflineAt error, time.Now(): %d, OfflineAt: %d",time.Now().Second(), sgot.OfflineAt.Second())
	}

	sublist := m.Subscriptions()
	if len(sublist) == 0 {
		t.Fatalf("Subscriptions() error, want 2, got %d", len(sublist))
	}
	if sublist[0] != sub1 {
		t.Fatalf("Subscriptions()[0] error, want %v, got %v", sub1, sublist[0])
	}
	if sublist[1] != sub2 {
		t.Fatalf("Subscriptions()[1] error, want %v, got %v", sub2, sublist[1])
	}
	clientSubList := m.ClientSubscriptions(opts.ClientId)
	for k, v := range sublist {
		if clientSubList[k] != v {
			t.Fatalf("clientSubList[%d] error, want %v, got %v",k ,v, clientSubList[k])
		}
	}
}

func TestMonitor_MsgEnQueue_MsgDeQueue(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	client := testMonitorClient(opts)
	m.Register(client,false)
	m.MsgEnQueue(opts.ClientId)
	s, _ := m.GetSession(opts.ClientId)
	if s.MsgQueueLen != 1 {
		t.Fatalf("MsgQueueLen error, want 1, got %d",s.MsgQueueLen)
	}
	m.MsgDeQueue(opts.ClientId)
	s, _ = m.GetSession(opts.ClientId)
	if s.MsgQueueLen != 0 {
		t.Fatalf("MsgQueueLen error, want 0, got %d",s.MsgQueueLen)
	}
}

func TestMonitor_AddInflight_DelInflight(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	client := testMonitorClient(opts)
	m.Register(client,false)
	m.AddInflight(opts.ClientId)
	s, _ := m.GetSession(opts.ClientId)
	if s.InflightLen != 1 {
		t.Fatalf("InflightLen error, want 1, got %d",s.InflightLen)
	}
	m.DelInflight(opts.ClientId)
	s, _ = m.GetSession(opts.ClientId)
	if s.InflightLen != 0 {
		t.Fatalf("InflightLen error, want 0, got %d",s.InflightLen)
	}
}

func TestMonitor_Subscribe_UnSubscribe(t *testing.T) {
	m := testMemMonitor()
	defer m.Repository.Close()
	opts := &ClientOptions{
		ClientId:"clientId",
		Username:"username",
		KeepAlive:60,
		CleanSession:false,
	}
	client := testMonitorClient(opts)
	m.Register(client,false)
	sub1 := SubscriptionsInfo{
		opts.ClientId,
		2,
		"qos2",
		time.Now(),
	}
	sub2 := SubscriptionsInfo{
		opts.ClientId,
		1,
		"qos1",
		time.Now().Add(1 * time.Second),
	}
	m.Subscribe(sub1)
	m.Subscribe(sub2)

	sublist := m.Subscriptions()
	if len(sublist) == 0 {
		t.Fatalf("Subscriptions() error, want 2, got %d", len(sublist))
	}
	if sublist[0] != sub1 {
		t.Fatalf("Subscriptions()[0] error, want %v, got %v", sub1, sublist[0])
	}
	if sublist[1] != sub2 {
		t.Fatalf("Subscriptions()[1] error, want %v, got %v", sub2, sublist[1])
	}
	clientSubList := m.ClientSubscriptions(opts.ClientId)
	for k, v := range sublist {
		if clientSubList[k] != v {
			t.Fatalf("clientSubList[%d] error, want %v, got %v",k ,v, clientSubList[k])
		}
	}

	m.UnSubscribe(opts.ClientId,sub1.Name)
	m.UnSubscribe(opts.ClientId,sub2.Name)

	sublist = m.Subscriptions()
	if len(sublist) != 0 {
		t.Fatalf("Subscriptions() error, want 0, got %d", len(sublist))
	}

	clientSubList = m.ClientSubscriptions(opts.ClientId)
	if len(clientSubList) != 0 {
		t.Fatalf("ClientSubscriptions() error, want 0, got %d", len(clientSubList))
	}


}