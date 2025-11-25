package main

import (
	"strings"
	"time"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Expenses struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Description string    `json:"description"`
	Amount      string    `json:"amount"` // stored like "50.00$"
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

	if err := db.AutoMigrate(&Expenses{}); err != nil {
		panic("❌ Failed to migrate Expenses table")
	}

	println("✅ Database connected successfully")
}


func SignUp(c *gin.Context) {
	type User struct {
		ID       uint   `gorm:"primaryKey"`
		Email    string `gorm:"unique"`
		Password string
	}

	db.AutoMigrate(&User{})

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}

	var existing User
	db.Where("email = ?", req.Email).First(&existing)
	if existing.ID != 0 {
		c.JSON(409, gin.H{"message": "Email already exists"})
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := User{Email: req.Email, Password: string(hashed)}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to create user"})
		return
	}

	secret := []byte("SECRET_KEY")
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString(secret)

	c.JSON(200, gin.H{
		"message": "User successfully registered",
		"token":   signedToken,
	})
}

// ----------------------------------------
//           JWT AUTHENTICATION
// ----------------------------------------
func ValidateToken(tokenString string) (*jwt.Token, error) {
	secret := []byte("SECRET_KEY")
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
}

// ----------------------------------------
//            ADD EXPENSE
// ----------------------------------------
func AddExpense(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(401, gin.H{"message": "Missing Authorization header"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := ValidateToken(tokenString)
	if err != nil || !token.Valid {
		c.JSON(401, gin.H{"message": "Invalid or expired token"})
		return
	}

	var req struct {
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
	}

	if err := c.BindJSON(&req); err != nil || req.Description == "" {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	amountFormatted := fmt.Sprintf("%.2f$", req.Amount)

	expense := Expenses{
		Description: req.Description,
		Amount:      amountFormatted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(&expense).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to add expense"})
		return
	}

	c.JSON(201, expense)
}


func main() {
	connectDB()

	r := gin.Default()
	r.POST("/signup", SignUp)
	r.POST("/add-expense", AddExpense)

	r.Run(":9090")
}
