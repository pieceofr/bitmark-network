package p2p

import (
	"bitmark-network/routing"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bitmark-inc/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	curPath := os.Getenv("PWD")
	var logLevel map[string]string
	logLevel = make(map[string]string, 0)
	logLevel["DEFAULT"] = "info"
	var logConfig = logger.Configuration{
		Directory: curPath,
		File:      "p2p_test.log",
		Size:      1048576,
		Count:     20,
		Console:   true,
		Levels:    logLevel,
	}
	if err := logger.Initialise(logConfig); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	globalData.log = logger.New("nodes")
	os.Exit(m.Run())
}

func mockConfiguration(nType string, port int) *Configuration {
	return &Configuration{
		PublicIP:           []string{"127.0.0.1", "[::1]"},
		NodeType:           nType,
		Port:               port,
		DynamicConnections: true,
		PreferIPv6:         false,
		Listen:             []string{"0.0.0.0:2136"},
		Announce:           []string{"127.0.0.1:2136", "[::1]:2136"},
		PrivateKey:         "080112406eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8",
		Connect:            []StaticConnection{},
	}
}
func TestNewP2P(t *testing.T) {
	err := Initialise(mockConfiguration("servant", 12136))
	assert.NoError(t, err, "P2P  initialized error")
	time.Sleep(8 * time.Second)
	defer routing.Finalise()
}
