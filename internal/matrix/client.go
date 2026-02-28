package matrix

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MatrixClient struct {
	client        *mautrix.Client
	roomId        string
	ctx           context.Context
	closeFn       func() error
	callbacks     []MessageReceivedCallback
	running       bool
	runningMutext sync.Mutex
}

type MessageReceivedCallback func(ctx context.Context, roomId string, senderId string, message string)

func GenerateRuntimeConfig(initialConfig InitialConfig) (*RuntimeConfig, error) {
	// create matrix client
	client, err := mautrix.NewClient(initialConfig.HomeserverURL, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to create Matrix client: %w", err)
	}

	// login to get account information and credentials
	resp, err := client.Login(context.Background(), &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			User: initialConfig.UserName,
			Type: mautrix.IdentifierTypeUser,
		},
		Password:         initialConfig.Password,
		StoreCredentials: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not login to matrix server: %w", err)
	}

	// define runtime configuration
	return &RuntimeConfig{
		BaseConfig:  initialConfig.BaseConfig,
		AccessToken: resp.AccessToken,
		UserId:      resp.UserID.String(),
		DeviceId:    resp.DeviceID.String(),
	}, nil
}

func NewMatrixClientByRuntimeConfig(runtimeConfig *RuntimeConfig) (*MatrixClient, error) {
	// create matrix client
	client, err := mautrix.NewClient(runtimeConfig.HomeserverURL, id.UserID(runtimeConfig.UserId), runtimeConfig.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Matrix client: %w", err)
	}
	client.DeviceID = id.DeviceID(runtimeConfig.DeviceId)

	// initialize and register in memory state store
	// TODO: find implementation not requiring cgo
	client.StateStore = mautrix.NewMemoryStateStore()

	// initialize and register syncer
	syncer := mautrix.NewDefaultSyncer()
	client.Syncer = syncer

	// create result object
	mc := MatrixClient{
		client:    client,
		roomId:    runtimeConfig.RoomID,
		callbacks: make([]MessageReceivedCallback, 0),
	}
	// register mautrix sync and event handlers
	syncer.OnSync(client.DontProcessOldEvents)
	syncer.OnEvent(client.StateStoreSyncHandler)

	// register callback listener
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		for _, callback := range mc.callbacks {
			select {
			case <-ctx.Done():
				log.Println("message callback handling aborted because context is aborted")
				return
			default:
				callback(ctx, evt.RoomID.String(), evt.Sender.String(), evt.Content.AsMessage().Body)
			}
		}
	})

	// // debugging
	// syncer.OnEvent(func(ctx context.Context, evt *event.Event) {
	// 	log.Printf("%v", evt)
	// })

	return &mc, nil
}

func (mc *MatrixClient) Close() error {
	return mc.closeFn()
}

func (mc *MatrixClient) Run(ctx context.Context) error {
	err := mc.setToRunning()
	if err != nil {
		return err
	}
	defer func() {
		mc.runningMutext.Lock()
		defer mc.runningMutext.Unlock()
		mc.running = false
		mc.closeFn = func() error { return nil }
	}()

	// define sync context
	syncCtx, cancelSync := context.WithCancel(ctx)

	// define close function
	mc.closeFn = func() error {
		cancelSync()
		return nil
	}

	// define and run sync
	return func() error {
		err := mc.client.SyncWithContext(syncCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("syncing is broken: %w", err)
		}
		return nil
	}()
}

func (mc *MatrixClient) setToRunning() error {
	mc.runningMutext.Lock()
	defer mc.runningMutext.Unlock()
	if mc.running {
		return fmt.Errorf("matrix client is already running")
	}
	mc.running = true
	return nil
}

func (mc *MatrixClient) SendMessage(message string) error {
	mc.runningMutext.Lock()
	defer mc.runningMutext.Unlock()
	if !mc.running {
		return fmt.Errorf("matrix client is not running")
	}
	_, err := mc.client.SendText(context.Background(), id.RoomID(mc.roomId), message)
	return err
}

func (mc *MatrixClient) RegisterCallback(callback MessageReceivedCallback) {
	mc.callbacks = append(mc.callbacks, callback)
}

func (mc *MatrixClient) GetRoomInfo(ctx context.Context) (*mautrix.PublicRoomInfo, error) {
	resp, err := mc.client.GetRoomSummary(ctx, mc.roomId)
	if err != nil {
		return nil, err
	}
	return &resp.PublicRoomInfo, nil
}
