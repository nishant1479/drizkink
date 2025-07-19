package connection

import (
	"drizlink/helper"
	"drizlink/server/interfaces"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func Connect(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return nil, err
	}
	return listener, nil
}

func Close(conn net.Conn) {
	conn.Close()
}

func Start(server *interfaces.Server) {
	listen, err := net.Listen("tcp", server.Address)
	if err != nil {
		fmt.Println("error in listen")
		panic(err)
	}

	defer listen.Close()
	fmt.Println("Server started on", server.Address)

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("error in accept")
			continue
		}

		go HandleConnection(conn, server)
	}
}

func HandleConnection(conn net.Conn, server *interfaces.Server) {
	ipAddr := conn.RemoteAddr().String()
	ip := strings.Split(ipAddr, ":")[0]
	fmt.Println("New connection from", ip)
	if existingUser := server.IpAddresses[ip]; existingUser != nil {
		fmt.Println("Connection already exists for IP:", ip)
		// Send reconnection signal with existing user data
		reconnectMsg := fmt.Sprintf("/RECONNECT %s %s", existingUser.Username, existingUser.StoreFilePath)
		_, err := conn.Write([]byte(reconnectMsg))
		if err != nil {
			fmt.Println("Error sending reconnect signal:", err)
			return
		}

		// Update connection and online status
		server.Mutex.Lock()
		existingUser.Conn = conn
		existingUser.IsOnline = true
		server.Mutex.Unlock()

		// Encrypt and broadcast welcome back message
		welcomeMsg := fmt.Sprintf("User %s has rejoined the chat", existingUser.Username)
		BroadcastGlobalMessage(welcomeMsg, server, existingUser)

		// Start handling messages for the reconnected user
		handleUserMessages(conn, existingUser, server)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("error in read username")
		return
	}
	username := string(buffer[:n])

	n, err = conn.Read(buffer)
	if err != nil {
		fmt.Println("error in read storeFilePath")
		return
	}
	storeFilePath := string(buffer[:n])

	userId := helper.GenerateUserId()

	user := &interfaces.User{
		UserId:        userId,
		Username:      username,
		StoreFilePath: storeFilePath,
		Conn:          conn,
		IsOnline:      true,
		IpAddress:     ip,
	}

	server.Mutex.Lock()
	server.Connections[user.UserId] = user
	server.IpAddresses[ip] = user

	// Initialize rooms map if not exists
	if server.Rooms == nil {
		server.Rooms = make(map[string]*interfaces.Room)
	}
	server.Mutex.Unlock()

	welcomeMsg := fmt.Sprintf("User %s has joined the chat", username)
	BroadcastGlobalMessage(welcomeMsg, server, user)

	fmt.Printf("New user connected: %s (ID: %s)\n", username, userId)

	// Start handling messages for the new user
	handleUserMessages(conn, user, server)
}

func handleUserMessages(conn net.Conn, user *interfaces.User, server *interfaces.Server) {
	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("User disconnected: %s\n", user.Username)
			server.Mutex.Lock()
			user.IsOnline = false
			server.Mutex.Unlock()
			offlineMsg := fmt.Sprintf("User %s is now offline", user.Username)
			BroadcastGlobalMessage(offlineMsg, server, user)
			return
		}

		messageContent := string(buffer[:n])

		switch {
		case messageContent == "/exit":
			server.Mutex.Lock()
			user.IsOnline = false
			server.Mutex.Unlock()
			offlineMsg := fmt.Sprintf("User %s is now offline", user.Username)
			BroadcastGlobalMessage(offlineMsg, server, user)
			return
		case strings.HasPrefix(messageContent, "/createroom"):
			args := strings.Fields(messageContent)
			if len(args) < 3 {
				_, err = conn.Write([]byte("❌ Invalid arguments. Use: /createroom <roomName> <userId1> [userId2] ...\n"))
				if err != nil {
					fmt.Println("Error sending create room error:", err)
				}
				continue
			}
			roomName := args[1]
			participantIds := args[2:]
			HandleCreateRoom(server, user, roomName, participantIds)
			continue
		case strings.HasPrefix(messageContent, "/joinroom"):
			args := strings.Fields(messageContent)
			if len(args) != 2 {
				_, err = conn.Write([]byte("❌ Invalid arguments. Use: /joinroom <roomId>\n"))
				if err != nil {
					fmt.Println("Error sending join room error:", err)
				}
				continue
			}
			roomId := args[1]
			HandleJoinRoom(server, user, roomId)
			continue
		case strings.HasPrefix(messageContent, "/leaveroom"):
			args := strings.Fields(messageContent)
			if len(args) != 2 {
				_, err = conn.Write([]byte("❌ Invalid arguments. Use: /leaveroom <roomId>\n"))
				if err != nil {
					fmt.Println("Error sending leave room error:", err)
				}
				continue
			}
			roomId := args[1]
			HandleLeaveRoom(server, user, roomId)
			continue
		case strings.HasPrefix(messageContent, "/selectroom"):
			args := strings.Fields(messageContent)
			if len(args) != 2 {
				_, err = conn.Write([]byte("❌ Invalid arguments. Use: /selectroom <roomId>\n"))
				if err != nil {
					fmt.Println("Error sending select room error:", err)
				}
				continue
			}
			roomId := args[1]
			HandleSelectRoom(server, user, roomId)
			continue
		case strings.HasPrefix(messageContent, "/listrooms"):
			HandleListRooms(server, user)
			continue
		case strings.HasPrefix(messageContent, "/roominfo"):
			args := strings.Fields(messageContent)
			if len(args) != 2 {
				_, err = conn.Write([]byte("❌ Invalid arguments. Use: /roominfo <roomId>\n"))
				if err != nil {
					fmt.Println("Error sending room info error:", err)
				}
				continue
			}
			roomId := args[1]
			HandleRoomInfo(server, user, roomId)
			return
		case strings.HasPrefix(messageContent, "/FILE_REQUEST"):
			args := strings.SplitN(messageContent, " ", 5) // Updated to include checksum
			if len(args) < 4 {
				fmt.Println("Invalid arguments. Use: /FILE_REQUEST <userId> <filename> <fileSize> [checksum]")
				continue
			}
			recipientId := args[1]
			fileName := args[2]
			fileSizeStr := strings.TrimSpace(args[3])
			fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)

			// Include checksum in filename if provided
			if len(args) == 5 {
				checksum := strings.TrimSpace(args[4])
				fileName = fileName + "|" + checksum
			}

			if err != nil {
				fmt.Println("Invalid fileSize. Use: /FILE_REQUEST <userId> <filename> <fileSize> [checksum]")
				continue
			}

			HandleFileTransfer(server, conn, recipientId, fileName, fileSize)
			continue
		case strings.HasPrefix(messageContent, "/FOLDER_REQUEST"):
			args := strings.SplitN(messageContent, " ", 5) // Updated to include checksum
			if len(args) < 4 {
				fmt.Println("Invalid arguments. Use: /FOLDER_REQUEST <userId> <folderName> <folderSize> [checksum]")
				continue
			}
			recipientId := args[1]
			folderName := args[2]
			folderSizeStr := strings.TrimSpace(args[3])
			folderSize, err := strconv.ParseInt(folderSizeStr, 10, 64)

			// Include checksum in foldername if provided
			if len(args) == 5 {
				checksum := strings.TrimSpace(args[4])
				folderName = folderName + "|" + checksum
			}

			if err != nil {
				fmt.Println("Invalid folderSize. Use: /FOLDER_REQUEST <userId> <folderName> <folderSize> [checksum]")
				continue
			}

			HandleFolderTransfer(server, conn, recipientId, folderName, folderSize)
			continue
		case messageContent == "PONG\n":
			continue
		case strings.HasPrefix(messageContent, "/status"):
			_, err = conn.Write([]byte("USERS:\n"))
			if err != nil {
				fmt.Println("Error sending user list header:", err)
				continue
			}
			server.Mutex.Lock()
			for _, user := range server.Connections {
				if user.IsOnline {
					roomStatus := "No room"
					if user.CurrentRoom != "" {
						if room, exists := server.Rooms[user.CurrentRoom]; exists {
							roomStatus = fmt.Sprintf("In room: %s", room.RoomName)
						}
					}
					statusMsg := fmt.Sprintf("%s [ID: %s] - %s\n", user.Username, user.UserId, roomStatus)
					_, err = conn.Write([]byte(statusMsg))
					if err != nil {
						fmt.Println("Error sending user list:", err)
						continue
					}
				}
			}
			server.Mutex.Unlock()
			continue
		case strings.HasPrefix(messageContent, "/LOOK"):
			args := strings.SplitN(messageContent, " ", 2)
			if len(args) != 2 {
				fmt.Println("Invalid arguments. Use: /LOOK <userId>")
				continue
			}
			recipientId := strings.TrimSpace(args[1])
			HandleLookupRequest(server, conn, recipientId)
			continue
		case strings.HasPrefix(messageContent, "/DIR_LISTING"):
			args := strings.SplitN(messageContent, " ", 3)
			if len(args) != 3 {
				fmt.Println("Invalid arguments. Use: /DIR_LISTING <userId> <files>")
				continue
			}
			userId := strings.TrimSpace(args[1])
			files := strings.TrimSpace(args[2])
			HandleLookupResponse(server, conn, userId, strings.Split(files, " "))
			continue
		case strings.HasPrefix(messageContent, "/DOWNLOAD_REQUEST"):
			args := strings.SplitN(messageContent, " ", 3)
			if len(args) != 3 {
				fmt.Println("Invalid arguments. Use: /DOWNLOAD_REQUEST <userId> <filename>")
				continue
			}
			senderId := strings.TrimSpace(args[1])
			recipientId := user.UserId
			filePath := strings.TrimSpace(args[2])
			HandleDownloadRequest(server, conn, senderId, recipientId, filePath)
			continue
		default:
			// Send message to current room or globally if no room selected
			if user.CurrentRoom != "" {
				BroadcastRoomMessage(messageContent, server, user, user.CurrentRoom)
			} else {
				BroadcastGlobalMessage(messageContent, server, user)
			}
		}
	}
}

func BroadcastGlobalMessage(content string, server *interfaces.Server, sender *interfaces.User) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()
	for _, recipient := range server.Connections {
		if recipient.IsOnline && recipient != sender {
			_, _ = recipient.Conn.Write([]byte(fmt.Sprintf("%s: %s\n", sender.Username, content)))
		}
	}
}

func BroadcastRoomMessage(content string, server *interfaces.Server, sender *interfaces.User, roomId string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	room, exists := server.Rooms[roomId]
	if !exists {
		_, _ = sender.Conn.Write([]byte("❌ Room not found\n"))
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	for _, participant := range room.Participants {
		if participant.IsOnline && participant != sender {
			_, _ = participant.Conn.Write([]byte(fmt.Sprintf("[%s] %s: %s\n", room.RoomName, sender.Username, content)))
		}
	}
}

func StartHeartBeat(interval time.Duration, server *interfaces.Server) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			server.Mutex.Lock()
			for _, user := range server.Connections {
				if user.IsOnline {
					_, err := user.Conn.Write([]byte("PING\n"))
					if err != nil {
						fmt.Printf("User disconnected: %s\n", user.Username)
						user.IsOnline = false
						BroadcastGlobalMessage(fmt.Sprintf("User %s is now offline", user.Username), server, user)
					}
				}
			}
			server.Mutex.Unlock()
		}
	}()
}
