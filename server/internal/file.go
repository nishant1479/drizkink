package connection

import (
	"drizlink/server/interfaces"
	"fmt"
	"io"
	"net"
	"strings"
)

func HandleFileTransfer(server *interfaces.Server, conn net.Conn, recipientId, fileName string, fileSize int64) {
	// Get sender information
	var sender *interfaces.User
	for _, user := range server.Connections {
		if user.Conn == conn {
			sender = user
			break
		}
	}
	
	if sender == nil {
		fmt.Println("Error: Could not identify sender")
		return
	}
	
	// Extract checksum if present
	checksum := ""
	fileNameWithChecksum := fileName
	
	parts := strings.SplitN(fileName, "|", 2)
	if len(parts) == 2 {
		fileName = parts[0]
		checksum = parts[1]
		fmt.Println("Original checksum:", checksum)
	}
	
	recipient, exists := server.Connections[recipientId]
	if exists && recipient.IsOnline {
		// Check if both users are in the same room (if sender has a current room)
		if sender.CurrentRoom != "" {
			room, roomExists := server.Rooms[sender.CurrentRoom]
			if roomExists {
				room.Mutex.Lock()
				_, senderInRoom := room.Participants[sender.UserId]
				_, recipientInRoom := room.Participants[recipientId]
				room.Mutex.Unlock()
				
				if !senderInRoom || !recipientInRoom {
					_, err := sender.Conn.Write([]byte("❌ Both users must be in the same room for file transfer\n"))
					if err != nil {
						fmt.Printf("Error sending room restriction message: %v\n", err)
					}
					return
				}
			}
		}
		
		// Include checksum in response if available
		_, err := recipient.Conn.Write([]byte(fmt.Sprintf("/FILE_RESPONSE %s %s %d %s", 
		    recipientId, fileNameWithChecksum, fileSize, recipient.StoreFilePath)))
		if err != nil {
			fmt.Printf("Error sending file response to %s: %v\n", recipientId, err)
		}
		n, err := io.CopyN(recipient.Conn, conn, fileSize)
		if err != nil {
			fmt.Printf("Error receiving file from %s: %v\n", recipientId, err)
		}
		fmt.Printf("Transferred %d bytes from %s\n", n, recipientId)
		if err != nil {
			fmt.Printf("Error sending file to %s: %v\n", recipientId, err)
		}
	} else {
		fmt.Printf("User %s not found or offline\n", recipientId)
	}
}

func SendFile(server *interfaces.Server, senderId, recipientId, filePath string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	_, exists := server.Connections[recipientId]
	if !exists {
		fmt.Printf("User %s not found\n", recipientId)
		return
	}

	sender, exists := server.Connections[senderId]
	if !exists {
		fmt.Printf("User %s not found\n", senderId)
		return
	}

	_, err := sender.Conn.Write([]byte(fmt.Sprintf("/sendfile %s %s\n", recipientId, filePath)))
	if err != nil {
		fmt.Printf("Error sending file to %s: %v\n", recipientId, err)
	}
}

func HandleDownloadRequest(server *interfaces.Server, conn net.Conn, senderId, recipientId, filePath string) {
	// Get requester information
	var requester *interfaces.User
	for _, user := range server.Connections {
		if user.Conn == conn {
			requester = user
			break
		}
	}
	
	if requester == nil {
		fmt.Println("Error: Could not identify requester")
		return
	}
	
	sender, exists := server.Connections[senderId]
	if !exists {
		fmt.Printf("User %s not found\n", senderId)
		return
	}

	if !sender.IsOnline {
		fmt.Printf("User %s is not online\n", senderId)
		return
	}
	
	// Check if both users are in the same room (if requester has a current room)
	if requester.CurrentRoom != "" {
		room, roomExists := server.Rooms[requester.CurrentRoom]
		if roomExists {
			room.Mutex.Lock()
			_, requesterInRoom := room.Participants[requester.UserId]
			_, senderInRoom := room.Participants[senderId]
			room.Mutex.Unlock()
			
			if !requesterInRoom || !senderInRoom {
				_, err := requester.Conn.Write([]byte("❌ Both users must be in the same room for file download\n"))
				if err != nil {
					fmt.Printf("Error sending room restriction message: %v\n", err)
				}
				return
			}
		}
	}

	_, err := sender.Conn.Write([]byte(fmt.Sprintf("/DOWNLOAD_REQUEST %s %s\n", recipientId, filePath)))
	if err != nil {
		fmt.Printf("Error sending file request to %s: %v\n", senderId, err)
	}
	fmt.Println("Download request sent successfully")
}
