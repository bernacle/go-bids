package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageKind int

const (
	//Requests
	PlaceBid MessageKind = iota

	//Ok/Success
	SuccessfullyPlacedBid

	//Errors
	FailedToPlaceBid
	InvalidJSON

	//Info
	NewBidPlaced
	AuctionFinished
)

type Message struct {
	Message string      `json:"message,omitempty"`
	Amount  float64     `json:"amount,omitempty"`
	Kind    MessageKind `json:"kind"`
	UserID  uuid.UUID   `json:"user_id,omitempty"`
}

type AuctionLobby struct {
	sync.Mutex
	Rooms map[uuid.UUID]*AuctionRoom
}

type AuctionRoom struct {
	Id         uuid.UUID
	Context    context.Context
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Clients    map[uuid.UUID]*Client

	BidsService BidsService
}

func (r *AuctionRoom) registerClient(c *Client) {
	slog.Info("New user connected", "Client", c)
	r.Clients[c.UserID] = c
}

func (r *AuctionRoom) unregisterClient(c *Client) {
	slog.Info("User disconnected", "Client", c)
	delete(r.Clients, c.UserID)
}

func (r *AuctionRoom) brodcastMessage(m Message) {
	slog.Info("New message received", "RoomID", r.Id, "message", m, "user_id", m.UserID)
	switch m.Kind {
	case PlaceBid:
		bid, error := r.BidsService.PlaceBid(r.Context, r.Id, m.UserID, m.Amount)
		if error != nil {
			if errors.Is(error, ErrBidIsTooLow) {
				if client, ok := r.Clients[m.UserID]; ok {
					client.Send <- Message{Kind: FailedToPlaceBid, Message: ErrBidIsTooLow.Error()}
				}
				return
			}
		}

		if client, ok := r.Clients[m.UserID]; ok {
			client.Send <- Message{
				Kind:    SuccessfullyPlacedBid,
				Message: "Your bid was placed with success",
			}
		}

		for id, client := range r.Clients {
			newBidMessage := Message{Kind: NewBidPlaced, Message: "A new bid was placed", Amount: bid.BidAmount}
			if id == m.UserID {
				continue
			}
			client.Send <- newBidMessage
		}
	case InvalidJSON:
		client, ok := r.Clients[m.UserID]
		if !ok {
			slog.Info("Client not found", "user_id", m.UserID)
			return
		}
		client.Send <- m
	}
}

func (r *AuctionRoom) Run() {
	slog.Info("Auction has started", "auctionId", r.Id)
	defer func() {
		close(r.Broadcast)
		close(r.Register)
		close(r.Unregister)
	}()
	for {
		select {
		case client := <-r.Register:
			r.registerClient(client)
		case client := <-r.Unregister:
			r.unregisterClient(client)
		case message := <-r.Broadcast:
			r.brodcastMessage(message)
		case <-r.Context.Done():
			slog.Info("Auction has ended", "auctionId", r.Id)
			for _, client := range r.Clients {
				client.Send <- Message{Kind: AuctionFinished, Message: "Auction has been finished"}
			}
			return
		}
	}
}

func NewAuctionRoom(ctx context.Context, id uuid.UUID, BidsService BidsService) *AuctionRoom {
	return &AuctionRoom{
		Id:          id,
		Broadcast:   make(chan Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Clients:     make(map[uuid.UUID]*Client),
		Context:     ctx,
		BidsService: BidsService,
	}
}

type Client struct {
	Room   *AuctionRoom
	Conn   *websocket.Conn
	Send   chan Message
	UserID uuid.UUID
}

func NewClient(room *AuctionRoom, conn *websocket.Conn, userId uuid.UUID) *Client {
	return &Client{
		Room:   room,
		Conn:   conn,
		Send:   make(chan Message, 512),
		UserID: userId,
	}
}

const (
	maxMessageSize = 512
	readDeadline   = 60 * time.Second
)

func (c *Client) ReadEventLoop() {
	defer func() {
		c.Room.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	for {
		var m Message
		m.UserID = c.UserID
		err := c.Conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Unexpected close error", "error", err)
				return
			}
			c.Room.Broadcast <- Message{
				Kind:    InvalidJSON,
				Message: "this message should be a valid JSON",
				UserID:  m.UserID,
			}
		}

		c.Room.Broadcast <- m
	}
}
