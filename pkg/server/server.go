package server

import (
	"encoding/gob"
	"fmt"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"unicode"
)

const sessionName = "session"
const sessionUserValue = "user"
const sessionMessageValue = "message"



func sessionKey () []byte {
	env := os.Getenv("SESSION_KEY")
	if env == "" {
		res := RandSeq(64)
		log.Printf("using random session key %s", res)

		return []byte(res)
	} else {
		return []byte(env)
	}
}

func dbLocation () string {
	env := os.Getenv("DB_LOCATION")
	if env == "" {
		res := "store.db"
		log.Printf("using %s as db location", res)

		return res
	} else {
		return env
	}
}

func baseUrl () string {
	env := os.Getenv("BASE_URL")
	if env == "" {
		panic("no base url set (set BASE_URL)")
	} else {
		return env
	}
}
var BaseUrl = baseUrl()

func IsUrl(str string) bool {
	u, err := url2.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func StartServer() error {
	r := mux.NewRouter()
	r.Use(middleware.Logger)

	gob.Register(SessionUser{})

	sessionStore := sessions.NewCookieStore(sessionKey())

	funcMap := template.FuncMap{
		"url": func(s string) template.URL {
			return template.URL(s)
		},
	}
	index, err := template.New("index.gohtml").
		Funcs(funcMap).
		ParseFiles("static/index.gohtml")
	if err != nil {
		return err
	}

	store, err := NewStore(dbLocation())
	if err != nil {
		return err
	}
	defer store.Close()

	lm, err := NewLoginManager(store)
	if err != nil {
		return err
	}

	r.HandleFunc("/__API__/logout", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		session.Values[sessionUserValue] = nil
		err = sessionStore.Save(r, w, session)
		if err != nil {
			log.Printf("%v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).Methods("POST")

	r.HandleFunc("/__API__/login", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		err = r.ParseForm()
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("bad request", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		su, err := lm.LogIn(User{
			Name:     username,
			Password: []byte(password),
		})

		if err != nil {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		session.Values[sessionUserValue] = su

		err = sessionStore.Save(r, w, session)
		if err != nil {
			log.Printf("%v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).Methods("POST")

	r.HandleFunc("/__API__/changepw", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		err = r.ParseForm()
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("bad request", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		password := r.FormValue("password")
		passwordRepeat := r.FormValue("password-repeat")

		if passwordRepeat != password {
			session.AddFlash("passwords don't match", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		fmt.Printf("%v", session.Values)

		if session.Values[sessionUserValue] == nil {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		su, ok := session.Values[sessionUserValue].(SessionUser)
		if !ok {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, err := lm.LoggedIn(su)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		err = lm.ChangePassword(*user, password)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		session.Values[sessionUserValue] = nil
		_ = sessionStore.Save(r, w, session)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}).Methods("POST")

	r.HandleFunc("/__API__/rmalias", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		if session.Values[sessionUserValue] == nil {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		su, ok := session.Values[sessionUserValue].(SessionUser)
		if !ok {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, err := lm.LoggedIn(su)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		aliasName := string(body)

		alias, err := store.GetAlias(aliasName)
		if err != nil {
			fmt.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if alias.Owner != user.Name && !user.Admin {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		err = store.RmAlias(alias)
		if err != nil {
			fmt.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}



		_ = sessionStore.Save(r, w, session)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}).Methods("POST")

	r.HandleFunc("/__API__/createalias", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		err = r.ParseForm()
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("bad request", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if session.Values[sessionUserValue] == nil {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		su, ok := session.Values[sessionUserValue].(SessionUser)
		if !ok {
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, err := lm.LoggedIn(su)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("unauthorized", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		url := r.FormValue("url")
		alias := r.FormValue("alias")

		existingAlias, err := store.GetAlias(alias)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if existingAlias != nil {
			session.AddFlash("alias name exists", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if !IsUrl(url) {
			session.AddFlash("not a valid url", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if alias == "__API__" {
			session.AddFlash("can't use __API__ as alias (used internally)", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if !IsValidAlias(alias) {
			log.Printf("%v", err)
			session.AddFlash("not a valid url", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		err = store.CreateAlias(Alias{
			Owner: user.Name,
			Url:   url,
			Alias: alias,
		})
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		_ = sessionStore.Save(r, w, session)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}).Methods("POST")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		var user *User
		if session.Values[sessionUserValue] != nil {
			su, ok := session.Values[sessionUserValue].(SessionUser)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			user, err = lm.LoggedIn(su)
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		messagesI := session.Flashes("message")
		messages := make([]string, len(messagesI))
		for index, elem := range messagesI {
			messages[index] = elem.(string)
		}

		randomAlias, err := NonExistentRandom(store)
		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var aliases []Alias
		if user != nil {
			aliases, err = store.GetUserAliases(user)
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		err = index.Execute(w, struct {
			User *User
			Messages []string
			NonExistentRandom string
			Aliases []Alias
			BaseUrl string
		}{
			user,
			messages,
			randomAlias,
			aliases,
			BaseUrl,
		})
		if err != nil {
			log.Printf("%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_ = sessionStore.Save(r, w, session)
	}).Methods("GET")

	r.HandleFunc("/{alias}", func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, sessionName)
		if err != nil {
			log.Printf("%v", err)
			// continue, we may not be able to get it, but we can set it
		}

		params := mux.Vars(r)
		aliasName := params["alias"]

		alias, err := store.GetAlias(aliasName)
		if err != nil {
			log.Printf("%v", err)
			session.AddFlash("server error", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if alias == nil {
			session.AddFlash("alias could not be found. Did you make a typo?", sessionMessageValue)
			_ = sessionStore.Save(r, w, session)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}


		http.Redirect(w, r, alias.Url, http.StatusPermanentRedirect)
	}).Methods("GET")

	url := "0.0.0.0:3000"
	log.Printf("listening on %s", url)
	return http.ListenAndServe(url, r)
}

func IsValidAlias(alias string) bool {
	for _, c := range alias {
		switch {
		case unicode.IsLetter(c):
			continue
		case unicode.IsNumber(c):
			continue
		case c == '-':
			continue
		case c == '_':
			continue
		default:
			return false
		}
	}

	return true
}

