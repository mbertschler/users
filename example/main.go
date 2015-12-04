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

func index(w http.ResponseWriter, r *http.Request) {
	sess, ok, err := userStore.GetSession(r)
	if err != nil {
		log.Println(err)
		return
	}
	if !ok {
		err = userStore.SaveSession(w, sess)
		if err != nil {
			log.Println(err)
		}
	}
	user, err := userStore.GetUser(sess)
	//store := userStore.(*users.MemoryStore)
	//body := ""
	// for name := range store.users {
	// 	body += `<tr>
	// 			<td>` + name + `</td>
	// 		</tr>`
	// }
	// sessionsTable := ""
	// usersTable := `<table border="1">
	// 	<thead>
	// 			<th>Users</th>
	// 	</thead>
	// 	<tbody>
	// 		` + body + `
	// 	</tbody>
	// </table>`
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
					<td>Session found</td>
					<td>` + fmt.Sprint(ok) + `</td>
				</tr>
				<tr>
					<td>Session ID</td>
					<td style="word-break: break-all;">` + fmt.Sprint(sess.ID) + `</td>
				</tr>
				<tr>
					<td>Session Expires</td>
					<td>` + fmt.Sprint(sess.Expires) + `</td>
				</tr>
				<tr>
					<td>Session LastCon</td>
					<td>` + fmt.Sprint(sess.LastCon) + `</td>
				</tr>
				<tr>
					<td>Session Bound</td>
					<td>` + fmt.Sprint(sess.Bound) + `</td>
				</tr>
				<tr>
					<td>Session LoggedIn</td>
					<td>` + fmt.Sprint(sess.LoggedIn) + `</td>
				</tr>
				<tr>
					<td>Session User</td>
					<td>` + fmt.Sprint(sess.User) + `</td>
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
			<p>` + "Val" + `</p>
			<form action="/save" method="POST">
				<input type="text" name="val" placeholder="Value"/> <br/>
				<button type="submit">Save</button>
			</form>
		</div>
		` + /*usersTable +*/ `
	</body>
</html>
`))
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	sess, _, err := userStore.GetSession(r)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = userStore.Login(sess,
		r.PostFormValue("user"),
		r.PostFormValue("pass"))
	if err != nil {
		log.Println("Login error:", err)
	}
	err = userStore.SaveSession(w, sess)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	sess, _, err := userStore.GetSession(r)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = userStore.Register(sess,
		r.PostFormValue("user"),
		r.PostFormValue("pass"))
	if err != nil {
		log.Println("Register error:", err)
	}
	err = userStore.SaveSession(w, sess)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	sess, _, err := userStore.GetSession(r)
	if err != nil {
		log.Println(err)
		return
	}
	err = userStore.Logout(sess)
	if err != nil {
		log.Println("Logout error:", err)
	}
	err = userStore.SaveSession(w, sess)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func save(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
