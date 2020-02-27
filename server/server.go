package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"nvm.ga/mastersofcode/golang_2019/tictactoe/game"
	msg "nvm.ga/mastersofcode/golang_2019/tictactoe/messaging"
)

type Server struct {
	board        *game.Board
	gameLock     sync.Mutex
	xConn, yConn *ConnectedPlayer
	serverSock   net.Listener
	numClients   int
	gameStarted  bool
}

func StartServer(port string, done chan<- bool) {
	serverSock, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start a server: %s\n", err)
	}
	done <- true
	board := game.MakeBoard(3)
	server := &Server{board: board, serverSock: serverSock}
	server.run()
}

type ConnectedPlayer struct {
	player          game.Player
	conn            net.Conn
	opponentUpdates <-chan bool
	myUpdates       chan<- bool
}

func (server *Server) run() {
	updatesX := make(chan bool)
	updatesO := make(chan bool)
	for {
		// stop accepting any new clients when the game has started
		if server.gameStarted {
			return
		}
		conn, err := server.serverSock.Accept()
		if err != nil {
			log.Printf("Failed to handle a client: %s\n", err)
			continue
		}
		if server.numClients == 0 {
			connPlayer := &ConnectedPlayer{
				player:          game.PLAYER_X,
				conn:            conn,
				opponentUpdates: updatesO,
				myUpdates:       updatesX,
			}
			server.xConn = connPlayer
			go server.handleClient(connPlayer)
		} else {
			connPlayer := &ConnectedPlayer{
				player:          game.PLAYER_O,
				conn:            conn,
				opponentUpdates: updatesX,
				myUpdates:       updatesO,
			}
			server.yConn = connPlayer
			go server.handleClient(connPlayer)
		}

		server.numClients++
		if server.numClients == 2 {
			server.gameStarted = true
		}
	}
}

// handle a client: reply to every message with modified client message
func (server *Server) handleClient(connPlayer *ConnectedPlayer) {
	defer connPlayer.conn.Close()
	sendMessage(connPlayer, msg.HelloMessage{"Welcome to this tic tac toe server!", connPlayer.player})
	if server.gameStarted {
		sendMessage(connPlayer, msg.BoardMessage{server.board})
	} else {
		sendMessage(connPlayer, msg.WaitingMessage{})
		for !server.gameStarted {
			time.Sleep(300 * time.Millisecond)
		}
		sendMessage(connPlayer, msg.BoardMessage{server.board})
	}
	clientChan := make(chan interface{}, 0)
	go readClient(connPlayer, clientChan)
	for {
		select {
		case clientMessage := <-clientChan:
			server.handleMessage(connPlayer, clientMessage)
		case <-connPlayer.opponentUpdates:
			// todo: read a string from opponent updates, and dispatch on it
			// handle disconnected opponent gracefuly (add Exit Message)
			sendMessage(connPlayer, msg.BoardMessage{server.board})
		}
	}
}

// readClient reads messages from given player connection and puts them
// on messages channel
func readClient(connPlayer *ConnectedPlayer, messages chan<- interface{}) {
	reader := bufio.NewReader(connPlayer.conn)
	for {
		message, err := readMessage(reader)
		if err != nil {
			log.Printf("Error reading client message: %s\n", err)
			os.Exit(1)
		}
		messages <- message
	}
}

// read one client message data from the given reader, parse it
// and return as a message struct
func readMessage(reader *bufio.Reader) (interface{}, error) {
	data, err := reader.ReadString('\n')
	if err == io.EOF {
		return nil, fmt.Errorf("client disconnected")
	}
	if err != nil {
		return nil, fmt.Errorf("error reading from client %s", err)
	}
	data = strings.Trim(data, "\n")
	message, err := msg.UnmarshalMessage(data)
	if err != nil {
		return nil, fmt.Errorf("error when parsing client message: %s", err)
	}
	return message, nil
}

func (server *Server) handleMessage(connPlayer *ConnectedPlayer, message interface{}) {
	if !server.gameStarted {
		// ignore client messages until the game has started
		return
	}
	switch message := message.(type) {
	case msg.MoveMessage:
		server.gameLock.Lock()
		err := server.board.MakeMove(connPlayer.player, message.X, message.Y)
		server.gameLock.Unlock()
		if err != nil {
			sendMessage(connPlayer, msg.ErrorMessage{err.Error()})
		} else {
			sendMessage(connPlayer, msg.BoardMessage{server.board})
			connPlayer.myUpdates <- true
		}
	default:
		log.Printf("Unsupported message type: %T", message)
	}
}

// todo check if this is a blocking call
func sendMessage(connPlayer *ConnectedPlayer, message interface{}) {
	data, err := msg.MarshalMessage(message)
	if err != nil {
		log.Printf("Error marshaling message: %s\n", err)
		return
	}
	fmt.Fprintln(connPlayer.conn, data)
}