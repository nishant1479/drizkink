package main

import (
	"bufio"
	connection "drizlink/client/internal"
	"drizlink/helper"
	"drizlink/utils"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func discoverServers() []string {
	fmt.Println(utils.InfoColor("🔍 Searching for available servers via UDP broadcast..."))
	var availableServers []string

	// UDP broadcast
	broadcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: 9999}
	conn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		fmt.Println(utils.ErrorColor("❌ Error sending UDP broadcast:"), err)
		return availableServers
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	conn.Write([]byte("DRIZLINK_DISCOVER"))

	// Listen for responses
	listenConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		fmt.Println(utils.ErrorColor("❌ Error listening for UDP responses:"), err)
		return availableServers
	}
	defer listenConn.Close()

	listenConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	start := time.Now()
	for time.Since(start) < 2*time.Second {
		n, addr, err := listenConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		msg := string(buf[:n])
		if strings.HasPrefix(msg, "DRIZLINK_SERVER:") {
			port := strings.TrimPrefix(msg, "DRIZLINK_SERVER:")
			serverAddr := fmt.Sprintf("%s:%s", addr.IP.String(), port)
			if !contains(availableServers, serverAddr) {
				availableServers = append(availableServers, serverAddr)
				fmt.Printf("  ✅ Found server at %s\n", utils.SuccessColor(serverAddr))
			}
		}
	}
	return availableServers
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func promptForServerAddress() string {
	reader := bufio.NewReader(os.Stdin)

	// First try to discover servers automatically
	fmt.Println(utils.InfoColor("Would you like to search for available servers? (y/n)"))
	fmt.Print(utils.CommandColor(">>> "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice == "y" || choice == "yes" {
		servers := discoverServers()

		if len(servers) > 0 {
			fmt.Println(utils.SuccessColor("\n📡 Available servers found:"))
			for i, server := range servers {
				fmt.Printf("  %d. %s\n", i+1, utils.CommandColor(server))
			}

			fmt.Println(utils.InfoColor("\nEnter the number of the server to connect to:"))
			fmt.Print(utils.CommandColor(">>> "))
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if index, err := strconv.Atoi(choice); err == nil && index > 0 && index <= len(servers) {
				return servers[index-1]
			}
		} else {
			fmt.Println(utils.WarningColor("⚠️  No servers found automatically"))
		}
	}

	// Fallback to manual entry
	for {
		fmt.Println(utils.InfoColor("Enter server address (format host:port):"))
		fmt.Print(utils.CommandColor(">>> "))
		address, _ := reader.ReadString('\n')
		address = strings.TrimSpace(address)

		if !strings.Contains(address, ":") {
			fmt.Println(utils.ErrorColor("❌ Invalid address format. Please use host:port (e.g., localhost:8080)"))
			continue
		}

		// Check if server is available at this address
		available, errMsg := helper.CheckServerAvailability(address)
		if !available {
			fmt.Println(utils.ErrorColor("❌ No server available at " + address + ": " + errMsg))
			fmt.Println(utils.InfoColor("Would you like to try another address? (y/n)"))
			fmt.Print(utils.CommandColor(">>> "))

			retry, _ := reader.ReadString('\n')
			retry = strings.TrimSpace(strings.ToLower(retry))

			if retry != "y" && retry != "yes" {
				os.Exit(1)
			}
			continue
		}

		return address
	}
}

func main() {
	serverAddr := flag.String("server", "", "Server address in format host:port")
	flag.Parse()

	utils.PrintBanner()

	// If server address not provided via command line, ask user
	address := *serverAddr
	if address == "" {
		address = promptForServerAddress()
	} else {
		fmt.Println(utils.InfoColor("Connecting to server at " + address + "..."))

		// Check if server is available
		available, errMsg := helper.CheckServerAvailability(address)
		if !available {
			fmt.Println(utils.ErrorColor("❌ Error: No server running at " + address))
			fmt.Println(utils.ErrorColor("  Details: " + errMsg))
			fmt.Println(utils.InfoColor("Please check the address or start a server first."))
			return
		}
	}

	conn, err := connection.Connect(address)
	if err != nil {
		if err.Error() == "reconnect" {
			goto startChat
		} else {
			fmt.Println(utils.ErrorColor("❌ Error connecting to server:"), err)
			return
		}
	}

	defer connection.Close(conn)

	fmt.Println(utils.InfoColor("Please login to continue:"))
	err = connection.UserInput("Username", conn)
	if err != nil {
		if err.Error() == "reconnect" {
			goto startChat
		} else {
			fmt.Println(utils.ErrorColor("❌ Error during login:"), err)
			return
		}
	}

	err = connection.UserInput("Store File Path", conn)
	if err != nil {
		if err.Error() == "reconnect" {
			goto startChat
		} else {
			fmt.Println(utils.ErrorColor("❌ Error setting file path:"), err)
			return
		}
	}

startChat:
	fmt.Println(utils.HeaderColor("\n✨ Welcome to DrizLink - P2P File Sharing! ✨"))
	fmt.Println(utils.InfoColor("------------------------------------------------"))
	fmt.Println(utils.SuccessColor("✅ Successfully connected to server!"))
	fmt.Println(utils.InfoColor("Type /help to see available commands"))
	fmt.Println(utils.InfoColor("------------------------------------------------"))

	go connection.ReadLoop(conn)
	connection.WriteLoop(conn)
}
