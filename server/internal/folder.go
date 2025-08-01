package connection

import (
	"drizlink/server/interfaces"
	"fmt"
	"io"
	"net"
	"strings"
)

func HandleFolderTransfer(server *interfaces.Server, conn net.Conn, recipientId, folderName string, folderSize int64) {
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
					_, err := sender.Conn.Write([]byte("❌ Both users must be in the same room for folder transfer\n"))
					if err != nil {
						fmt.Printf("Error sending room restriction message: %v\n", err)
					}
					return
				}
			}
		}
		
		// Send folder transfer response to recipient
		_, err := recipient.Conn.Write([]byte(fmt.Sprintf("/FOLDER_RESPONSE %s %s %d %s", recipientId, folderName, folderSize, recipient.StoreFilePath)))
		if err != nil {
			fmt.Printf("Error sending folder response to %s: %v\n", recipientId, err)
			return
		}

		// Forward the zipped folder data from sender to recipient
		n, err := io.CopyN(recipient.Conn, conn, folderSize)
		if err != nil {
			fmt.Printf("Error transferring folder data: %v\n", err)
			return
		}
		fmt.Printf("Transferred %d bytes of folder data\n", n)
	} else {
		fmt.Printf("User %s not found or offline\n", recipientId)
	}
}

func HandleLookupRequest(server *interfaces.Server, conn net.Conn, userId string) {
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
	
	recipient, exists := server.Connections[userId]
	fmt.Printf("StoreFilePath: %s\n", recipient.StoreFilePath)
	if !exists {
		fmt.Printf("User %s not found\n", userId)
		_, err := conn.Write([]byte(fmt.Sprintf("User %s not found\n", userId)))
		if err != nil {
			fmt.Printf("Error sending lookup response: %v\n", err)
		}
		return
	}

	if !recipient.IsOnline {
		fmt.Printf("User %s is not online\n", userId)
		_, err := conn.Write([]byte(fmt.Sprintf("User %s is not online\n", userId)))
		if err != nil {
			fmt.Printf("Error sending lookup response: %v\n", err)
		}
		return
	}
	
	// Check if both users are in the same room (if requester has a current room)
	if requester.CurrentRoom != "" {
		room, roomExists := server.Rooms[requester.CurrentRoom]
		if roomExists {
			room.Mutex.Lock()
			_, requesterInRoom := room.Participants[requester.UserId]
			_, recipientInRoom := room.Participants[userId]
			room.Mutex.Unlock()
			
			if !requesterInRoom || !recipientInRoom {
				_, err := requester.Conn.Write([]byte("❌ Both users must be in the same room for file lookup\n"))
				if err != nil {
					fmt.Printf("Error sending room restriction message: %v\n", err)
				}
				return
			}
		}
	}

	// Send the lookup request to the recipient's connection
	fmt.Printf("StoreFilePath: %s\n", recipient.StoreFilePath)
	_, err := recipient.Conn.Write([]byte(fmt.Sprintf("/LOOK_REQUEST %s %s\n", userId, recipient.StoreFilePath)))
	if err != nil {
		fmt.Printf("Error sending lookup request to recipient: %v\n", err)
		_, respErr := conn.Write([]byte(fmt.Sprintf("Error looking up user %s's directory\n", userId)))
		if respErr != nil {
			fmt.Printf("Error sending error response: %v\n", respErr)
		}
		return
	}

	fmt.Printf("Lookup request sent to user %s\n", userId)
}

func HandleLookupResponse(server *interfaces.Server, conn net.Conn, userId string, files []string) {
	_, err := conn.Write([]byte(fmt.Sprintf("LOOK_RESPONSE %s %s\n", userId, strings.Join(files, " "))))
	if err != nil {
		fmt.Printf("Error sending lookup response: %v\n", err)
		return
	}
}
