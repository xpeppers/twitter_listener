package listener

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/fiorix/go-redis/redis"
)

// NewStore creates a new instance of Redis-based Store.
// The URL should not spcify any db number, e.g. "127.0.0.1:6379".
// Internally, connection URL is then constructed from dbURL and a database number.
func NewStore(dbURL string, appDB, userDB int) Store {
	return &redisStore{dbURL, appDB, userDB}
}

// redisStore is Redis-based implementation of Store
type redisStore struct {
	dbURL  string
	appDB  int
	userDB int
}

func (s *redisStore) ListAppNames() ([]string, error) {
	return s.newClient(s.appDB).Keys("*")
}

func (s *redisStore) GetApp(name string) (app *Application, getErr error) {
	jsonApp, err := s.newClient(s.appDB).Get(name)
	if err != nil {
		return nil, err
	}
	if jsonApp == "" {
		return nil, fmt.Errorf("App %q not found", name)
	}

	app = &Application{}
	getErr = json.Unmarshal([]byte(jsonApp), app)
	return
}

func (s *redisStore) ListTwitterIDs(name string) ([]string, error) {
	users := s.newClient(s.userDB)

	userIDs, err := users.SMembers("customer:" + name)
	if err != nil {
		return nil, err
	}

	twitterIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		jsonUser, getErr := users.Get(userID)
		if getErr != nil {
			log.Printf("ERROR getting user %q of app %q", userID, name)
			continue
		}

		user := &User{}
		if json.Unmarshal([]byte(jsonUser), user) != nil {
			log.Printf("ERROR parsing JSON user data of %q (%s)", userID, name)
			continue
		}
		if user.Metadata == nil {
			log.Printf("WARNING: no metadata for user %q (%s)", userID, name)
			continue
		}

		// TODO: find a better way to get twitter ID,
		// e.g. store it in user.Metadata["twitter.user.id"]
		if _, ok := user.Metadata["twitter.user.screenName"]; ok {
			twitterIDs = append(twitterIDs, user.Username)
		}
	}

	return twitterIDs, nil
}

func (s *redisStore) Subscribe(topic string, msg chan redis.PubSubMessage, stop chan bool) error {
	return s.newClient(s.appDB).Subscribe(topic, msg, stop)
}

func (s *redisStore) connectionURL(db int) string {
	return fmt.Sprintf("%s db=%d", s.dbURL, db)
}

func (s *redisStore) newClient(db int) *redis.Client {
	return redis.New(s.connectionURL(db))
}
