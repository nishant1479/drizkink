# 🔗 DrizLink - P2P File Sharing Application 🔗

A peer-to-peer file sharing application with room-based communication and integrated chat functionality, allowing users to create rooms, communicate, and share files directly with each other in organized groups.

## ✨ Features

- **👤 User Authentication**: Connect with a username and maintain persistent sessions
- **🏠 Room Management**: Create and join rooms for organized communication
- **💬 Real-time Chat**: Send and receive messages globally or within specific rooms
- **📁 File Sharing**: Transfer files directly between users
- **📂 Folder Sharing**: Share entire folders with other users
- **🔍 File Discovery**: Look up and browse other users' shared directories
- **🎯 Room-based Operations**: File transfers and lookups work within room context
- **🔄 Automatic Reconnection**: Seamlessly reconnect with your existing session
- **👥 Status Tracking**: Monitor which users are currently online
- **🎨 Colorful UI**: Enhanced CLI interface with colors and emojis
- **📊 Progress Bars**: Visual feedback for file and folder transfers
- **🔒 Data Integrity**: MD5 checksum verification for files and folders

## 🚀 Installation

### Prerequisites
- Go (1.16 or later) 🔧

### Steps
1. Clone the repository ⬇️
```bash
git clone https://github.com/Harsh2563/DrizLink_Cli.git
cd DrizLink_Cli
```

2. Build the application 🛠️
```bash
go build -o DrizLink_Cli
```

## 🎮 Usage

### Starting the Server 🖥️
```bash
# Start server on default port 8080
go run ./server/cmd --port 8080

# Start server on custom port
go run ./server/cmd --port 3000

```

### Connecting as a Client 📱
```bash
# Connect to local server with default port
go run ./client/cmd --server localhost:8080

# Connect to remote server
go run ./client/cmd --server 192.168.0.203:4000

```

The application will validate:
- Server availability before client connection attempts
- Port availability before starting a server
- Existence of shared folder paths

## 🏠 Room-Based Architecture

DrizLink now operates with a room-based system for enhanced organization:

### How Rooms Work
1. **🌐 Global Discovery**: All connected users are visible via `/status` command
2. **🏠 Room Creation**: Any user can create a room and invite specific users
3. **💬 Room Chat**: Messages sent within a room are only visible to room participants
4. **📁 Room File Sharing**: File operations (send, lookup, download) work within room context
5. **🎯 Selective Communication**: Users can switch between rooms or communicate globally

### Room Workflow
1. Connect to server and see all online users with `/status`
2. Create a room with `/createroom <roomName> <userId1> <userId2> ...`
3. Select the room as active with `/selectroom <roomId>`
4. Chat and share files within the room context
5. Switch rooms or leave rooms as needed

## 🏗️ Architecture

The application follows a hybrid P2P architecture with room-based organization:
- 🌐 A central server handles user registration, discovery, and connection brokering
- 🏠 Server manages room creation, membership, and message routing
- ↔️ File and folder transfers occur directly between peers
- 🎯 Room context ensures organized communication and file sharing
- 💓 Server maintains connection status through regular heartbeat checks

## 📝 Commands

### Chat Commands 💬
| Command | Description |
|---------|-------------|
| `/help` | Show all available commands |
| `/status` | Show online users |
| `exit` | Disconnect and exit the application |

### Room Management 🏠
| Command | Description |
|---------|-------------|
| `/createroom <roomName> <userId1> [userId2] ...` | Create a new room with participants |
| `/joinroom <roomId>` | Join an existing room |
| `/leaveroom <roomId>` | Leave a room |
| `/selectroom <roomId>` | Select active room for chat and transfers |
| `/listrooms` | List all available rooms |
| `/roominfo <roomId>` | Show detailed room information |

### File Operations 📂
| Command | Description |
|---------|-------------|
| `/lookup <userId>` | Browse user's shared files |
| `/sendfile <userId> <filePath>` | Send a file to another user |
| `/sendfolder <userId> <folderPath>` | Send a folder to another user |
| `/download <userId> <filename>` | Download a file from another user |

**Note**: File operations work within the context of your selected room. Both users must be in the same room for transfers.

### Transfer Controls 📡
| Command | Description |
|---------|-------------|
| `/transfers` | Show all active transfers |
| `/pause <transferId>` | Pause an active transfer |
| `/resume <transferId>` | Resume a paused transfer |

## Terminal UI Features 🎨

- 🌈 **Color-coded messages**:
  - Commands appear in blue
  - Success messages appear in green
  - Error messages appear in red
  - User status notifications in yellow
   - Room messages have special formatting
  
- 📊 **Progress bars for file transfers**:
  ```
  [===================================>------] 75% (1.2 MB/1.7 MB)
  ```

- 📁 **Improved file listings**:
  ```
  === FOLDERS ===
  📁 [FOLDER] documents (Size: 0 bytes)
  📁 [FOLDER] images (Size: 0 bytes)
  
  === FILES ===
  📄 [FILE] document.pdf (Size: 1024 bytes)
  📄 [FILE] image.jpg (Size: 2048 bytes)
  ```

- 🏠 **Room indicators**:
  ```
  [Room: MyRoom] >>> Hello everyone in this room!
  ```

## 🎯 Usage Examples

### Creating and Using Rooms
```bash
# 1. Check who's online
/status

# 2. Create a room with specific users
/createroom ProjectTeam 1234 5678

# 3. Select the room as active
/selectroom 1

# 4. Chat within the room
Hello team! Let's share some files.

# 5. Share files within the room
/sendfile 1234 /path/to/document.pdf

# 6. List all rooms
/listrooms

# 7. Get room details
/roominfo 1
```

### File Sharing Workflow
```bash
# 1. Join or create a room with target users
/createroom FileShare 2345

# 2. Select the room
/selectroom 1

# 3. Look up user's files
/lookup 2345

# 4. Send files or folders
/sendfile 2345 /path/to/file.txt
/sendfolder 2345 /path/to/folder

# 5. Download files
/download 2345 filename.txt
```

## 🔒 Security

The application implements security measures including:

- **📁 Folder Path Validation**: The application verifies that shared folder paths exist before establishing a connection. If an invalid path is provided, the user will be prompted to enter a valid folder path.
- **🔌 Server Availability Check**: Client automatically verifies server availability before attempting connection, preventing connection errors.
- **🚫 Port Conflict Prevention**: Server detects if a port is already in use and alerts the user to choose another port.
- **🏠 Room-based Access Control**: File operations are restricted to users within the same room context.
- **👥 Session Management**: Server tracks IP addresses and user sessions for reconnection security.
- **🔐 Checksum Verification**: All file and folder transfers include MD5 checksum calculation to verify data integrity:
  - When sending, a unique MD5 hash is calculated for the file/folder contents
  - During transfer, the hash is securely transmitted alongside the data
  - Upon receiving, a new hash is calculated from the received data
  - The application compares both hashes to confirm the transfer was successful and uncorrupted
  - Users receive visual confirmation of integrity checks with clear success/failure messages

This checksum process ensures that files and folders arrive exactly as they were sent, protecting against data corruption during transfer.

## 🔍 New: LAN Peer Discovery (UDP Broadcast)

DrizLink now supports automatic discovery of other users/servers on the same WiFi network using UDP broadcast!

### How it works
- When you start the client, it sends a UDP broadcast to find all available DrizLink servers on your LAN.
- All running servers reply with their IP and port.
- The client shows a list of discovered peers (your friends on the same WiFi).
- You can select which peer(s) to connect to and create rooms or share files.

### Steps
1. **Start the server** on each device (your friends must run `go run ./server/cmd --port 8080` or similar).
2. **Run the client** (`go run ./client/cmd`).
3. When prompted, choose to search for available servers. The client will show all discovered peers.
4. **Select the peer(s)** you want to connect to from the list.
5. Proceed to create rooms and share files as usual!

No need to manually enter IP addresses—just make sure all devices are on the same WiFi network.

Made with ❤️ by the DrizLink Team
