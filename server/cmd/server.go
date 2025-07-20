package main

import (
	helper "drizlink/helper"
	"drizlink/server/interfaces"
	connection "drizlink/server/internal"
	"drizlink/utils"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"
)

func listenForUDPBroadcast(port string) {
	addr := net.UDPAddr{
		Port: 9999,
		IP:   net.IPv4zero,
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error starting UDP broadcast listener:", err)
		return
	}
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		msg := string(buf[:n])
		if msg == "DRIZLINK_DISCOVER" {
			// Reply with our TCP server address
			response := fmt.Sprintf("DRIZLINK_SERVER:%s", port)
			conn.WriteToUDP([]byte(response), remoteAddr)
		}
	}
}

func main() {
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	// Ensure port starts with a colon for address format
	formattedPort := *port
	if !strings.HasPrefix(formattedPort, ":") {
		formattedPort = ":" + formattedPort
	}

	// Check if port is already in use
	if helper.IsPortInUse(*port) {
		fmt.Println(utils.ErrorColor("❌ Error: Port " + *port + " is already in use"))
		fmt.Println(utils.InfoColor("Please choose a different port or stop the other server."))
		return
	}

	utils.PrintBanner()
	fmt.Println(utils.InfoColor("Starting server on port " + *port + "..."))

	server := interfaces.Server{
		Address:     formattedPort,
		Connections: make(map[string]*interfaces.User),
		IpAddresses: make(map[string]*interfaces.User),
		Messages:    make(chan interfaces.Message),
	}

	go connection.StartHeartBeat(100*time.Second, &server)
	go listenForUDPBroadcast(*port)
	connection.Start(&server)
}
