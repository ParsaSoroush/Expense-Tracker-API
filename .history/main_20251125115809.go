package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Expenses struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Description string    `json:"description"`
	Amount      string    `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Global DB
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

	// Response
	c.JSON(200, gin.H{
		"message": "User successfully registered",
		"token":   signedToken,
	})
}


func ValidateToken(tokenString string) (*jwt.Token, error) {
	secret := []byte("SECRET_KEY")
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
}

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

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	desc, _ := body["description"].(string)
	if desc == "" {
		if v, ok := body["Description"].(string); ok {
			desc = v
		}
	}
	if desc == "" {
		c.JSON(400, gin.H{"message": "description is required"})
		return
	}

	// amount: support number (float64/int) or string
	var amountFloat float64
	found := false
	tryKeys := []string{"amount", "Amount"}
	for _, key := range tryKeys {
		if v, ok := body[key]; ok {
			switch val := v.(type) {
			case float64:
				amountFloat = val
				found = true
			case int:
				amountFloat = float64(val)
				found = true
			case int64:
				amountFloat = float64(val)
				found = true
			case string:
				if parsed, err := strconv.ParseFloat(val, 64); err == nil {
					amountFloat = parsed
					found = true
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		c.JSON(400, gin.H{"message": "amount is required and must be a number"})
		return
	}

	amountFormatted := fmt.Sprintf("%.2f$", amountFloat)

	expense := Expenses{
		Description: desc,
		Amount:      amountFormatted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(&expense).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to add expense"})
		return
	}

	// return the newly created expense``
	c.JSON(201, expense)
}


func UpdateExpense(c *gin.Context) {
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

	// Get the ID from URL
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"message": "Expense ID is required"})
		return
	}

	// Parse ID to uint
	expenseID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid expense ID"})
		return
	}

	// Find the existing expense
	var expense Expenses
	result := db.First(&expense, expenseID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"message": "Expense not found"})
		} else {
			c.JSON(500, gin.H{"message": "Database error"})
		}
		return
	}

	// Parse request body
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	// Update description if provided
	if desc, ok := body["description"].(string); ok && desc != "" {
		expense.Description = desc
	} else if desc, ok := body["Description"].(string); ok && desc != "" {
		expense.Description = desc
	}

	// Update amount if provided
	var amountFloat float64
	found := false
	tryKeys := []string{"amount", "Amount"}
	for _, key := range tryKeys {
		if v, ok := body[key]; ok {
			switch val := v.(type) {
			case float64:
				amountFloat = val
				found = true
			case int:
				amountFloat = float64(val)
				found = true
			case int64:
				amountFloat = float64(val)
				found = true
			case string:
				if parsed, err := strconv.ParseFloat(val, 64); err == nil {
					amountFloat = parsed
					found = true
				}
			}
			if found {
				break
			}
		}
	}

	if found {
		expense.Amount = fmt.Sprintf("%.2f$", amountFloat)
	}

	expense.UpdatedAt = time.Now()

	if err := db.Save(&expense).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to update expense"})
		return
	}

	c.JSON(200, expense)
}

func GetAllExpenses(c *gin.Context) {
	var expenses []Expenses

	result := db.Find(&expenses)
	if result.Error != nil {
		c.JSON(500, gin.H{"message": "Failed to fetch data"})
		return
	}

	c.JSON(200, expenses)
}


func DeleteExpense(c *gin.Context) {
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

	// Get ID from URL
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"message": "Expense ID is required"})
		return
	}

	// Convert ID string → uint
	expenseID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid expense ID"})
		return
	}

	// Check if the expense exists
	var expense Expenses
	result := db.First(&expense, expenseID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"message": "Expense not found"})
		} else {
			c.JSON(500, gin.H{"message": "Database error"})
		}
		return
	}

	// Delete the record
	if err := db.Delete(&expense).Error; err != nil {
		c.JSON(500, gin.H{"message": "Failed to delete expense"})
		return
	}

	c.JSON(200, gin.H{"message": "Expense deleted successfully"})
}



func main() {
	connectDB()

	r := gin.Default()
	r.POST("/signup", SignUp)
	r.POST("/expenses", AddExpense)
	r.PUT("/expenses/:id", UpdateExpense)
	r.GET("/expenses", GetAllExpenses)
	r.DELETE("/expenses")

	r.Run(":9090")
}
