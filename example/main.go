package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mbertschler/users"
)

var (
	port      string
	path      string
	userStore users.Store
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&port, "port", ":8001", "Port for http server")
	flag.StringVar(&path, "path", "./users.db", "Path for db file")
	flag.Parse()

	userStore = users.NewMemoryStore("/")

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/save", save)

	log.Println("Testapp for \"github.com/mbertschler/users\"")
	log.Println("Serving HTTP at " + port)
	log.Println("Saving users DB at " + path)
	log.Println("------------------------------------------")
	log.Fatal(http.ListenAndServe(port, nil))
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.Login(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"),
	)
	if err != nil {
		log.Println("Login error:", err)
		w.Write(errorPage(fmt.Sprintln("Login error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.Register(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"))
	if err != nil {
		log.Println("Register error:", err)
		w.Write(errorPage(fmt.Sprintln("Register error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	err := userStore.Logout(w, r)
	if err != nil {
		log.Println("Logout error:", err)
		w.Write(errorPage(fmt.Sprintln("Logout error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func save(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	user, err := userStore.Get(w, r)
	if err != nil {
		log.Println("Save error 1:", err)
		w.Write(errorPage(fmt.Sprintln("Save error 1:", err)))
		return
	}
	user.Data = r.PostFormValue("val")
	err = userStore.Save(user)
	if err != nil {
		log.Println("Save error 2:", err)
		w.Write(errorPage(fmt.Sprintln("Save error 2:", err)))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func index(w http.ResponseWriter, r *http.Request) {
	user, err := userStore.Get(w, r)
	if err != nil {
		if err != users.NotLoggedIn {
			log.Println("Index error:", err)
			w.Write(errorPage(fmt.Sprintln("Index error:", err)))
			return
		}
	}

	data, ok := user.Data.(string)
	if !ok {
		data = "&nbsp;"
	}

	w.Write([]byte(`
<html>
	<head>
		<style>
			body{
				font-family: sans-serif;
				font-size:16px;
				color: #333;
			}
			td {
			    padding: 4px;
			}
			th {
			    padding: 6px;
			    font-size:19px;
			    background-color:#ddd;
			}
			input, button {
				font-size:16px;
				padding:3px;
				margin-top:6px;
			}
			div {
				display: inline-block;
				width: 200px;
				vertical-align:top;
			}
		</style>
	</head>
	<body>
		<h1>Testapp for package "github.com/mbertschler/users"</h1>
		<table border="1">
			<thead>
					<th>Variable</th>
					<th>Value</th>
			</thead>
			<tbody>
				<tr>
					<td>Session ID</td>
					<td style="word-break: break-all;">` + fmt.Sprint(user.Session.ID) + `</td>
				</tr>
				<tr>
					<td>Session Expires</td>
					<td>` + fmt.Sprint(user.Session.Expires) + `</td>
				</tr>
				<tr>
					<td>Session LastCon</td>
					<td>` + fmt.Sprint(user.Session.LastCon) + `</td>
				</tr>
				<tr>
					<td>Session Bound</td>
					<td>` + fmt.Sprint(user.Session.Bound) + `</td>
				</tr>
				<tr>
					<td>Session LoggedIn</td>
					<td>` + fmt.Sprint(user.Session.LoggedIn) + `</td>
				</tr>
				<tr>
					<td>Session User</td>
					<td>` + fmt.Sprint(user.Name) + `</td>
				</tr>
			</tbody>
		</table>
		<div>
			<h2>Register</h2>
			<form action="/register" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Register</button>
			</form>
		</div>
		<div>
			<h2>Login</h2>
			<form action="/login" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Login</button>
			</form>
		</div>
		<div>
			<h2>Logout</h2>
			<form action="/logout" method="POST">
				<button type="submit">Logout</button>
			</form>
		</div>
		<div>
			<h2>Set Value</h2>
			<p>` + data + `</p>
			<form action="/save" method="POST">
				<input type="text" name="val" placeholder="Value"/> <br/>
				<button type="submit">Save</button>
			</form>
		</div>
	</body>
</html>
`))
}

func errorPage(in string) []byte {
	return []byte(`
<html>
	<head>
		<style>
			body{
				font-family: sans-serif;
				font-size:16px;
				color: #333;
			}
			td {
			    padding: 4px;
			}
			th {
			    padding: 6px;
			    font-size:19px;
			    background-color:#ddd;
			}
			input, button {
				font-size:16px;
				padding:3px;
				margin-top:6px;
			}
			div {
				display: inline-block;
				width: 200px;
				vertical-align:top;
			}
		</style>
	</head>
	<body>
		<h1>Error</h1>
		<p>` + in + `</p>
		<a href="/"><button type="submit">Back</button></a>
	</body>
</html>
`)
}
