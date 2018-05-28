package agent

const (
	CMD_ERROR = iota + 1    // 错误响应
	CMD_TICK                // 心跳包
	CMD_AGENT
	CMD_STOP
	CMD_RELOAD
	CMD_SHOW_MEMBERS
	CMD_CRONTAB_CHANGE
	CMD_RUN_COMMAND
	CMD_PULL_COMMAND
	CMD_DEL_CACHE
	CMD_CRONTAB_CHANGE_OK
)

const (
	tcpMaxSendQueue               = 10000
	tcpDefaultReadBufferSize      = 1024
)


const (
	tcpNodeOnline = 1 << iota
)


type NodeFunc   func(n *TcpClientNode)
type NodeOption func(n *TcpClientNode)
type TcpClients []*TcpClientNode

var (
	packDataTickOk     = Pack(CMD_TICK, []byte("keepalive response ok"))
)

type AgentConfig struct {
	Enable bool          `toml:"enable"`
	Type string          `toml:"type"`
	Lock string          `toml:"lock"`
	AgentListen string   `toml:"agent_listen"`
	ConsulAddress string `toml:"consul_address"`
}

