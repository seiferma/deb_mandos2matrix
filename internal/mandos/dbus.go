package mandos

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

type ClientStatusHandler func(string)
type ClientStatusHandlerWithParam func(string, string)

type MandosCtl struct {
	con              *dbus.Conn
	sigChannel       chan *dbus.Signal
	onClientEnabled  ClientStatusHandler
	onClientDisabled ClientStatusHandler
	onClientRejected ClientStatusHandlerWithParam
}

type Client struct {
	Path          string
	Name          string
	Enabled       bool
	LastCheckedOk string
}

func NewMandosCtl() (*MandosCtl, error) {
	// establish connection
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	// create dbus signal channel
	c := make(chan *dbus.Signal, 50)
	conn.Signal(c)

	// register signal matchers for clients
	err = conn.AddMatchSignal(dbus.WithMatchInterface("se.recompile.Mandos.Client"))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Failed to register signal matcher for clients: %w", err)
	}

	// default dummy handlers
	onClientEnabled := func(clientPath string) {
		log.Printf("client %s enabled", clientPath)
	}
	onClientDisabled := func(clientPath string) {
		log.Printf("client %s disabled", clientPath)
	}
	onClientRejected := func(clientPath, reason string) {
		log.Printf("client %s rejected because: %s", clientPath, reason)
	}

	// create control struct
	ctl := MandosCtl{
		con:              conn,
		sigChannel:       c,
		onClientEnabled:  onClientEnabled,
		onClientDisabled: onClientDisabled,
		onClientRejected: onClientRejected,
	}

	// async processing of all received signals
	go func() {
		for true {
			signal := <-c
			ctl.handleSignal(signal)
		}

	}()

	return &ctl, nil
}

func (c *MandosCtl) Close() error {
	close(c.sigChannel)
	return c.con.Close()
}

func (c *MandosCtl) SetOnClientEnabledHandler(handler ClientStatusHandler) {
	c.onClientEnabled = handler
}

func (c *MandosCtl) SetOnClientDisabledHandler(handler ClientStatusHandler) {
	c.onClientDisabled = handler
}

func (c *MandosCtl) SetOnClientRejectedHandler(handler ClientStatusHandlerWithParam) {
	c.onClientRejected = handler
}

func (c *MandosCtl) LookupClientPathByName(name string) (string, error) {
	clients, err := c.GetClients()
	if err != nil {
		return "", err
	}
	for _, client := range clients {
		if client.Name == name {
			return client.Path, nil
		}
	}
	return "", fmt.Errorf("could not find client with name \"%s\"", name)
}

func (c *MandosCtl) LookupClientNameByPath(path string) (string, error) {
	clients, err := c.GetClients()
	if err != nil {
		return "", err
	}
	for _, client := range clients {
		if client.Path == path {
			return client.Name, nil
		}
	}
	return "", fmt.Errorf("could not find client with name \"%s\"", path)
}

func (c *MandosCtl) GetClientEnabled(client string) (bool, error) {
	obj := c.con.Object("se.recompile.Mandos", dbus.ObjectPath(client))
	var status bool
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, "se.recompile.Mandos.Client", "Enabled").Store(&status)
	if err != nil {
		return false, fmt.Errorf("Failed to set the Enabled flag of client %s to %v: %w", client, status, err)
	}
	return status, nil
}

func (c *MandosCtl) SetClientEnabled(client string, status bool) error {
	obj := c.con.Object("se.recompile.Mandos", dbus.ObjectPath(client))
	err := obj.Call("org.freedesktop.DBus.Properties.Set", 0, "se.recompile.Mandos.Client", "Enabled", dbus.MakeVariant(status)).Err
	if err != nil {
		return fmt.Errorf("Failed to set the Enabled flag of client %s to %v: %w", client, status, err)
	}
	return nil
}

func (c *MandosCtl) GetClients() ([]Client, error) {
	// discover clients
	obj := c.con.Object("se.recompile.Mandos", "/")
	var clients map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&clients)
	if err != nil {
		return nil, fmt.Errorf("Failed to get clients: %w", err)
	}
	foundClients := make([]Client, 0)
	for clientPath, clientValues := range clients {
		for interfaceName := range clientValues {
			if interfaceName == "se.recompile.Mandos.Client" {
				properties := clientValues["se.recompile.Mandos.Client"]
				client := Client{
					Path:          string(clientPath),
					Name:          properties["Name"].Value().(string),
					Enabled:       properties["Enabled"].Value().(bool),
					LastCheckedOk: properties["LastCheckedOK"].Value().(string),
				}
				foundClients = append(foundClients, client)
			}
		}
	}
	return foundClients, nil
}

func (c *MandosCtl) handleSignal(signal *dbus.Signal) {
	clientPath := string(signal.Path)
	switch signal.Name {
	case "se.recompile.Mandos.Client.PropertyChanged":
		propertyName := signal.Body[0].(string)
		propertyValue := signal.Body[1].(dbus.Variant)
		c.handlePropertyChange(clientPath, propertyName, propertyValue)
	case "se.recompile.Mandos.Client.Rejected":
		reason := signal.Body[0].(string)
		c.handleRejected(clientPath, reason)
	default:
		// do nothing
		//log.Println(signal)
	}
}

func (c *MandosCtl) handlePropertyChange(client string, propertyName string, propertyValue dbus.Variant) {
	switch propertyName {
	case "Enabled":
		if propertyValue.Value().(bool) {
			c.handleEnabled(client)
		} else {
			c.handleDisabled(client)
		}
	default:
		// do nothing
		// log.Printf("Property of client %s changed: %s = %s", client, propertyName, propertyValue)
	}
}

func (c *MandosCtl) handleEnabled(client string) {
	c.onClientEnabled(client)
}

func (c *MandosCtl) handleDisabled(client string) {
	c.onClientDisabled(client)
}

func (c *MandosCtl) handleRejected(client, reason string) {
	c.onClientRejected(client, reason)
}
