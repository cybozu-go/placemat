package virtualbmc

import (
	"github.com/cybozu-go/log"
)

type bmcUserHolder struct {
	users map[string]*bmcUser
}

type bmcUser struct {
	Username string
	Password string
}

func newBMCUserHolder() *bmcUserHolder {
	return &bmcUserHolder{users: make(map[string]*bmcUser)}
}

func (b *bmcUserHolder) addBMCUser(name string, password string) {
	newUser := &bmcUser{
		Username: name,
		Password: password,
	}
	b.users[name] = newUser
	log.Info("BMC USer: Add user", map[string]interface{}{"user": name})
}

func (b *bmcUserHolder) getBMCUser(name string) (*bmcUser, bool) {
	obj, ok := b.users[name]

	return obj, ok
}
