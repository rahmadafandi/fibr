package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRedis(t *testing.T) {
	// setup redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	r = New(client)

	// setup gorm
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	assert.NoError(t, err)

	// run migration
	err = db.AutoMigrate(&User{})
	assert.NoError(t, err)

	// create user
	user := User{
		Name: "test",
	}
	err = db.Create(&user).Error
	assert.NoError(t, err)

	// get user from redis
	var userFromRedis User
	err = r.GetGormResult(context.Background(), "user", &userFromRedis)
	assert.Error(t, err)

	// get user from db
	var userFromDB User
	gormResult := NewGormResult(db)
	err = gormResult.Find(context.Background(), "user", &userFromDB, 10*time.Second, func() (interface{}, error) {
		var user User
		err := db.First(&user).Error
		return user, err
	})
	assert.NoError(t, err)
	assert.Equal(t, user.Name, userFromDB.Name)

	// get user from redis
	err = r.GetGormResult(context.Background(), "user", &userFromRedis)
	assert.NoError(t, err)
	assert.Equal(t, user.Name, userFromRedis.Name)
}

type User struct {
	gorm.Model
	Name string
}
