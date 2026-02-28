package logic

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/seiferma/mandos2mqtt/internal/mandos"
)

const rejectQuietPeriodInSeconds = 30

type Mandos interface {
	LookupClientPathByName(name string) (string, error)
	LookupClientNameByPath(path string) (string, error)
	GetClientEnabled(client string) (bool, error)
	SetClientEnabled(client string, status bool) error
	GetClients() ([]mandos.Client, error)
}

type Logic struct {
	mandos        Mandos
	sendMessage   func(string)
	rejectBackoff map[string]time.Time
}

func CreateLogic(sendFn func(string), mandos Mandos) *Logic {
	return &Logic{
		mandos:        mandos,
		sendMessage:   sendFn,
		rejectBackoff: map[string]time.Time{},
	}
}

func (l *Logic) HandleClientStatusChange(ctx context.Context, clientPath string, status bool) {
	clientName, err := l.mandos.LookupClientNameByPath(clientPath)
	if err != nil {
		l.sendMessage(fmt.Sprintf("could not get client name for %s", clientPath))
	}
	l.sendMessage(fmt.Sprintf("client %s is now %s", clientName, getEnableWord(status)))
}

func (l *Logic) HandleClientRequestRejected(ctx context.Context, clientPath, reason string) {
	nextTime, ok := l.rejectBackoff[clientPath]
	if !ok {
		nextTime = time.Now().Add(-1 * time.Second)
	}

	if time.Now().After(nextTime) {
		l.rejectBackoff[clientPath] = time.Now().Add(rejectQuietPeriodInSeconds * time.Second)
		clientName, err := l.mandos.LookupClientNameByPath(clientPath)
		if err != nil {
			clientName = clientPath
		}
		l.sendMessage(fmt.Sprintf("client %s is requesting a key but is rejected (reason: %s)", clientName, reason))
	}
}

func (l *Logic) HandleChatMessage(ctx context.Context, senderId, message string) {
	words := strings.Fields(message)
	if len(words) < 1 {
		return
	}
	command := strings.ToLower(words[0])
	switch command {
	case "status":
		{
			l.handleStatus()
		}
	case "enable":
		if len(words) < 2 {
			l.sendUsage()
			return
		}
		clientName := words[1]
		l.handleStatusChange(clientName, true)
	case "disable":
		if len(words) < 2 {
			l.sendUsage()
			return
		}
		clientName := words[1]
		l.handleStatusChange(clientName, false)
	}
}

func (l *Logic) handleStatusChange(clientName string, enable bool) {
	clientPath, err := l.mandos.LookupClientPathByName(clientName)
	if err != nil {
		l.sendMessage(fmt.Sprintf("Could not find client \"%s\"", clientName))
		return
	}
	err = l.mandos.SetClientEnabled(clientPath, enable)
	if err != nil {
		l.sendMessage(fmt.Sprintf("Could not change status of client \"%s\": %v", clientName, err))
		return
	}
	clientStatus, err := l.mandos.GetClientEnabled(clientPath)
	if err != nil {
		l.sendMessage(fmt.Sprintf("Could not verify status of client \"%s\": %v", clientName, err))
		return
	}
	if clientStatus != enable {
		log.Printf("client status (%t) does not match set status (%t)", clientStatus, enable)
	}
}

func (l *Logic) handleStatus() {
	resultLines := make([]string, 0)
	resultLines = append(resultLines, "Name\tStatus\tLast Check OK")
	clients, err := l.mandos.GetClients()
	if err != nil {
		log.Panicf("could not get clients because of mandos: %v", err)
	}
	for _, client := range clients {
		resultLine := fmt.Sprintf("%s\t%s\t%s", client.Name, getEnableWord(client.Enabled), client.LastCheckedOk)
		resultLines = append(resultLines, resultLine)
	}
	l.sendMessage(strings.Join(resultLines, "\n"))
}

func (l *Logic) sendUsage() {
	l.sendMessage(`You can use the following commands:
	status\t\tprint status of all known clients
	enable <name>\t\tenables the client with the given name
	disable <name>\t\tdisabled the client with the given name`)
}

func getEnableWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
