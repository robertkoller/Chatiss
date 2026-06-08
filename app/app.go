package app

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application struct. It bridges the React frontend and the
// persistent Service backend.
type App struct {
	ctx     context.Context
	service *Service
}

func NewApp() *App {
	return &App{}
}

// Startup is called by Wails when the app window is ready.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Login creates and starts the background service. Called from the login screen.
func (a *App) Login(passphrase, username string) string {
	if a.service != nil {
		a.service.Stop()
	}
	svc, err := NewService(passphrase, username, func(name string, data ...any) {
		runtime.EventsEmit(a.ctx, name, data...)
	})
	if err != nil {
		return err.Error()
	}
	a.service = svc
	return ""
}

// GetContacts returns all stored contacts with online status.
func (a *App) GetContacts() ([]UIContact, error) {
	if err := a.requireService(); err != nil {
		return nil, err
	}
	return a.service.GetContacts()
}

// AddContact looks up a username on the STUN server and saves them as a contact.
func (a *App) AddContact(username string) string {
	if err := a.requireService(); err != nil {
		return err.Error()
	}
	if err := a.service.AddContact(username); err != nil {
		return err.Error()
	}
	return ""
}

// GetMessages returns the stored message history for a contact.
func (a *App) GetMessages(username string) ([]UIMessage, error) {
	if err := a.requireService(); err != nil {
		return nil, err
	}
	return a.service.GetMessages(username)
}

// Connect triggers a STUN lookup for the given contact. Call when opening a
// conversation so a live P2P session is established if the peer is online.
func (a *App) Connect(username string) {
	if a.service == nil {
		return
	}
	a.service.Connect(username)
}

// SendMessage sends a message to a contact (live P2P or via mailbox if offline).
func (a *App) SendMessage(username, text string) string {
	if err := a.requireService(); err != nil {
		return err.Error()
	}
	if err := a.service.SendMessage(username, text); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) requireService() error {
	if a.service == nil {
		return fmt.Errorf("not logged in")
	}
	return nil
}
