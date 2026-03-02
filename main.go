package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
	"github.com/whyttea/lenslocked/controllers"
	"github.com/whyttea/lenslocked/migrations"
	"github.com/whyttea/lenslocked/models"
	"github.com/whyttea/lenslocked/templates"
	"github.com/whyttea/lenslocked/views"
)

// func executeTemplate(w http.ResponseWriter, filepath string) {
// 	tpl, err := views.ParseFS(filepath)
// 	if err != nil {
// 		log.Printf("parsing template: %v", err)
// 		http.Error(w, "There was an error parsing the template file.", http.StatusInternalServerError)
// 	}
// 	tpl.Execute(w, nil)
// }

// func homeHandler(w http.ResponseWriter, r *http.Request) {

// 	tplPath := filepath.Join("templates", "home.gohtml")
// 	executeTemplate(w, tplPath)

// IDparam := chi.URLParam(r, "id")
// ctx := r.Context()
// key := ctx.Value("key").(string)

// w.Write([]byte(fmt.Sprintf("New ID is %v, %v", IDparam, key)))
// response := fmt.Sprintf("<h2>%s</h2>", IDparam)
// fmt.Fprint(w, response)

// }

// type Router struct {
// }

// func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	switch r.URL.Path {
// 	case "/":
// 		homeHandler(w, r)
// 	case "/contact":
// 		contactHandler(w, r)
// 	case "/faq":
// 		faqHandler(w, r)
// 	default:
// 		// TODO: add the defualt page
// 		http.Error(w, "Page not found", http.StatusNotFound)
// 	}
// }

//	func pathHandler(w http.ResponseWriter, r *http.Request) {
//		switch r.URL.Path {
//		case "/":
//			homeHandler(w, r)
//		case "/contact":
//			contactHandler(w, r)
//		default:
//			// TODO: add the defualt page
//			http.Error(w, "Page not found", http.StatusNotFound)
//		}
//	}

type config struct {
	PSQL models.PostgresConfig
	SMTP models.SMTPConfig
	CSRF struct {
		Key    string
		Secure bool
	}
	Server struct {
		Address string
	}
}

func loanEnvConfig() (config, error) {
	var cfg config
	err := godotenv.Load()
	if err != nil {
		return cfg, err
	}
	// TODO: PSQL
	cfg.PSQL = models.DefaultPostgresConfig()
	// TODO: SMTP
	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	portStr := os.Getenv("SMTP_PORT")
	port, err := strconv.Atoi(portStr) // port is read as a string so we need to convert it to int
	if err != nil {
		return cfg, err
	}
	cfg.SMTP.Port = port
	// TODO: CSRF
	cfg.CSRF.Key = "gFvi45R4fy5xNBlnEeZtQbfAVCYEIAUX"
	cfg.CSRF.Secure = false
	// TODO: Server
	cfg.Server.Address = ":3000"

	return cfg, nil
}

func main() {
	// load env variables
	cfg, err := loanEnvConfig()
	if err != nil {
		panic(err)
	}

	// Setup db connection & migrations
	db, err := models.Open(cfg.PSQL)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to the DB")

	err = models.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	// setup required services
	userService := &models.UserService{
		DB: db,
	}
	sessionService := &models.SessionService{
		DB: db,
	}
	pwResetService := &models.PasswordResetService{
		DB: db,
	}
	emailService := models.NewEmailService(cfg.SMTP)
	galleryService := &models.GalleryService{
		DB: db,
	}

	//setup middleware
	umw := controllers.UserMiddleweare{
		SessionService: sessionService,
	}

	csrfMw := csrf.Protect(
		[]byte(cfg.CSRF.Key),
		csrf.Secure(cfg.CSRF.Secure), // requires https connection by default, but when in staging we do not have that
		csrf.Path("/"),
	)

	// setup controllers
	usersC := controllers.Users{
		UserService:          userService,
		SessionService:       sessionService,
		PasswordResetService: pwResetService,
		EmailService:         emailService,
	}
	galleriesC := controllers.Galleries{
		GalleryService: galleryService,
	}

	usersC.Templates.New = views.Must(views.ParseFS(templates.FS, "signup.gohtml", "tailwind.gohtml"))
	usersC.Templates.SignIn = views.Must(views.ParseFS(templates.FS, "signin.gohtml", "tailwind.gohtml"))
	usersC.Templates.ForgotPassword = views.Must(views.ParseFS(templates.FS, "forgot-pw.gohtml", "tailwind.gohtml"))
	usersC.Templates.CheckYourEmail = views.Must(views.ParseFS(templates.FS, "check-your-email.gohtml", "tailwind.gohtml"))
	usersC.Templates.ResetPassword = views.Must(views.ParseFS(templates.FS, "reset-pw.gohtml", "tailwind.gohtml"))
	galleriesC.Templates.New = views.Must(views.ParseFS(templates.FS, "galleries/new.gohtml", "tailwind.gohtml"))

	// setup our router and routes
	r := chi.NewRouter()
	// apply mw onto the router
	r.Use(csrfMw)
	r.Use(umw.SetUser)

	tpl := views.Must(views.ParseFS(templates.FS, "home.gohtml", "tailwind.gohtml"))
	r.Get("/", controllers.StaticHandler(tpl))
	tpl = views.Must(views.ParseFS(templates.FS, "contact.gohtml", "tailwind.gohtml"))
	r.Get("/contact", controllers.StaticHandler(tpl))
	// tpl = views.Must(views.Parse(filepath.Join("templates", "faq.gohtml")))
	tpl = views.Must(views.ParseFS(templates.FS, "faq.gohtml", "tailwind.gohtml"))

	r.Get("/faq", controllers.FAQ(tpl))
	r.Get("/signup", usersC.New)
	r.Post("/signup", usersC.Create)
	r.Post("/users", usersC.Create)
	r.Get("/signin", usersC.SignIn)
	r.Post("/signin", usersC.ProcessSignIn)
	r.Post("/signout", usersC.ProcessSignOut)
	r.Get("/forgot-pw", usersC.ForgotPassword)
	r.Post("/forgot-pw", usersC.ProcessForgotPassword)
	r.Get("/reset-pw", usersC.ResetPassword)
	r.Post("/reset-pw", usersC.ProcessResetPassword)
	// an exmaple of requiredUser mw implementation below
	r.Route("/users/me", func(r chi.Router) {
		r.Use(umw.RequireUser)
		r.Get("/", usersC.CurrentUser)
	})
	r.Route("/galleries", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(umw.RequireUser)
			r.Get("/new", galleriesC.New)
			r.Post("/", galleriesC.Create)
		})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Page not found", http.StatusNotFound)
	})
	// http.HandleFunc("/contact", contactHandler)

	// start the server
	fmt.Printf("Starting the server at %s...\n", cfg.Server.Address)
	err = http.ListenAndServe(cfg.Server.Address, r)
	if err != nil {
		panic(err)
	}

}
