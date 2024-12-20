package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:root@tcp(localhost:3306)/img_editor")
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Could not establish connection: %v", err)
	}

	fmt.Println("Connected to MySQL database")
}
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type AuthService struct {
	SecretKey string
}

func (s *AuthService) ValidateToken(tokenString string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return false, err
	}
	return true, nil
}
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Missing username or password", http.StatusBadRequest)
		return
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		http.Error(w, "Could not hash password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Missing username or password", http.StatusBadRequest)
		return
	}

	var hashedPassword string
	var userID int
	err := db.QueryRow("SELECT password, id FROM users WHERE username = ?", username).Scan(&hashedPassword, &userID)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if !checkPasswordHash(password, hashedPassword) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"id":       userID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte("ac17c7ae33f56f6aafc550dce6f5b98134be4d96903c8fd5538a01ec4d2bb5e6"))
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    tokenString,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	log.Println(r.Cookie("session_token"))
	w.Header().Add("Hx-Redirect", "/protected/editor")

}
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	w.Header().Add("Hx-Redirect", "/login")
}
func getUserIDFromContext(ctx context.Context) (int, error) {
	claims, ok := ctx.Value("user").(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("could not extract claims from context")
	}

	userID, ok := claims["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("user ID not found in token claims")
	}

	return int(userID), nil
}

func authMiddleware(authService *AuthService) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenParts := strings.Split(authHeader, " ")
				if len(tokenParts) == 2 && strings.ToLower(tokenParts[0]) == "bearer" {
					token = tokenParts[1]
				} else {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
			} else {
				cookie, err := r.Cookie("session_token")
				if err != nil {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				token = cookie.Value
			}

			isValid, err := authService.ValidateToken(token)
			if !isValid || err != nil {
				log.Printf("Token validation failed: %v", err)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			parsedToken, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				return []byte(authService.SecretKey), nil
			})

			if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
				ctx := context.WithValue(r.Context(), "user", claims)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Path[1:]
	if page == "" {
		page = "index"
	}
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {

	}
	log.Println(userID)
	http.ServeFile(w, r, "static/"+page+".html")
}
func saveImageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user ID", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form data", http.StatusBadRequest)
		return
	}
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to get file ", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filtersJSON := r.FormValue("filters")
	if filtersJSON == "" {
		http.Error(w, "Missing filters ", http.StatusBadRequest)
		return
	}

	var filters map[string]int
	err = json.Unmarshal([]byte(filtersJSON), &filters)
	if err != nil {
		http.Error(w, "Failed to parse filters ", http.StatusBadRequest)
		return
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s", timestamp, fileHeader.Filename)

	uploadDir := "images"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, os.ModePerm)
	}
	filepath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to create file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file on server", http.StatusInternalServerError)
		return
	}

	result, err := db.Exec("INSERT INTO images (filename, filepath, user_id) VALUES (?, ?, ?)", filename, filepath, id)
	if err != nil {
		http.Error(w, "Failed to save file metadata in database", http.StatusInternalServerError)
		return
	}

	imageID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Failed to retrieve last insert ID", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO image_filters (image_id, hue, contrast, vibrance, sepia, vignette, brightness) VALUES (?, ?, ?, ?, ?, ?, ?)",
		imageID,
		filters["hue"],
		filters["contrast"],
		filters["vibrance"],
		filters["sepia"],
		filters["vignette"],
		filters["brightness"],
	)
	if err != nil {
		http.Error(w, "Failed to save filter data in database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`[{"message": "Image and filters uploaded successfully"}]`))
}

func main() {
	initDB()
	defer db.Close()

	authService := &AuthService{SecretKey: "ac17c7ae33f56f6aafc550dce6f5b98134be4d96903c8fd5538a01ec4d2bb5e6"}
	r := mux.NewRouter()

	protected := r.PathPrefix("/protected").Subrouter()
	protected.Use(authMiddleware(authService))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	protected.HandleFunc("/hello", serveHTML).Methods("GET")
	protected.HandleFunc("/editor", serveHTML).Methods("GET")
	protected.HandleFunc("/upload", saveImageHandler).Methods("POST")

	r.Handle("/images", http.FileServer(http.Dir("images")))
	r.HandleFunc("/login", serveHTML).Methods("GET")
	r.HandleFunc("/register", serveHTML).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/", serveHTML).Methods("GET")

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}
