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

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type Img struct {
	Id         int    `json:"id"`
	Filename   string `json:"filename"`
	Filepath   string `json:"filepath"`
	Created    string `json:"created_at"`
	User       int    `json:"user_id"`
	FilterId   int    `json:"filter_id"`
	ImageId    int    `json:"image_id"`
	Contrast   int    `json:"contrast"`
	Vibrance   int    `json:"vibrance"`
	Sepia      int    `json:"sepia"`
	Vignette   int    `json:"vignette"`
	Brightness int    `json:"brightness"`
	Saturation int    `json:"saturation"`
	Exposure   int    `json:"exposure`
	Noise      int    `json:"noise"`
	Sharpen    int    `json:"sharpen"`
}
type JsonResponse struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data,omitempty"` // Optional field for extra response data
}

func JSONResponse(w http.ResponseWriter, message string, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := JsonResponse{
		Message: message,
		Status:  status,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

var db *sql.DB

func initDB() {
	var err error
	// db, err = sql.Open("mysql", "root:root@tcp(host.docker.internal:3306)/img_editor")
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
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			fmt.Fprintf(w, `<div>User already exists</div>`)
			return
		}
		JSONResponse(w, "Could not create user", http.StatusInternalServerError, nil)

		log.Println(err)
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
		fmt.Fprintf(w, `<div>Invalid username or password</div>`)
		return
	}

	if !checkPasswordHash(password, hashedPassword) {
		fmt.Fprintf(w, `<div>Invalid username or password</div>`)
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
	_, err := getUserIDFromContext(r.Context())
	if err != nil {

	}
	http.ServeFile(w, r, "static/"+page+".html")
}

func saveImageHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user ID", http.StatusInternalServerError)
		return
	}
	//checknut ci uz prekrocil 3 obrazky
	var imageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM images WHERE user_id = ?", id).Scan(&imageCount)
	if err != nil {
		http.Error(w, "Failed to check image count", http.StatusInternalServerError)
		return
	}

	if imageCount >= 3 {
		JSONResponse(w, "User has exceeded the maximum allowed images", http.StatusOK, nil)
		// w.Header().Set("Content-Type", "application/json")
		// w.Write([]byte(`{"message": "User has exceeded the maximum allowed images"}`))
		log.Println("yes")
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
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	filtersJSON := r.FormValue("filters")
	if filtersJSON == "" {
		http.Error(w, "Missing filters", http.StatusBadRequest)
		return
	}

	var filters map[string]int
	err = json.Unmarshal([]byte(filtersJSON), &filters)
	if err != nil {
		http.Error(w, "Failed to parse filters", http.StatusBadRequest)
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

	_, err = db.Exec("INSERT INTO image_filters (image_id, contrast, vibrance, sepia, vignette, brightness, saturation, exposure, noise, sharpen) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		imageID,
		filters["contrast"],
		filters["vibrance"],
		filters["sepia"],
		filters["vignette"],
		filters["brightness"],
		filters["saturation"],
		filters["exposure"],
		filters["noise"],
		filters["sharpen"],
	)
	if err != nil {
		http.Error(w, "Failed to save filter data in database", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, "Image and filters uploaded successfully", http.StatusCreated, map[string]int64{"imgId": imageID})
}
func updateFiltersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}

	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user ID", http.StatusInternalServerError)
		return
	}
	imageId := r.FormValue("imageId")
	if imageId == "" {
		http.Error(w, "Missing Id", http.StatusBadRequest)
		return
	}

	var filepath string
	//Overit uzivatela
	err = db.QueryRow("SELECT filepath FROM images WHERE id = ? AND user_id = ?", imageId, userID).Scan(&filepath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Image not found or you are not authorized to delete this image", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve image data", http.StatusInternalServerError)
		}

	}
	//ziskat filtre z requestu
	filtersJSON := r.FormValue("filters")
	if filtersJSON == "" {
		http.Error(w, "Missing filters", http.StatusBadRequest)
		return
	}

	var filters map[string]int
	err = json.Unmarshal([]byte(filtersJSON), &filters)
	if err != nil {
		http.Error(w, "Failed to parse filters", http.StatusBadRequest)
		return
	}
	//updatnut zaznam
	_, err = db.Exec("update image_filters SET , contrast = ?, vibrance = ?, sepia = ?, vignette = ?, brightness = ?, saturation = ?, exposure = ?, noise = ?, sharpen = ? where image_id = ?;",
		filters["contrast"],
		filters["vibrance"],
		filters["sepia"],
		filters["vignette"],
		filters["brightness"],
		filters["saturation"],
		filters["exposure"],
		filters["noise"],
		filters["sharpen"],
		imageId,
	)
	if err != nil {
		http.Error(w, "Failed to update data", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, "Filters Updated", http.StatusOK, nil)
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// w.Write([]byte(`{"message": "Filters Updated"}`))

}

func getImages(w http.ResponseWriter, r *http.Request) {
	id, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user id", http.StatusInternalServerError)
		return
	}

	res, err := db.Query("select * from images join image_filters f on images.id = f.image_id WHERE user_id = ?", id)
	if err != nil {
		http.Error(w, "Failed to retrieve images", http.StatusInternalServerError)
		return
	}
	defer res.Close()

	var images []Img
	for res.Next() {
		var img Img
		err = res.Scan(&img.Id, &img.Filename, &img.Filepath, &img.Created, &img.User, &img.FilterId,
			&img.ImageId, &img.Contrast, &img.Vibrance, &img.Sepia, &img.Vignette, &img.Brightness,
			&img.Saturation, &img.Exposure, &img.Noise, &img.Sharpen)
		if err != nil {
			http.Error(w, "Error scanning database result", http.StatusInternalServerError)
			return
		}
		images = append(images, img)
	}

	if len(images) == 0 {
		JSONResponse(w, "No images found", http.StatusOK, nil)
		return
	}

	JSONResponse(w, "Images retrieved", http.StatusOK, images)

}
func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
	vars := mux.Vars(r)
	imageID := vars["id"]

	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user ID", http.StatusInternalServerError)
		return
	}

	var filepath string
	err = db.QueryRow("SELECT filepath FROM images WHERE id = ? AND user_id = ?", imageID, userID).Scan(&filepath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Image not found or you are not authorized to delete this image", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve image data", http.StatusInternalServerError)
		}
		return
	}

	err = os.Remove(filepath)
	if err != nil {
		http.Error(w, "Failed to delete image file", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM images WHERE id = ? AND user_id = ?", imageID, userID)
	if err != nil {
		http.Error(w, "Failed to delete image metadata from the database", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM image_filters WHERE image_id = ?", imageID)
	if err != nil {
		http.Error(w, "Failed to delete image filters from the database", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, "Image deleted successfully", http.StatusOK, nil)

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
	protected.HandleFunc("/getimages", getImages).Methods("POST")
	protected.HandleFunc("/delete/{id}", deleteImageHandler).Methods("DELETE")
	protected.HandleFunc("/update", updateFiltersHandler).Methods("POST")

	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir("images"))))
	r.HandleFunc("/login", serveHTML).Methods("GET")
	r.HandleFunc("/register", serveHTML).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/", serveHTML).Methods("GET")

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}
