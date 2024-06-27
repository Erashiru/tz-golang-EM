package postgre

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	gre "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID             int       `gorm:"primaryKey" json:"id"`
	PassportNumber string    `json:"passportNumber"`
	CreatedAt      time.Time `json:"createdAt"`
	Surname        string    `json:"surname"`
	Name           string    `json:"name"`
	Patronymic     string    `json:"patronymic"`
	Address        string    `json:"address"`
}

type Task struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	UserID      int       `json:"userId"`
	Description string    `json:"description"`
	StartedAt   time.Time `json:"startedAt"`
	EndedAt     time.Time `json:"endedAt"`
}

var db *gorm.DB

func InitDB() {
	var err error
	dsn := "host=localhost user=yerassyl password=12345678 dbname=users port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
}

func RunMigrations() {
	dbSQL, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	driver, err := gre.WithInstance(dbSQL, &gre.Config{})
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres", driver)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
}

func CreateUser(newUser *User) error {
	return db.Create(newUser).Error
}

func UpdateUser(userID int, updateUser *User) error {
	if db.Model(&User{}).Where("id = ?", userID).Updates(updateUser).RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func DeleteUser(userID int) error {
	if db.Delete(&User{}, userID).RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func GetUsers(page, pageSize int) ([]User, error) {
	var users []User
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func GetUserTasks(userID int, startDate, endDate time.Time) ([]Task, error) {
	var tasks []Task
	if err := db.Where("user_id = ? AND started_at >= ? AND ended_at <= ?", userID, startDate, endDate).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func StartTask(newTask *Task) error {
	return db.Create(newTask).Error
}

func EndTask(userID, taskID int) (*Task, error) {
	var task Task
	if db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).RowsAffected == 0 {
		return nil, fmt.Errorf("task not found")
	}
	task.EndedAt = time.Now()
	if err := db.Save(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
