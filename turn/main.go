package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn"
)

func createAuthHandler() turn.AuthHandler {
	return func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
		return []byte("password"), true
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	publicIP := flag.String("public-ip", "80.87.198.65", "IP Address that TURN can be contacted by.")

	if len(*publicIP) == 0 {
		log.Fatalf("'public-ip' is required")
	}

	realm := os.Getenv("REALM")
	if realm == "" {
		log.Panic("REALM is a required environment variable")
	}

	udpPortStr := os.Getenv("UDP_PORT")
	if udpPortStr == "" {
		udpPortStr = "3478"
	}

	// Create a UDP listener to pass into pion/turn
	// pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
	// this allows us to add logging, storage or modify inbound/outbound traffic
	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+udpPortStr)
	if err != nil {
		log.Panicf("Failed to create TURN server listener: %s", err)
	}
	//udpPort, err := strconv.Atoi(udpPortStr)
	//if err != nil {
	//	log.Panic(err)
	//}

	var channelBindTimeout time.Duration
	channelBindTimeoutStr := os.Getenv("CHANNEL_BIND_TIMEOUT")
	if channelBindTimeoutStr != "" {
		channelBindTimeout, err = time.ParseDuration(channelBindTimeoutStr)
		if err != nil {
			log.Panicf("CHANNEL_BIND_TIMEOUT=%s is an invalid time Duration", channelBindTimeoutStr)
		}
	}

	conf := turn.ServerConfig{
		Realm:              realm,
		AuthHandler:        createAuthHandler(),
		ChannelBindTimeout: channelBindTimeout,
		//ListeningPort:      udpPort,
		LoggerFactory:      logging.NewDefaultLoggerFactory(),
		//Software:           os.Getenv("SOFTWARE"),
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(*publicIP), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",              // But actually be listening on every interface
				},
			},
		},
	}

	s , err := turn.NewServer(conf)
	if err != nil {
		log.Panic(err)
	}

	//err = s.Start()
	//if err != nil {
	//	log.Panic(err)
	//}

	<-sigs

	if err = s.Close(); err != nil {
		log.Panic(err)
	}
}
