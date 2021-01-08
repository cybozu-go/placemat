package virtualbmc

import (
	"github.com/cybozu-go/log"
)

// BMCUser holds BMCUsers
type BMCUserHolder struct {
	users map[string]*BMCUser
}

// BMCUser represents user name and password
type BMCUser struct {
	Username string
	Password string
}

// NewBMCUserHolder creates a BMCUser
func NewBMCUserHolder() *BMCUserHolder {
	return &BMCUserHolder{users: make(map[string]*BMCUser)}
}

// AddBMCUser add a user to the holder
func (b *BMCUserHolder) AddBMCUser(name string, password string) {
	newUser := &BMCUser{
		Username: name,
		Password: password,
	}
	b.users[name] = newUser
	log.Info("BMC USer: Add user", map[string]interface{}{"user": name})
}

// RemoveBMCUser removes the user specified from the holder
func (b *BMCUserHolder) RemoveBMCUser(name string) {
	_, ok := b.users[name]

	if ok {
		delete(b.users, name)
	}
}

// GetBMCUser get the user specified from the holder
func (b *BMCUserHolder) GetBMCUser(name string) (*BMCUser, bool) {
	obj, ok := b.users[name]

	return obj, ok
}
