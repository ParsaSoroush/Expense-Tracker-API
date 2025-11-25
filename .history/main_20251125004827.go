package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)


type Blog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var db *gorm.DB

func connectDB() {
	dsn := "expense_tracker:Expense_Tracker$1234@tcp(127.0.0.1:3306)/expense_database?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ Failed to connect to database")
	}

	if err := db.AutoMigrate(&Blog{}); err != nil {
		panic("❌ Failed to migrate table")
	}

	println("✅ Database connected successfully")
}


func SignUp(c *gin.Context) {
	type User struct {
		ID       uint   `gorm:"primaryKey"`
		Email    string `gorm:"unique"`
		Password string
	}

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Get request body
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}

	// Connect DB
	db, err := gorm.Open(mysql.Open("username:password@tcp(localhost:3306)/database"), &gorm.Config{})
	if err != nil {
		c.JSON(500, gin.H{"message": "Database connection failed"})
		return
	}
	db.AutoMigrate(&User{})

	// Check if user exists
	var existing User
	db.Where("email = ?", req.Email).First(&existing)
	if existing.ID != 0 {
		c.JSON(409, gin.H{"message": "Email already exists"})
		return
	}

	// Hash password
	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	// Create user
	user := User{Name: req.Name, Email: req.Email, Password: string(hashed)}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to create user"})
		return
	}

	// Generate JWT Token
	secret := []byte("YOUR_SECRET_KEY")
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString(secret)

	// Response
	c.JSON(200, gin.H{
		"message": "User successfully registered",
		"token":   signedToken,
	})
}



func main() {
	connectDB()
	r := gin.Default()

	r.Run(":9090")
}
