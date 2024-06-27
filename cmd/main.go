package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	post "tz-golang-EM/internal/storage/postgre"

	"github.com/gin-gonic/gin"
)

type People struct {
	Surname    string `json:"surname"`
	Name       string `json:"name"`
	Patronymic string `json:"patronymic"`
	Address    string `json:"address"`
}

func main() {
	post.InitDB()
	post.RunMigrations()

	r := gin.Default()

	r.GET("/users", getUsers)
	r.GET("/users/:id/tasks", getUserTasks)
	r.POST("/users", createUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)
	r.POST("/users/:id/tasks/start", startTask)
	r.POST("/users/:id/tasks/end", endTask)

	r.Run(":8080")
}

func getUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	users, err := post.GetUsers(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func getUserTasks(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))
	startDate, _ := time.Parse("2006-01-02", c.Query("startDate"))
	endDate, _ := time.Parse("2006-01-02", c.Query("endDate"))

	tasks, err := post.GetUserTasks(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func createUser(c *gin.Context) {
	var newUser post.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract passport series and number
	passportParts := strings.Split(newUser.PassportNumber, " ")
	if len(passportParts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid passport number format"})
		return
	}
	passportSerie := passportParts[0]
	passportNumber := passportParts[1]

	// Request additional info from external API
	info, err := getPeopleInfo(passportSerie, passportNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newUser.Surname = info.Surname
	newUser.Name = info.Name
	newUser.Patronymic = info.Patronymic
	newUser.Address = info.Address
	newUser.CreatedAt = time.Now()

	if err := post.CreateUser(&newUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newUser)
}

func getPeopleInfo(passportSerie, passportNumber string) (People, error) {
	url := fmt.Sprintf("http://external-api-url/info?passportSerie=%s&passportNumber=%s", passportSerie, passportNumber)
	resp, err := http.Get(url)
	if err != nil {
		return People{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return People{}, fmt.Errorf("failed to fetch people info, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return People{}, err
	}

	var info People
	if err := json.Unmarshal(body, &info); err != nil {
		return People{}, err
	}

	return info, nil
}

func updateUser(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	var updateUser post.User
	if err := c.ShouldBindJSON(&updateUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := post.UpdateUser(userID, &updateUser); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, updateUser)
}

func deleteUser(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	if err := post.DeleteUser(userID); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func startTask(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))

	var newTask post.Task
	if err := c.ShouldBindJSON(&newTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newTask.UserID = userID
	newTask.StartedAt = time.Now()

	if err := post.StartTask(&newTask); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newTask)
}

func endTask(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Param("id"))
	taskID, _ := strconv.Atoi(c.Query("taskID"))

	task, err := post.EndTask(userID, taskID)
	if err != nil {
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}
