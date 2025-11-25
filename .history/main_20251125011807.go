package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

type Expenses struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
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


func AddExpens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := getUserIDFromContext(r)

	var req TodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if len(req.Title) > 255 {
		respondError(w, http.StatusBadRequest, "Title must be less than 255 characters")
		return
	}

	now := time.Now()

	res, err := db.Exec(
		"INSERT INTO todos (user_id, title, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		userID, req.Title, req.Description, now, now,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create todo")
		return
	}

	id64, _ := res.LastInsertId()
	id := int(id64)

	todo := Todo{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	respondJSON(w, http.StatusCreated, todo)
}


func main() {
	connectDB()

	r := gin.Default()

	r.POST("/signup", SignUp)

	r.Run(":9090")
}
