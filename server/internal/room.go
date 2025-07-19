package connection

import (
	"drizlink/server/interfaces"
	"fmt"
	"strconv"
	"time"
)

var roomIdCounter = 1

func generateRoomId() string {
	id := strconv.Itoa(roomIdCounter)
	roomIdCounter++
	return id
}

func HandleCreateRoom(server *interfaces.Server, creator *interfaces.User, roomName string, participantIds []string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	// Validate participants exist and are online
	participants := make(map[string]*interfaces.User)
	participants[creator.UserId] = creator // Add creator to participants

	for _, participantId := range participantIds {
		if participant, exists := server.Connections[participantId]; exists && participant.IsOnline {
			participants[participantId] = participant
		} else {
			_, err := creator.Conn.Write([]byte(fmt.Sprintf("❌ User %s not found or offline\n", participantId)))
			if err != nil {
				fmt.Println("Error sending create room error:", err)
			}
			return
		}
	}

	// Create room
	roomId := generateRoomId()
	room := &interfaces.Room{
		RoomId:       roomId,
		RoomName:     roomName,
		Creator:      creator.UserId,
		Participants: participants,
		Messages:     make(chan interfaces.Message, 100),
		CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}

	server.Rooms[roomId] = room

	// Notify all participants about room creation
	for _, participant := range participants {
		message := fmt.Sprintf("🏠 Room '%s' (ID: %s) created by %s. You have been added to the room.\n", roomName, roomId, creator.Username)
		_, err := participant.Conn.Write([]byte(message))
		if err != nil {
			fmt.Printf("Error notifying participant %s: %v\n", participant.Username, err)
		}
	}

	fmt.Printf("Room '%s' (ID: %s) created by %s with %d participants\n", roomName, roomId, creator.Username, len(participants))
}

func HandleJoinRoom(server *interfaces.Server, user *interfaces.User, roomId string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	room, exists := server.Rooms[roomId]
	if !exists {
		_, err := user.Conn.Write([]byte("❌ Room not found\n"))
		if err != nil {
			fmt.Println("Error sending join room error:", err)
		}
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Check if user is already in room
	if _, alreadyIn := room.Participants[user.UserId]; alreadyIn {
		_, err := user.Conn.Write([]byte("⚠️ You are already in this room\n"))
		if err != nil {
			fmt.Println("Error sending join room warning:", err)
		}
		return
	}

	// Add user to room
	room.Participants[user.UserId] = user

	// Notify user
	_, err := user.Conn.Write([]byte(fmt.Sprintf("✅ Successfully joined room '%s' (ID: %s)\n", room.RoomName, roomId)))
	if err != nil {
		fmt.Println("Error sending join confirmation:", err)
	}

	// Notify other participants
	for _, participant := range room.Participants {
		if participant != user && participant.IsOnline {
			message := fmt.Sprintf("👋 %s joined room '%s'\n", user.Username, room.RoomName)
			_, err := participant.Conn.Write([]byte(message))
			if err != nil {
				fmt.Printf("Error notifying participant %s: %v\n", participant.Username, err)
			}
		}
	}

	fmt.Printf("User %s joined room '%s' (ID: %s)\n", user.Username, room.RoomName, roomId)
}

func HandleLeaveRoom(server *interfaces.Server, user *interfaces.User, roomId string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	room, exists := server.Rooms[roomId]
	if !exists {
		_, err := user.Conn.Write([]byte("❌ Room not found\n"))
		if err != nil {
			fmt.Println("Error sending leave room error:", err)
		}
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Check if user is in room
	if _, inRoom := room.Participants[user.UserId]; !inRoom {
		_, err := user.Conn.Write([]byte("⚠️ You are not in this room\n"))
		if err != nil {
			fmt.Println("Error sending leave room warning:", err)
		}
		return
	}

	// Remove user from room
	delete(room.Participants, user.UserId)

	// Clear current room if this was the selected room
	if user.CurrentRoom == roomId {
		user.CurrentRoom = ""
	}

	// Notify user
	_, err := user.Conn.Write([]byte(fmt.Sprintf("✅ Successfully left room '%s' (ID: %s)\n", room.RoomName, roomId)))
	if err != nil {
		fmt.Println("Error sending leave confirmation:", err)
	}

	// Notify other participants
	for _, participant := range room.Participants {
		if participant.IsOnline {
			message := fmt.Sprintf("👋 %s left room '%s'\n", user.Username, room.RoomName)
			_, err := participant.Conn.Write([]byte(message))
			if err != nil {
				fmt.Printf("Error notifying participant %s: %v\n", participant.Username, err)
			}
		}
	}

	// Delete room if empty
	if len(room.Participants) == 0 {
		delete(server.Rooms, roomId)
		fmt.Printf("Room '%s' (ID: %s) deleted - no participants remaining\n", room.RoomName, roomId)
	}

	fmt.Printf("User %s left room '%s' (ID: %s)\n", user.Username, room.RoomName, roomId)
}

func HandleSelectRoom(server *interfaces.Server, user *interfaces.User, roomId string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	room, exists := server.Rooms[roomId]
	if !exists {
		_, err := user.Conn.Write([]byte("❌ Room not found\n"))
		if err != nil {
			fmt.Println("Error sending select room error:", err)
		}
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Check if user is in room
	if _, inRoom := room.Participants[user.UserId]; !inRoom {
		_, err := user.Conn.Write([]byte("❌ You are not a participant in this room\n"))
		if err != nil {
			fmt.Println("Error sending select room error:", err)
		}
		return
	}

	// Set current room
	user.CurrentRoom = roomId

	// Notify user
	_, err := user.Conn.Write([]byte(fmt.Sprintf("✅ Selected room '%s' (ID: %s) as active room\n", room.RoomName, roomId)))
	if err != nil {
		fmt.Println("Error sending select confirmation:", err)
	}

	fmt.Printf("User %s selected room '%s' (ID: %s) as active\n", user.Username, room.RoomName, roomId)
}

func HandleListRooms(server *interfaces.Server, user *interfaces.User) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	_, err := user.Conn.Write([]byte("🏠 Available Rooms:\n"))
	if err != nil {
		fmt.Println("Error sending room list header:", err)
		return
	}

	if len(server.Rooms) == 0 {
		_, err := user.Conn.Write([]byte("No rooms available\n"))
		if err != nil {
			fmt.Println("Error sending no rooms message:", err)
		}
		return
	}

	for _, room := range server.Rooms {
		room.Mutex.Lock()
		participantCount := len(room.Participants)
		isParticipant := ""
		if _, inRoom := room.Participants[user.UserId]; inRoom {
			isParticipant = " (You are in this room)"
		}
		
		activeIndicator := ""
		if user.CurrentRoom == room.RoomId {
			activeIndicator = " [ACTIVE]"
		}
		
		roomInfo := fmt.Sprintf("  🏠 %s (ID: %s) - %d participants%s%s\n", 
			room.RoomName, room.RoomId, participantCount, isParticipant, activeIndicator)
		_, err := user.Conn.Write([]byte(roomInfo))
		if err != nil {
			fmt.Printf("Error sending room info: %v\n", err)
		}
		room.Mutex.Unlock()
	}
}

func HandleRoomInfo(server *interfaces.Server, user *interfaces.User, roomId string) {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()

	room, exists := server.Rooms[roomId]
	if !exists {
		_, err := user.Conn.Write([]byte("❌ Room not found\n"))
		if err != nil {
			fmt.Println("Error sending room info error:", err)
		}
		return
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Send room details
	_, err := user.Conn.Write([]byte(fmt.Sprintf("🏠 Room Information:\n")))
	if err != nil {
		fmt.Println("Error sending room info header:", err)
		return
	}

	_, err = user.Conn.Write([]byte(fmt.Sprintf("  Name: %s\n", room.RoomName)))
	if err != nil {
		fmt.Println("Error sending room name:", err)
		return
	}

	_, err = user.Conn.Write([]byte(fmt.Sprintf("  ID: %s\n", room.RoomId)))
	if err != nil {
		fmt.Println("Error sending room ID:", err)
		return
	}

	creatorName := "Unknown"
	if creator, exists := server.Connections[room.Creator]; exists {
		creatorName = creator.Username
	}
	_, err = user.Conn.Write([]byte(fmt.Sprintf("  Creator: %s\n", creatorName)))
	if err != nil {
		fmt.Println("Error sending room creator:", err)
		return
	}

	_, err = user.Conn.Write([]byte(fmt.Sprintf("  Created: %s\n", room.CreatedAt)))
	if err != nil {
		fmt.Println("Error sending room creation time:", err)
		return
	}

	_, err = user.Conn.Write([]byte(fmt.Sprintf("  Participants (%d):\n", len(room.Participants))))
	if err != nil {
		fmt.Println("Error sending participants header:", err)
		return
	}

	for _, participant := range room.Participants {
		status := "Offline"
		if participant.IsOnline {
			status = "Online"
		}
		participantInfo := fmt.Sprintf("    👤 %s (ID: %s) - %s\n", participant.Username, participant.UserId, status)
		_, err := user.Conn.Write([]byte(participantInfo))
		if err != nil {
			fmt.Printf("Error sending participant info: %v\n", err)
		}
	}
}