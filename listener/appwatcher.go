package listener

import (
	"log"

	"github.com/fiorix/go-redis/redis"
)

type AppWatcher struct {
	topic     string
	listeners []Listener
	store     Store
}

func NewAppWatcher(topic string, listeners []Listener, store Store) *AppWatcher {
	return &AppWatcher{topic, listeners, store}
}

func (a *AppWatcher) Watch(qc chan *Tweet, errc chan int) error {
	msgc := make(chan redis.PubSubMessage)
	stopc := make(chan bool, 1)
	if err := a.store.Subscribe(a.topic, msgc, stopc); err != nil {
		return err
	}

	go a.loop(msgc, qc, errc)
	a.waitAll(errc)

	stopc <- true
	return nil
}

func (a *AppWatcher) loop(msgc chan redis.PubSubMessage, qc chan *Tweet, errc chan int) {
	for {
		msg := <-msgc
		if msg.Error != nil {
			log.Printf("Message channel error %s", msg.Error)
			continue
		}
		appName := msg.Value
		var listener Listener
		for _, l := range a.listeners {
			if l.Name() == appName {
				listener = l
				break
			}
		}
		if listener == nil {
			app, err := a.store.GetApp(appName)
			if err != nil {
				log.Printf("Message channel error %s", err)
				continue
			}
			userIDs, userErr := a.store.ListTwitterIDs(appName)
			if userErr != nil {
				log.Printf("Message channel error %s", userErr)
				continue
			}
			if len(userIDs) == 0 {
				log.Printf("No users found for app %q. Exiting.", appName)
				continue
			}
			listener = NewListener(app, userIDs, qc)
			a.listeners = append(a.listeners, listener)
		}
		listener.Restart(errc)
	}

}

// waitAll reads from the
func (a *AppWatcher) waitAll(c chan int) {
	count := len(a.listeners)
	for {
		status := <-c
		log.Printf("Listener exited with status %d", status)
		if count -= 1; count == 0 {
			log.Printf("Quit application")
			break
		}
	}
}
