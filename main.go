package main

//Importy
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
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// Vlastné dátové typy
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
	Exposure   int    `json:"exposure"`
	Noise      int    `json:"noise"`
	Sharpen    int    `json:"sharpen"`
}
type JsonResponse struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data,omitempty"`
}
type AuthService struct {
	SecretKey string
}

// Funkcia posiela JSON Response na frontend
func JSONResponse(w http.ResponseWriter, message string, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := JsonResponse{
		Message: message,
		Status:  status,
		Data:    data,
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Println("Error encoding JSON response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	// log.Println(response)
}

// Inicializuje pripojenie na databázu
var db *sql.DB

func initDB() {
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Could not load .env file, using system environment variables")
		} else {
			log.Println("Loaded .env file")
		}
	}
	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("MYSQL_DATABASE")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbName)
	var err error

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Could not create database connection: %v", err)
	}

	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			fmt.Println("Connected to database")
			return
		}
		log.Println("Retrying database connection...", err)
		time.Sleep(5 * time.Second)
	}

	log.Fatalf("❌ Could not connect to database after 10 attempts: %v", err)
}

// Hashuje heslo
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Porovná heslá
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Overí správnosť JWT tokenu
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

// Spracúva registráciu
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !usernameRegex.MatchString(username) {
		fmt.Fprintf(w, "<div>Username cannot contain special characters</div>")
		http.Error(w, "Invalid username. Only letters and numbers are allowed.", http.StatusBadRequest)
		return
	}

	if len(password) < 8 {
		fmt.Fprintf(w, "<div>Password must be at least 8 characters</div>")
		http.Error(w, "Password must be at least 8 characters long.", http.StatusBadRequest)
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
			fmt.Fprintf(w, "<div>User already exists</div>")
			return
		}
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Add("Hx-Redirect", "/login")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

// Spracúva login
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
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Could not load .env file, using system environment variables")
		} else {
			log.Println("Loaded .env file")
		}
	}
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_TOKEN_STRING")))
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

	w.Header().Add("Hx-Redirect", "/protected/editor")

}

// Spracúva logout
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

// Zistí User id z JWT tokenu
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

// Autorizuje užívateľa na základe Cookies a skontroluje JWT token
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

// Redirectuje užívateľa pre "/"
func indexHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if cookie != nil {
		http.Redirect(w, r, "/protected/editor", http.StatusSeeOther)
		return
	}
	log.Println(err)
	http.Redirect(w, r, "/login", http.StatusSeeOther)

}

// Užívateľovi zobrazí dané HTML súbory
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

// Spracúva obrázky z frontendu overuje ich počet, ukladá ich na server aj do databázy spoločne s filtrami
func saveImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	id, err := getUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unable to get user ID", http.StatusInternalServerError)
		return
	}
	//checknut ci uz prekrocil 3 obrazky
	var imageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM images WHERE user_id = ?", id).Scan(&imageCount)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to check image count", http.StatusInternalServerError)
		return
	}

	if imageCount >= 3 {
		JSONResponse(w, "User has exceeded the maximum allowed images", http.StatusOK, nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)

	err = r.ParseMultipartForm(20 << 20)
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

// Updatuje tabuľku filters
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
	err = db.QueryRow("SELECT filepath FROM images WHERE id = ? AND user_id = ?", imageId, userID).Scan(&filepath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Image not found or you are not authorized to update this image", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve image data", http.StatusInternalServerError)
		}

	}
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
	_, err = db.Exec("UPDATE image_filters SET contrast = ?, vibrance = ?, sepia = ?, vignette = ?, brightness = ?, saturation = ?, exposure = ?, noise = ?, sharpen = ? where image_id = ?;",
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
}

// Získa cesty ku obrázkom z databázy
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

// Maže obrázky z databázy a zo servera
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
	//Vytvorenie overenia
	authService := &AuthService{SecretKey: "ac17c7ae33f56f6aafc550dce6f5b98134be4d96903c8fd5538a01ec4d2bb5e6"}
	r := mux.NewRouter()
	//Definuje chránené cesty
	protected := r.PathPrefix("/protected").Subrouter()
	protected.Use(authMiddleware(authService))
	protected.HandleFunc("/hello", serveHTML).Methods("GET")
	protected.HandleFunc("/editor", serveHTML).Methods("GET")
	protected.HandleFunc("/upload", saveImageHandler).Methods("POST")
	protected.HandleFunc("/getimages", getImages).Methods("POST")
	protected.HandleFunc("/delete/{id}", deleteImageHandler).Methods("DELETE")
	protected.HandleFunc("/update", updateFiltersHandler).Methods("POST")

	//Ostatné cesty
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir("images"))))
	r.HandleFunc("/login", serveHTML).Methods("GET")
	r.HandleFunc("/register", serveHTML).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/", indexHandler).Methods("GET")

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))

}
