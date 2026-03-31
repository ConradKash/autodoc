//go:generate autodoc-gen -router=gin

package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type UserRepository interface {
	GetAll() []User
	Create(user User) User
}

type InMemoryUserRepo struct {
	users []User
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		users: []User{},
	}
}

func (repo *InMemoryUserRepo) GetAll() []User {
	return repo.users
}

func (repo *InMemoryUserRepo) Create(user User) User {
	repo.users = append(repo.users, user)
	return user
}

func main() {
	r := gin.Default()

	// Use the interface type directly
	var repo UserRepository = NewInMemoryUserRepo()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.GET("/hello/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(http.StatusOK, gin.H{"message": "Hello, " + name + "!"})
	})

	// List all users
	r.GET("/users", func(c *gin.Context) {
		users := repo.GetAll()
		c.JSON(http.StatusOK, users)
	})

	// Create a new user
	r.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		created := repo.Create(user)
		c.JSON(http.StatusCreated, created)
	})

	if err := r.Run(); err != nil {
		panic(err)
	}
}
