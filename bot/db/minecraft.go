package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Tnze/go-mc/chat"
	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/xrjr/mcutils/pkg/bedrock"
	"github.com/xrjr/mcutils/pkg/ping"
)

type MinecraftServerDB interface {
	Set(hash string, server MinecraftServer) error
	Get(hash string) (MinecraftServer, error)
	GetAll() ([]MinecraftServer, error)
	Del(hash string) error
}

type minecraftServerDBImpl struct {
	db *redis.Client
}

func (m minecraftServerDBImpl) Set(hash string, server MinecraftServer) error {
	buf, err := json.Marshal(server)
	if err != nil {
		return err
	}
	res := m.db.HSet(context.TODO(), "mc-server", hash, buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (m minecraftServerDBImpl) Get(hash string) (MinecraftServer, error) {
	res := m.db.HGet(context.TODO(), "mc-server", hash)
	if err := res.Err(); err != nil {
		return MinecraftServer{}, err
	}
	var val MinecraftServer
	if err := json.Unmarshal([]byte(res.Val()), &val); err != nil {
		return MinecraftServer{}, err
	}
	return val, nil
}

func (m minecraftServerDBImpl) GetAll() ([]MinecraftServer, error) {
	res := m.db.HGetAll(context.TODO(), "mc-server")
	if err := res.Err(); err != nil {
		return nil, err
	}
	var val []MinecraftServer
	for _, v := range res.Val() {
		var v2 MinecraftServer
		if err := json.Unmarshal([]byte(v), &v2); err != nil {
			continue
		}
		val = append(val, v2)
	}
	return val, nil
}

func (m minecraftServerDBImpl) Del(hash string) error {
	res := m.db.HDel(context.TODO(), "mc-server", hash)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

type MinecraftServerType string

const (
	MinecraftServerTypeJava    = "java"
	MinecraftServerTypeBedrock = "bedrock"
)

func Address2Hash(address string, port int) (string, error) {
	hash := sha256.New()
	if _, err := io.WriteString(hash, fmt.Sprintf("%s:%d", address, port)); err != nil {
		return "", err
	}
	buf := hash.Sum(nil)
	str := hex.EncodeToString(buf)
	return str, nil
}

func NewMinecraftServer(hash, address string, port uint16, s_type MinecraftServerType) MinecraftServer {
	return MinecraftServer{
		Hash:    hash,
		Address: address,
		Port:    port,
		Type:    s_type,
	}
}

type MinecraftServer struct {
	Hash             string                 `json:"hash"`
	Type             MinecraftServerType    `json:"type"`
	Address          string                 `json:"address"`
	Port             uint16                 `json:"port"`
	WatchingGuilds   []snowflake.ID         `json:"watching_guilds"`
	LastResponseTime time.Time              `json:"last_response_time"`
	LastResponse     *MinecraftPingResponse `json:"last_response"`
}

func (m MinecraftServer) String() string {
	if (m.Port == 25565 && m.Type == MinecraftServerTypeJava) || (m.Port == 19132 && m.Type == MinecraftServerTypeBedrock) {
		return m.Address
	}
	return fmt.Sprintf("%s:%d", m.Address, m.Port)
}

func (ms *MinecraftServer) Fetch() (r *MinecraftPingResponse, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	defer func() {
		if r != nil {
			ms.LastResponse = r
		}
	}()
	go func() {
		switch ms.Type {
		case MinecraftServerTypeJava:
			c := ping.NewClient(ms.Address, int(ms.Port))
			if err = c.Connect(); err != nil {
				return
			}
			defer func() { _ = c.Disconnect() }()
			var h ping.Handshake
			t := time.Now()
			h, err = c.Handshake()
			if err != nil {
				return
			}
			latency := time.Since(t).Milliseconds()
			r = &MinecraftPingResponse{
				Infos:   h.Properties.Infos(),
				Latency: latency,
				Type:    ms.Type,
			}
			nbt_mes := chat.Message{}
			b, _ := json.Marshal(h.Properties["description"])
			_ = nbt_mes.UnmarshalJSON(b)
			r.Description = nbt_mes.String()
		case MinecraftServerTypeBedrock:
			c := bedrock.NewClient(ms.String(), int(ms.Port))
			if err = c.Connect(); err != nil {
				return
			}
			var res bedrock.UnconnectedPong
			var latency int
			res, latency, err = c.UnconnectedPing()
			if err != nil {
				return
			}
			r = &MinecraftPingResponse{
				Infos: ping.Infos{
					Description: res.MOTD,
					Favicon:     "",
				},
				Latency: int64(latency),
				Type:    ms.Type,
			}
			r.Version.Name = res.MinecraftVersion
			r.Version.Protocol = res.ProtocolVersion
			r.Players.Max = res.MaxPlayers
			r.Players.Online = res.OnlinePlayers
		}
		cancel()
	}()
	for {
		time.Sleep(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("timeout")
			}
			return
		default:
		}
	}
}

type MinecraftPingResponse struct {
	ping.Infos
	Latency int64
	Type    MinecraftServerType `json:"type"`
}

func (r MinecraftPingResponse) AnsiDescription() string {
	return chat.Text(r.Description).String()
}
